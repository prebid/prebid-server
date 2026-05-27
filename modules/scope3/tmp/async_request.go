package tmp

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/logger"
)

// contextCacheKey derives a stable hex string from inputs that scope a Context
// Match result. Same (property_rid, placement_id, page/app) returns the same
// key. User identity is intentionally excluded: Context Match is
// user-identity-free by spec, and multiple users on the same page share one
// cache entry.
func contextCacheKey(pool *sync.Pool, propertyRID, placementID string, br *openrtb2.BidRequest) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("p:" + propertyRID))
	_, _ = h.Write([]byte("|pl:" + placementID))
	writeSiteOrApp(h, br)
	return hex.EncodeToString(h.Sum(nil))
}

// identityCacheKey derives a stable hex string from inputs that scope an
// Identity Match result. Identity match results are page-context-free, so the
// key intentionally excludes site/app/placement.
func identityCacheKey(pool *sync.Pool, sellerAgentURL, country string, idents []IdentityToken) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("s:" + sellerAgentURL))
	_, _ = h.Write([]byte("|c:" + country))
	for _, t := range idents {
		_, _ = h.Write([]byte("|id:" + t.UIDType + "=" + t.UserToken))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeSiteOrApp(h hash.Hash, br *openrtb2.BidRequest) {
	if br.Site != nil {
		_, _ = h.Write([]byte("|d:" + br.Site.Domain))
		if br.Site.Page != "" {
			_, _ = h.Write([]byte("|pg:" + br.Site.Page))
		}
	}
	if br.App != nil {
		_, _ = h.Write([]byte("|a:" + br.App.Bundle))
	}
}

func writePrivacySafeUserIDs(h hash.Hash, user *openrtb2.User) {
	if user == nil {
		return
	}
	var ext struct {
		EIDs []openrtb2.EID `json:"eids"`
	}
	if len(user.Ext) > 0 {
		_ = json.Unmarshal(user.Ext, &ext)
	}
	for _, eid := range ext.EIDs {
		if len(eid.UIDs) > 0 {
			_, _ = h.Write([]byte("|e:" + eid.Source + "=" + eid.UIDs[0].ID))
		}
	}
}

// intersect returns the package IDs that appear in both the context offers
// list and the identity-eligible list. Order follows the contextOffers; output
// is deduplicated. Returns an empty (non-nil) slice when either input is empty.
func intersect(contextOffers []Offer, identityEligible []string) []string {
	out := []string{}
	if len(contextOffers) == 0 || len(identityEligible) == 0 {
		return out
	}
	eligible := make(map[string]struct{}, len(identityEligible))
	for _, id := range identityEligible {
		eligible[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(contextOffers))
	for _, offer := range contextOffers {
		if _, alreadyEmitted := seen[offer.PackageID]; alreadyEmitted {
			continue
		}
		if _, ok := eligible[offer.PackageID]; ok {
			out = append(out, offer.PackageID)
			seen[offer.PackageID] = struct{}{}
		}
	}
	return out
}

// AsyncResult is the data the auction response hook reads after fan-out.
type AsyncResult struct {
	PerPlacement   map[string]PlacementResult // placement_id → result
	ImpToPlacement map[string]string          // imp.id → placement_id
	TMPX           string
}

// PlacementResult holds the per-placement enrichment that ends up on
// each bid whose impid maps to this placement.
type PlacementResult struct {
	EligiblePackages []string
	TargetingKVs     []KeyValuePair
	Segments         []string
}

// AsyncRequest is per-auction state created in HandleEntrypointHook and
// drained in HandleAuctionResponseHook.
type AsyncRequest struct {
	module *Module
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	result *AsyncResult
	err    error
}

// newAsyncRequest creates per-auction state. Done is nil until fetchAsync
// runs — the auction-response hook must check for nil before reading.
func newAsyncRequest(parent context.Context) *AsyncRequest {
	ctx, cancel := context.WithCancel(parent)
	return &AsyncRequest{ctx: ctx, cancel: cancel}
}

func fetchContext(ctx context.Context, client *http.Client, routerURL, authKey string, req *ContextMatchRequest) (*ContextMatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode context request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, routerURL+"/tmp/context", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authKey != "" {
		httpReq.Header.Set("x-scope3-auth", authKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("context match returned status %d: %s", resp.StatusCode, string(body))
	}

	var out ContextMatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode context response: %w", err)
	}
	return &out, nil
}

func fetchIdentity(ctx context.Context, client *http.Client, routerURL, authKey string, req *IdentityMatchRequest) (*IdentityMatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode identity request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, routerURL+"/tmp/identity", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authKey != "" {
		httpReq.Header.Set("x-scope3-auth", authKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("identity match returned status %d: %s", resp.StatusCode, string(body))
	}

	var out IdentityMatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode identity response: %w", err)
	}
	return &out, nil
}

// fetchAsync runs the full N+1 fan-out in a goroutine. The Done channel is
// closed when the result (or error) is ready. Callers should:
//   - wait on <-ar.done (or <-ar.ctx.Done() for graceful timeout)
//   - read ar.result OR ar.err
//   - call ar.cancel() to release the context.
func (ar *AsyncRequest) fetchAsync(br *openrtb2.BidRequest, accountCfg json.RawMessage, requestExt json.RawMessage) {
	ar.done = make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ar.err = fmt.Errorf("panic in fetchAsync: %v", r)
				logger.Errorf("scope3.tmp: panic in fetchAsync for auction %s: %v", br.ID, r)
			}
			close(ar.done)
		}()
		ar.run(br, accountCfg, requestExt)
	}()
}

func (ar *AsyncRequest) run(br *openrtb2.BidRequest, accountCfg, requestExt json.RawMessage) {
	resolver := accountResolver{accountConfig: accountCfg, requestExt: requestExt, moduleCfg: ar.module.cfg}
	ids, err := resolver.resolveAuction()
	if err != nil {
		logger.Warnf("scope3.tmp: skipping enrichment for auction %s: %v", br.ID, err)
		ar.err = err
		return
	}

	// Resolve per-imp placements; dedupe.
	impToPlacement := make(map[string]string, len(br.Imp))
	uniquePlacements := []string{}
	seenPlacement := map[string]struct{}{}
	for _, imp := range br.Imp {
		place, ok := resolver.resolvePlacement(imp.TagID)
		if !ok || place == "" {
			continue
		}
		impToPlacement[imp.ID] = place
		if _, dup := seenPlacement[place]; !dup {
			seenPlacement[place] = struct{}{}
			uniquePlacements = append(uniquePlacements, place)
		}
	}
	if len(uniquePlacements) == 0 {
		logger.Warnf("scope3.tmp: no placements resolved for any imp in auction %s (property_rid=%s)", br.ID, ids.PropertyRID)
		ar.err = errors.New("no placements resolved for any imp")
		return
	}

	masked := br
	if ar.module.cfg.Masking.Enabled {
		masked = maskBidRequest(br, ar.module.cfg.Masking)
		if masked == nil {
			logger.Errorf("scope3.tmp: masking failed for auction %s (property_rid=%s); refusing to send unmasked", br.ID, ids.PropertyRID)
			ar.err = errors.New("masking failed; refusing to send unmasked request")
			return
		}
	}

	identities := extractIdentities(masked.User, ar.module.cfg.Masking.User.PreserveEids)
	country := ""
	if masked.Device != nil && masked.Device.Geo != nil {
		country = countryAlpha3ToAlpha2(masked.Device.Geo.Country)
	}

	// Fan out: N context calls + 1 identity call. Collect results via mutex-
	// protected maps / pointer. Errors are collected into a buffered channel
	// (capacity = N+1) so goroutines never block on send.
	//
	// gctx is a child context shared by all goroutines. cancelFanout is called
	// as soon as any goroutine fails so that in-flight HTTP calls are aborted
	// promptly via their request context.
	gctx, cancelFanout := context.WithCancel(ar.ctx)
	defer cancelFanout()

	contextResults := make(map[string]*ContextMatchResponse, len(uniquePlacements))
	var contextMu sync.Mutex
	var identityResp *IdentityMatchResponse

	// Check cache for context results before spawning goroutines. Cache hits are
	// written directly (no mutex needed; goroutines haven't started yet).
	placementsToFetch := make([]string, 0, len(uniquePlacements))
	for _, placement := range uniquePlacements {
		cacheKey := contextCacheKey(ar.module.sha256Pool, ids.PropertyRID, placement, masked)
		if cached, err := ar.module.cache.Get([]byte(cacheKey)); err == nil {
			var resp ContextMatchResponse
			if json.Unmarshal(cached, &resp) == nil {
				contextResults[placement] = &resp
				continue
			}
		}
		placementsToFetch = append(placementsToFetch, placement)
	}

	// Check cache for identity result.
	identityCacheKey := identityCacheKey(ar.module.sha256Pool, ids.SellerAgentURL, country, identities)
	identityCached := false
	if cached, err := ar.module.cache.Get([]byte(identityCacheKey)); err == nil {
		var resp IdentityMatchResponse
		if json.Unmarshal(cached, &resp) == nil {
			identityResp = &resp
			identityCached = true
		}
	}

	total := len(placementsToFetch)
	if !identityCached {
		total++
	}
	errc := make(chan error, total)

	var wg sync.WaitGroup
	wg.Add(total)

	for _, placement := range placementsToFetch {
		placement := placement
		go func() {
			defer wg.Done()
			req := &ContextMatchRequest{
				Type:         TypeContextMatchRequest,
				RequestID:    mustUUID(),
				PropertyRID:  ids.PropertyRID,
				PropertyType: ids.PropertyType,
				PlacementID:  placement,
			}
			if masked.Site != nil && masked.Site.Page != "" {
				req.ArtifactRefs = []ArtifactRef{{Type: "url", Value: masked.Site.Page}}
			}
			resp, err := fetchContext(gctx, ar.module.httpClient, ids.RouterURL, ar.module.cfg.AuthKey, req)
			if err != nil {
				logger.Errorf("scope3.tmp: context call failed for auction %s placement=%s request_id=%s: %v", br.ID, placement, req.RequestID, err)
				errc <- fmt.Errorf("context placement=%s: %w", placement, err)
				cancelFanout()
				return
			}
			// Write to cache only when server indicates caching is desired.
			if resp.CacheTTL != 0 {
				ttl := ar.module.cfg.CacheTTLSeconds
				if resp.CacheTTL > 0 && resp.CacheTTL < ttl {
					ttl = resp.CacheTTL
				}
				if data, merr := json.Marshal(resp); merr == nil {
					cacheKey := contextCacheKey(ar.module.sha256Pool, ids.PropertyRID, placement, masked)
					_ = ar.module.cache.Set([]byte(cacheKey), data, ttl)
				}
			}
			contextMu.Lock()
			contextResults[placement] = resp
			contextMu.Unlock()
		}()
	}

	if !identityCached {
		go func() {
			defer wg.Done()
			req := &IdentityMatchRequest{
				Type:           TypeIdentityMatchRequest,
				RequestID:      mustUUID(),
				SellerAgentURL: ids.SellerAgentURL,
				Identities:     identities,
				Country:        country,
			}
			resp, err := fetchIdentity(gctx, ar.module.httpClient, ids.RouterURL, ar.module.cfg.AuthKey, req)
			if err != nil {
				logger.Errorf("scope3.tmp: identity call failed for auction %s request_id=%s: %v", br.ID, req.RequestID, err)
				errc <- fmt.Errorf("identity: %w", err)
				cancelFanout()
				return
			}
			// Write to cache only when server indicates caching is desired.
			if resp.TTLSec != 0 {
				ttl := ar.module.cfg.CacheTTLSeconds
				if resp.TTLSec > 0 && resp.TTLSec < ttl {
					ttl = resp.TTLSec
				}
				if data, merr := json.Marshal(resp); merr == nil {
					_ = ar.module.cache.Set([]byte(identityCacheKey), data, ttl)
				}
			}
			identityResp = resp
		}()
	}

	wg.Wait()
	close(errc)

	// P1 strict: any failure means no partial result.
	for err := range errc {
		if err != nil {
			ar.err = err
			return
		}
	}

	perPlacement := make(map[string]PlacementResult, len(contextResults))
	identityElig := []string{}
	if identityResp != nil {
		identityElig = identityResp.EligiblePackageIDs
	}
	for placement, ctxResp := range contextResults {
		perPlacement[placement] = PlacementResult{
			EligiblePackages: intersect(ctxResp.Offers, identityElig),
			TargetingKVs:     ctxResp.Signals.TargetingKVs,
			Segments:         ctxResp.Signals.Segments,
		}
	}

	tmpx := ""
	if identityResp != nil {
		tmpx = identityResp.Tmpx
	}
	ar.result = &AsyncResult{
		PerPlacement:   perPlacement,
		ImpToPlacement: impToPlacement,
		TMPX:           tmpx,
	}
}

// mustUUID returns a new random UUID string. Returns "" on the extremely rare
// failure (entropy exhaustion); downstream handles empty request_id gracefully.
func mustUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return ""
	}
	return u.String()
}
