package tmp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
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
