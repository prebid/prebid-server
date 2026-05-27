package tmp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

func newPool() *sync.Pool {
	return &sync.Pool{New: func() any { return sha256.New() }}
}

func TestContextCacheKey_StableAndDistinct(t *testing.T) {
	pool := newPool()
	br := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/x"},
		User: &openrtb2.User{Ext: []byte(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R1"}]}]}`)},
	}
	a := contextCacheKey(pool, "rid_A", "place_1", br)
	b := contextCacheKey(pool, "rid_A", "place_1", br)
	require.Equal(t, a, b, "same inputs → same key")

	c := contextCacheKey(pool, "rid_B", "place_1", br)
	require.NotEqual(t, a, c, "different property_rid → different key")

	d := contextCacheKey(pool, "rid_A", "place_2", br)
	require.NotEqual(t, a, d, "different placement_id → different key")

	br2 := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/x"},
		User: &openrtb2.User{Ext: []byte(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R2"}]}]}`)},
	}
	e := contextCacheKey(pool, "rid_A", "place_1", br2)
	require.NotEqual(t, a, e, "different user identifier → different key")
}

func TestIdentityCacheKey_StableAndDistinct(t *testing.T) {
	pool := newPool()
	idents := []IdentityToken{{UIDType: "liveramp.com", UserToken: "R1"}}
	a := identityCacheKey(pool, "https://us", "US", idents)
	b := identityCacheKey(pool, "https://us", "US", idents)
	require.Equal(t, a, b)

	c := identityCacheKey(pool, "https://other", "US", idents)
	require.NotEqual(t, a, c)
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name           string
		contextOffers  []Offer
		identityElig   []string
		want           []string
	}{
		{
			name:          "both empty",
			contextOffers: nil,
			identityElig:  nil,
			want:          []string{},
		},
		{
			name:          "context empty",
			contextOffers: nil,
			identityElig:  []string{"pkg1"},
			want:          []string{},
		},
		{
			name:          "identity empty",
			contextOffers: []Offer{{PackageID: "pkg1"}},
			identityElig:  nil,
			want:          []string{},
		},
		{
			name:          "full overlap",
			contextOffers: []Offer{{PackageID: "pkg1"}, {PackageID: "pkg2"}},
			identityElig:  []string{"pkg2", "pkg1"},
			want:          []string{"pkg1", "pkg2"}, // order follows contextOffers
		},
		{
			name:          "partial overlap",
			contextOffers: []Offer{{PackageID: "pkg1"}, {PackageID: "pkg2"}, {PackageID: "pkg3"}},
			identityElig:  []string{"pkg2"},
			want:          []string{"pkg2"},
		},
		{
			name:          "no overlap",
			contextOffers: []Offer{{PackageID: "pkg1"}},
			identityElig:  []string{"pkg2"},
			want:          []string{},
		},
		{
			name:          "dedupe within context offers",
			contextOffers: []Offer{{PackageID: "pkg1"}, {PackageID: "pkg1"}, {PackageID: "pkg2"}},
			identityElig:  []string{"pkg1", "pkg2"},
			want:          []string{"pkg1", "pkg2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := intersect(tc.contextOffers, tc.identityElig)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestAsyncRequest_LifecycleNoFetch(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	ar := newAsyncRequest(parent)
	require.NotNil(t, ar)
	require.NotNil(t, ar.ctx)
	require.NotNil(t, ar.cancel)

	// No fetch was called; Done channel should be nil.
	require.Nil(t, ar.done)

	ar.cancel()
}

func TestFetchContext_HappyPath(t *testing.T) {
	want := ContextMatchResponse{
		Type:      TypeContextMatchResponse,
		RequestID: "req-x",
		Offers:    []Offer{{PackageID: "pkg_abc"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/tmp/context", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	req := ContextMatchRequest{
		Type:        TypeContextMatchRequest,
		RequestID:   "req-x",
		PropertyRID: "rid",
		PlacementID: "pl",
	}
	got, err := fetchContext(context.Background(), &http.Client{}, srv.URL, "", &req)
	require.NoError(t, err)
	require.Equal(t, want.RequestID, got.RequestID)
	require.Equal(t, "pkg_abc", got.Offers[0].PackageID)
}

func TestFetchContext_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := fetchContext(context.Background(), &http.Client{}, srv.URL, "", &ContextMatchRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "400")
}

func TestFetchIdentity_HappyPath(t *testing.T) {
	want := IdentityMatchResponse{
		Type:               TypeIdentityMatchResponse,
		RequestID:          "id-y",
		EligiblePackageIDs: []string{"pkg_abc"},
		Tmpx:               "k1.xyz",
		TTLSec:             60,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/tmp/identity", r.URL.Path)

		var body IdentityMatchRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotEmpty(t, body.RequestID)
		require.Equal(t, "auth-token", r.Header.Get("x-scope3-auth"))
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	req := IdentityMatchRequest{Type: TypeIdentityMatchRequest, RequestID: "id-y", SellerAgentURL: "https://us"}
	got, err := fetchIdentity(context.Background(), &http.Client{}, srv.URL, "auth-token", &req)
	require.NoError(t, err)
	require.Equal(t, want.RequestID, got.RequestID)
	require.Equal(t, "k1.xyz", got.Tmpx)
}

func TestFetchAsync_MultiImpThreePlacements_HappyPath(t *testing.T) {
	var ctxCalls, idCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tmp/context":
			ctxCalls.Add(1)
			var req ContextMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(ContextMatchResponse{
				Type:      TypeContextMatchResponse,
				RequestID: req.RequestID,
				Offers:    []Offer{{PackageID: "pkg_" + req.PlacementID}},
				Signals:   Signals{TargetingKVs: []KeyValuePair{{Key: "k_" + req.PlacementID, Value: "v"}}},
			})
		case "/tmp/identity":
			idCalls.Add(1)
			var req IdentityMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(IdentityMatchResponse{
				Type:               TypeIdentityMatchResponse,
				RequestID:          req.RequestID,
				EligiblePackageIDs: []string{"pkg_header_728x90", "pkg_preroll_video"},
				Tmpx:               "k1.token",
			})
		}
	}))
	defer srv.Close()

	mod, err := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	require.NoError(t, err)
	m := mod.(*Module)

	br := &openrtb2.BidRequest{
		ID: "auction-1",
		Imp: []openrtb2.Imp{
			{ID: "imp1", TagID: "header"},
			{ID: "imp2", TagID: "sidebar"},
			{ID: "imp3", TagID: "video"},
		},
		Site:   &openrtb2.Site{Domain: "example.com", Page: "https://example.com/x"},
		User:   &openrtb2.User{Ext: json.RawMessage(`{"eids":[{"source":"liveramp.com","uids":[{"id":"R1"}]}]}`)},
		Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
	}
	accountCfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"header":"header_728x90","sidebar":"sidebar_300x250","video":"preroll_video"}}}}`)

	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, accountCfg, json.RawMessage(`{}`))
	<-ar.done

	require.NoError(t, ar.err)
	require.NotNil(t, ar.result)
	require.Equal(t, int32(3), ctxCalls.Load(), "one context call per unique placement")
	require.Equal(t, int32(1), idCalls.Load(), "exactly one identity call regardless of imp count")

	require.Equal(t, "k1.token", ar.result.TMPX)
	require.Equal(t, []string{"pkg_header_728x90"}, ar.result.PerPlacement["header_728x90"].EligiblePackages)
	require.Equal(t, []string{"pkg_preroll_video"}, ar.result.PerPlacement["preroll_video"].EligiblePackages)
	require.Empty(t, ar.result.PerPlacement["sidebar_300x250"].EligiblePackages, "sidebar pkg not in identity eligible set")

	require.Equal(t, "header_728x90", ar.result.ImpToPlacement["imp1"])
}

func TestFetchAsync_SharedPlacementDeduped(t *testing.T) {
	var ctxCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tmp/context":
			ctxCalls.Add(1)
			var req ContextMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(ContextMatchResponse{Type: TypeContextMatchResponse, RequestID: req.RequestID, Offers: []Offer{{PackageID: "pkg_shared"}}})
		case "/tmp/identity":
			var req IdentityMatchRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(IdentityMatchResponse{Type: TypeIdentityMatchResponse, RequestID: req.RequestID, EligiblePackageIDs: []string{"pkg_shared"}})
		}
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)
	br := &openrtb2.BidRequest{
		ID:  "a",
		Imp: []openrtb2.Imp{{ID: "i1", TagID: "h"}, {ID: "i2", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
	}
	cfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"shared"}}}}`)
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, cfg, nil)
	<-ar.done
	require.Equal(t, int32(1), ctxCalls.Load(), "shared placement dedupes to one context call")
}

func TestFetchAsync_PartialFailure_P1Strict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tmp/identity" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		var req ContextMatchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(ContextMatchResponse{Type: TypeContextMatchResponse, RequestID: req.RequestID, Offers: []Offer{{PackageID: "pkg_a"}}})
	}))
	defer srv.Close()

	mod, _ := Builder(json.RawMessage(`{"router_url":"`+srv.URL+`","seller_agent_url":"https://us","masking":{"enabled":false}}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	m := mod.(*Module)
	br := &openrtb2.BidRequest{
		ID:  "a",
		Imp: []openrtb2.Imp{{ID: "i1", TagID: "h"}},
		Site: &openrtb2.Site{Domain: "x.com"},
	}
	cfg := json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"p"}}}}`)
	ar := newAsyncRequest(context.Background())
	ar.module = m
	ar.fetchAsync(br, cfg, nil)
	<-ar.done

	require.Error(t, ar.err, "P1 strict: identity failure means whole fetch is errored")
	require.Nil(t, ar.result)
}
