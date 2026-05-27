package tmp

import (
	"crypto/sha256"
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
