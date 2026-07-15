package tmp

import (
	"context"
	"sync"
	"time"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/logger"
)

// providerResult holds one provider's contribution after both endpoints have
// been called (whichever were configured). Nil fields mean "not configured" or
// "call failed" — callers should treat both the same way when merging.
type providerResult struct {
	Name     string
	Context  *tmproto.ContextMatchResponse
	Identity *tmproto.IdentityMatchResponse
	Errs     []error
}

// routerResult is the joined view across all providers.
type routerResult struct {
	Providers []providerResult
	// Segments are the flat targeting strings the response hook writes into
	// bid ext. Each string is "key=value" so consumers can split on the
	// separator downstream.
	Segments []string
}

// fanOut executes the module's TMP flow for a single auction: adapt the bid
// request, resolve the property, then call every configured provider's
// context and identity endpoints in parallel. Returns quickly if the property
// cannot be resolved — the auction proceeds without TMP signals.
func (m *Module) fanOut(ctx context.Context, req *openrtb2.BidRequest) *routerResult {
	inputs := deriveInputs(&m.cfg, req)

	// Domain / bundle → property_rid.
	lookupKey := inputs.Domain
	if lookupKey == "" {
		lookupKey = inputs.Bundle
	}
	if lookupKey == "" {
		return &routerResult{}
	}

	prop, ok, err := m.registry.Resolve(ctx, lookupKey)
	if err != nil {
		logger.Warnf("adcontextprotocol.tmp: property registry lookup for %q failed: %v", lookupKey, err)
		return &routerResult{}
	}
	if !ok || prop == nil || prop.PropertyRID == "" {
		return &routerResult{}
	}
	propertyType := prop.PropertyType
	if propertyType == "" {
		propertyType = inputs.PropertyType
	}

	// Apply masking before we let the ContextMatchRequest leave the process.
	if m.cfg.Masking.Enabled {
		maskedGeo := m.maskGeoMap(inputs.Geo)
		if maskedGeo != nil {
			inputs.Geo = maskedGeo
		}
		inputs.Identities = m.filterIdentities(inputs.Identities)
	}

	ctxReq := &tmproto.ContextMatchRequest{
		Type:           "context_match_request",
		RequestID:      newRequestID(),
		PropertyRID:    prop.PropertyRID,
		PropertyID:     prop.PropertyID,
		PropertyType:   propertyType,
		PlacementID:    inputs.PlacementID,
		SellerAgentURL: m.cfg.SellerAgentURL,
		Geo:            inputs.Geo,
		ArtifactRefs:   inputs.ArtifactRefs,
	}

	// Identity request stays absent when the auction has no usable tokens.
	var idReq *tmproto.IdentityMatchRequest
	if len(inputs.Identities) > 0 {
		idReq = &tmproto.IdentityMatchRequest{
			Type:           "identity_match_request",
			RequestID:      newRequestID(),
			SellerAgentURL: m.cfg.SellerAgentURL,
			Identities:     inputs.Identities,
			Consent:        inputs.Consent,
			Country:        inputs.Country,
		}
	}

	results := make([]providerResult, len(m.cfg.Providers))
	var wg sync.WaitGroup

	for i, p := range m.cfg.Providers {
		wg.Add(1)
		go func(i int, p ProviderConfig) {
			defer wg.Done()
			res := providerResult{Name: p.Name}

			// Per-provider deadline; falls back to the module-level timeout.
			timeout := time.Duration(p.TimeoutMs) * time.Millisecond
			if timeout <= 0 {
				timeout = time.Duration(m.cfg.TimeoutMs) * time.Millisecond
			}
			pCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Context and identity fire in parallel per provider so a slow
			// endpoint on one side does not starve the other.
			var innerWG sync.WaitGroup
			var mu sync.Mutex

			if p.ContextURL != "" {
				innerWG.Go(func() {
					resp, err := m.callContext(pCtx, p, ctxReq)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						res.Errs = append(res.Errs, err)
					} else {
						res.Context = resp
					}
				})
			}
			if p.IdentityURL != "" && idReq != nil {
				innerWG.Go(func() {
					resp, err := m.callIdentity(pCtx, p, idReq)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						res.Errs = append(res.Errs, err)
					} else {
						res.Identity = resp
					}
				})
			}
			innerWG.Wait()
			results[i] = res
		}(i, p)
	}
	wg.Wait()

	return &routerResult{
		Providers: results,
		Segments:  mergeSegments(results),
	}
}

// mergeSegments joins each provider's context offers with its identity
// eligibility and flattens the survivors into "key=value" strings suitable
// for prebid targeting. Response-level signals from the context response are
// passed through as targeting keys directly.
func mergeSegments(results []providerResult) []string {
	var out []string
	for _, r := range results {
		if r.Context == nil {
			continue
		}
		eligible := eligibilitySet(r.Identity)
		filterEligibility := r.Identity != nil

		for _, offer := range r.Context.Offers {
			if filterEligibility {
				if !eligible[offer.PackageID] {
					continue
				}
			}
			out = append(out, r.Name+"_package="+offer.PackageID)
		}

		for k, v := range r.Context.Signals {
			s, ok := v.(string)
			if !ok {
				continue
			}
			out = append(out, r.Name+"_"+k+"="+s)
		}
	}
	return out
}

func eligibilitySet(idResp *tmproto.IdentityMatchResponse) map[string]bool {
	if idResp == nil {
		return nil
	}
	set := make(map[string]bool, len(idResp.EligiblePackageIDs))
	for _, id := range idResp.EligiblePackageIDs {
		set[id] = true
	}
	return set
}
