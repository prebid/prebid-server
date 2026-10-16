package identity

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
)

func eid(source string, ids ...string) openrtb2.EID {
	uids := make([]openrtb2.UID, 0, len(ids))
	for _, id := range ids {
		uids = append(uids, openrtb2.UID{ID: id})
	}
	return openrtb2.EID{Source: source, UIDs: uids}
}

func lmt(v int8) *int8 { return &v }

func TestCandidateKeys(t *testing.T) {
	tests := []struct {
		name    string
		maxKeys int
		req     *openrtb2.BidRequest
		want    []cache.Key
	}{
		{
			name: "nil request",
			req:  nil,
			want: nil,
		},
		{
			name:    "priority order iiq, pubcid, maid, other, device",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: []openrtb2.EID{
					eid("intentiq.com", "iiqid"),
					eid("pubcid.org", "pub1"),
					eid("uidapi.com", "uid2"),
				}},
				Device: &openrtb2.Device{IFA: "ifa-1", IP: "1.2.3.4"}, // no UA: composite is ifa_ip
			},
			want: []cache.Key{
				{Key: "iiq:iiqid", Type: cache.ThirdParty},
				{Key: "pubcid:pub1", Type: cache.FirstParty},
				{Key: "maid:ifa-1", Type: cache.FirstParty},
				{Key: "uidapi.com:uid2", Type: cache.FirstParty},
				{Key: "dev:ifa-1_1.2.3.4", Type: cache.Device},
			},
		},
		{
			name:    "sharedid maps to pubcid namespace",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: []openrtb2.EID{eid("sharedid.org", "s1")}},
			},
			want: []cache.Key{{Key: "pubcid:s1", Type: cache.FirstParty}},
		},
		{
			name:    "lmt=1 suppresses maid but keeps device composite off ifa? no - composite still uses ifa",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IFA: "ifa-x", Lmt: lmt(1)},
			},
			// maid key skipped; device composite still built from ifa (matches Java addDeviceComposite,
			// which does not consult lmt).
			want: []cache.Key{{Key: "dev:ifa-x", Type: cache.Device}},
		},
		{
			name:    "CTV devicetype uppercases maid ifa",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IFA: "abc-DEF", DeviceType: 3},
			},
			want: []cache.Key{
				{Key: "maid:ABC-DEF", Type: cache.FirstParty},
				{Key: "dev:abc-DEF", Type: cache.Device},
			},
		},
		{
			name:    "blank uids skipped",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: []openrtb2.EID{eid("intentiq.com", "  ", "real")}},
			},
			want: []cache.Key{{Key: "iiq:real", Type: cache.ThirdParty}},
		},
		{
			name:    "dedup keeps first occurrence",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: []openrtb2.EID{
					eid("uidapi.com", "dup"),
					eid("uidapi.com", "dup"),
				}},
			},
			want: []cache.Key{{Key: "uidapi.com:dup", Type: cache.FirstParty}},
		},
		{
			name:    "maxKeys cap",
			maxKeys: 2,
			req: &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: []openrtb2.EID{
					eid("intentiq.com", "a"),
					eid("pubcid.org", "b"),
					eid("uidapi.com", "c"),
				}},
			},
			want: []cache.Key{
				{Key: "iiq:a", Type: cache.ThirdParty},
				{Key: "pubcid:b", Type: cache.FirstParty},
			},
		},
		{
			name:    "device composite ipv6 fallback when ip blank",
			maxKeys: 10,
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IPv6: "::1"},
			},
			want: []cache.Key{{Key: "dev:::1", Type: cache.Device}},
		},
		{
			name:    "no ids yields no keys",
			maxKeys: 10,
			req:     &openrtb2.BidRequest{},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewFirstPartyKeyExtractor(tt.maxKeys)
			got := e.CandidateKeys(tt.req)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestCandidateKeysDeviceCompositeUsesNormalizedUA verifies the dev: composite embeds the normalized
// UA (not the raw string), computed via normalizeUA so the assertion never couples to the parser's
// exact output.
func TestCandidateKeysDeviceCompositeUsesNormalizedUA(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IFA: "ifa", UA: ua, IP: "9.9.9.9"}}

	devKey := "dev:ifa_" + normalizeUA(ua) + "_9.9.9.9"
	got := NewFirstPartyKeyExtractor(10).CandidateKeys(req)

	assert.Equal(t, []cache.Key{
		{Key: "maid:ifa", Type: cache.FirstParty},
		{Key: devKey, Type: cache.Device},
	}, got)
	assert.NotContains(t, devKey, "Mozilla", "composite must use normalized UA, not the raw string")
}
