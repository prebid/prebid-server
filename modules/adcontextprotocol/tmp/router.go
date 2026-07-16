package tmp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/prebid-server/v4/logger"
)

// providerResult holds one provider's contribution after both endpoints
// have been called (whichever were configured).
type providerResult struct {
	Name string
	// Context is set when the context call succeeded, nil otherwise.
	Context *tmproto.ContextMatchResponse
	// Identity is set when the identity call succeeded, nil otherwise.
	Identity *tmproto.IdentityMatchResponse
	// IdentityAttempted is true when the module actually issued an
	// identity call for this provider (URL configured AND tokens
	// present). Lets the merge distinguish "identity errored" (fail
	// closed) from "identity not applicable" (offers pass).
	IdentityAttempted bool
	Errs              []error
}

// routerResult is the joined view across all providers.
type routerResult struct {
	Providers []providerResult
	// Segments are the flat targeting strings the response hook writes into
	// bid ext. Each string is "key=value" so consumers can split on the
	// separator downstream. Post-cap.
	Segments []string
	// ErrCount is the number of providers that produced at least one
	// error. Surfaced via analytics so a silent-failure regression is
	// visible in dashboards.
	ErrCount int
}

// fanOut executes the module's TMP flow for a single auction against a
// pre-derived tmpInputs snapshot. Caller must have already run
// deriveInputs synchronously — this function does not touch the
// BidRequest, so it is safe to run in a background goroutine while the
// auction continues to mutate the request wrapper.
func (m *Module) fanOut(ctx context.Context, inputs tmpInputs) *routerResult {
	// Domain / bundle → property_rid.
	lookupKey := inputs.Domain
	if lookupKey == "" {
		lookupKey = inputs.Bundle
	}
	if lookupKey == "" || inputs.PlacementID == "" {
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

	// Masking is applied at input derivation time (deriveInputs already
	// respected the masking config) so nothing here needs to re-filter.

	results := make([]providerResult, len(m.cfg.Providers))
	var wg sync.WaitGroup

	for i, p := range m.cfg.Providers {
		wg.Add(1)
		go func(i int, p ProviderConfig) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("adcontextprotocol.tmp: panic in provider %s fan-out: %v", p.Name, r)
					results[i] = providerResult{Name: p.Name, Errs: []error{fmt.Errorf("panic: %v", r)}}
				}
			}()
			results[i] = m.callProvider(ctx, p, inputs, prop, propertyType)
		}(i, p)
	}
	wg.Wait()

	errCount := 0
	for _, r := range results {
		if len(r.Errs) > 0 {
			errCount++
		}
	}

	return &routerResult{
		Providers: results,
		Segments:  m.mergeSegments(results),
		ErrCount:  errCount,
	}
}

// callProvider builds fresh per-provider request objects (so request_ids
// do not correlate across providers) and issues the configured calls.
func (m *Module) callProvider(
	ctx context.Context,
	p ProviderConfig,
	inputs tmpInputs,
	prop *PropertyRecord,
	propertyType tmproto.PropertyType,
) providerResult {
	res := providerResult{Name: p.Name}

	timeout := time.Duration(p.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Duration(m.cfg.TimeoutMs) * time.Millisecond
	}
	pCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Per-provider request IDs — two colluding providers should not be
	// able to join on identical ids for the same auction. TMP §514/555
	// requires context and identity ids not correlate; this goes further
	// and gives each provider its own pair.
	ctxRequestID, err := newRequestID()
	if err != nil {
		res.Errs = append(res.Errs, fmt.Errorf("request id: %w", err))
		return res
	}
	ctxReq := &tmproto.ContextMatchRequest{
		Type:           "context_match_request",
		RequestID:      ctxRequestID,
		PropertyRID:    prop.PropertyRID,
		PropertyID:     prop.PropertyID,
		PropertyType:   propertyType,
		PlacementID:    inputs.PlacementID,
		SellerAgentURL: m.cfg.SellerAgentURL,
		Geo:            inputs.Geo,
		ArtifactRefs:   inputs.ArtifactRefs,
	}

	var idReq *tmproto.IdentityMatchRequest
	if p.IdentityURL != "" && len(inputs.Identities) > 0 {
		idRequestID, err := newRequestID()
		if err != nil {
			res.Errs = append(res.Errs, fmt.Errorf("request id: %w", err))
			return res
		}
		idReq = &tmproto.IdentityMatchRequest{
			Type:           "identity_match_request",
			RequestID:      idRequestID,
			SellerAgentURL: m.cfg.SellerAgentURL,
			Identities:     inputs.Identities,
			Consent:        inputs.Consent,
			Country:        inputs.Country,
		}
		res.IdentityAttempted = true
	}

	// Context and identity fire in parallel per provider so a slow
	// endpoint on one side does not starve the other. Order is
	// randomized per request and the second call is optionally jittered
	// so a passive observer cannot rely on stable timing to pair the two.
	var innerWG sync.WaitGroup
	var mu sync.Mutex

	var calls []func()
	if p.ContextURL != "" {
		calls = append(calls, func() {
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					res.Errs = append(res.Errs, fmt.Errorf("panic in context call: %v", r))
					mu.Unlock()
					logger.Errorf("adcontextprotocol.tmp: panic in context call to %s: %v", p.Name, r)
				}
			}()
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
	if idReq != nil {
		calls = append(calls, func() {
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					res.Errs = append(res.Errs, fmt.Errorf("panic in identity call: %v", r))
					mu.Unlock()
					logger.Errorf("adcontextprotocol.tmp: panic in identity call to %s: %v", p.Name, r)
				}
			}()
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

	rand.Shuffle(len(calls), func(a, b int) { calls[a], calls[b] = calls[b], calls[a] })
	maxDelay := m.cfg.DecorrelationMaxDelayMs
	for idx, call := range calls {
		innerWG.Go(func() {
			if idx > 0 && maxDelay > 0 {
				delay := time.Duration(rand.IntN(maxDelay+1)) * time.Millisecond
				select {
				case <-time.After(delay):
				case <-pCtx.Done():
					return
				}
			}
			call()
		})
	}
	innerWG.Wait()
	return res
}

// mergeSegments joins each provider's context offers with its identity
// eligibility and flattens the survivors into "key=value" strings suitable
// for prebid targeting. The emitted segments cover four surfaces the AdCP
// TMP spec calls out (see adcp docs/trusted-match/specification.mdx and
// adcp-go tmproto/types_gen.go):
//
//  1. Matched package IDs → cfg.PackageTargetingKey, comma-joined and
//     deduplicated across providers. Empty PackageTargetingKey disables
//     this line entirely.
//  2. ContextMatchResponse.Signals → raw keys (last-wins on collision
//     across providers, with an emitted warn segment recording the loser).
//  3. Offer.Macros (per-offer creative macros) → raw keys.
//  4. IdentityMatchResponse.TmpxMacros[] → each TmpxMacro's own Name as
//     the key, Value verbatim. Names are provider-namespaced upstream in
//     the provider's registered tmpx_macros list; no transformation here.
//
// Capped at cfg.MaxSegments and per-value length so a hostile provider
// cannot bloat the bid response.
//
// Fail-closed on identity error: when a provider was asked to do identity
// gating (URL configured + tokens present) and the call did not return a
// response, we drop all its offers AND its TMPX macros. A hostile-or-flaky
// identity endpoint therefore cannot convert identity-gated packages into
// unconditionally-served packages, and cannot inject a TMPX token into the
// bid response by failing partway through.
func (m *Module) mergeSegments(results []providerResult) []string {
	maxLen := m.cfg.MaxSegmentValueLen
	maxCount := m.cfg.MaxSegments
	pkgKey := m.cfg.PackageTargetingKey

	var out []string
	appendSeg := func(s string) bool {
		if len(out) >= maxCount {
			return false
		}
		out = append(out, boundedSegment(s, maxLen))
		return true
	}

	// Signal-key tracking so we can log collisions across providers.
	// The producing provider's name is not exposed in the segment value,
	// only in the module's own log line — targeting keys stay clean.
	signalOwner := map[string]string{}

	// Package IDs: collect from every eligible offer, dedup, comma-join.
	// Order preserved by first-emission so tests are stable.
	var pkgIDs []string
	seenPkg := map[string]bool{}

	for _, r := range results {
		if r.Context == nil {
			continue
		}
		// Fail closed on identity-attempted-but-errored: eligibility
		// cannot be established, so no offers pass.
		if r.IdentityAttempted && r.Identity == nil {
			continue
		}
		eligible := eligibilitySet(r.Identity)
		filterEligibility := r.Identity != nil

		for _, offer := range r.Context.Offers {
			if filterEligibility && !eligible[offer.PackageID] {
				continue
			}
			if !seenPkg[offer.PackageID] {
				seenPkg[offer.PackageID] = true
				pkgIDs = append(pkgIDs, offer.PackageID)
			}
			// Per-offer creative macros. Only surfaced for eligible offers so
			// a provider cannot leak macros for packages the identity gate
			// filtered out.
			for k, v := range offer.Macros {
				if v == "" {
					continue
				}
				if prev, dup := signalOwner[k]; dup && prev != r.Name {
					logger.Warnf("adcontextprotocol.tmp: offer macro key %q from %q overwrites earlier value from %q", k, r.Name, prev)
				}
				signalOwner[k] = r.Name
				if !appendSeg(k + "=" + v) {
					return out
				}
			}
		}

		for k, v := range r.Context.Signals {
			str, ok := stringifySignal(v)
			if !ok {
				continue
			}
			if prev, dup := signalOwner[k]; dup && prev != r.Name {
				logger.Warnf("adcontextprotocol.tmp: context signal key %q from %q overwrites earlier value from %q", k, r.Name, prev)
			}
			signalOwner[k] = r.Name
			if !appendSeg(k + "=" + str) {
				return out
			}
		}

		// Identity TMPX macros. Names are already provider-namespaced by the
		// provider's registered tmpx_macros list (see adcp-go
		// tmproto/types_gen.go ProviderRegistration.TmpxMacros), so no key
		// transformation here — pass Name=Value verbatim to the ad server.
		if r.Identity != nil {
			for _, tm := range r.Identity.TmpxMacros {
				if tm.Name == "" || tm.Value == "" {
					continue
				}
				if !appendSeg(tm.Name + "=" + tm.Value) {
					return out
				}
			}
		}
	}

	if pkgKey != "" && len(pkgIDs) > 0 {
		if !appendSeg(pkgKey + "=" + strings.Join(pkgIDs, ",")) {
			return out
		}
	}
	return out
}

// stringifySignal accepts scalar signal values (string, bool, number)
// and rejects non-scalars — a map or slice from a hostile provider
// would flow into targeting as "map[…]" garbage otherwise.
func stringifySignal(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		return x, true
	case bool:
		return fmt.Sprintf("%t", x), true
	case float64, float32, int, int64, int32:
		return fmt.Sprintf("%v", x), true
	}
	return "", false
}

// boundedSegment truncates the segment string to the configured cap so
// a hostile provider cannot make single segments arbitrarily large.
func boundedSegment(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
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
