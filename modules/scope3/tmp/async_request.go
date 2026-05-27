package tmp

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"sync"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// contextCacheKey derives a stable hex string from inputs that scope a Context
// Match result. Same (property_rid, placement_id, page/app, privacy-safe ids)
// returns the same key.
func contextCacheKey(pool *sync.Pool, propertyRID, placementID string, br *openrtb2.BidRequest) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("p:" + propertyRID))
	_, _ = h.Write([]byte("|pl:" + placementID))
	writeSiteOrApp(h, br)
	writePrivacySafeUserIDs(h, br.User)
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
