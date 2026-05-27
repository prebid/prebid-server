package tmp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/require"
)

func TestCountryAlpha3ToAlpha2(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"USA", "US"},
		{"GBR", "GB"},
		{"DEU", "DE"},
		{"FRA", "FR"},
		{"JPN", "JP"},
		{"CAN", "CA"},
		{"AUS", "AU"},
		{"BRA", "BR"},
		{"IND", "IN"},
		{"CHN", "CN"},
		{"unknown", ""},
		{"", ""},
		{"US", ""},  // already alpha-2 — function only accepts alpha-3
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, countryAlpha3ToAlpha2(tc.in))
		})
	}
}

func TestExtractIdentities_RespectsOrderAndCap(t *testing.T) {
	tests := []struct {
		name         string
		preserveEids []string
		userExtJSON  string
		userID       string
		want         []IdentityToken
	}{
		{
			name:         "no user — empty",
			preserveEids: []string{"liveramp.com", "uidapi.com", "id5-sync.com"},
			userExtJSON:  ``,
			want:         nil,
		},
		{
			name:         "single liveramp eid",
			preserveEids: []string{"liveramp.com", "uidapi.com", "id5-sync.com"},
			userExtJSON:  `{"eids":[{"source":"liveramp.com","uids":[{"id":"RID-123"}]}]}`,
			want:         []IdentityToken{{UIDType: "liveramp.com", UserToken: "RID-123"}},
		},
		{
			name:         "all three preferred sources in order",
			preserveEids: []string{"liveramp.com", "uidapi.com", "id5-sync.com"},
			userExtJSON: `{"eids":[
				{"source":"id5-sync.com","uids":[{"id":"ID5-X"}]},
				{"source":"liveramp.com","uids":[{"id":"RID-1"}]},
				{"source":"uidapi.com","uids":[{"id":"UID-2"}]}
			]}`,
			want: []IdentityToken{
				{UIDType: "liveramp.com", UserToken: "RID-1"},
				{UIDType: "uidapi.com", UserToken: "UID-2"},
				{UIDType: "id5-sync.com", UserToken: "ID5-X"},
			},
		},
		{
			name:         "non-preferred source ignored",
			preserveEids: []string{"liveramp.com"},
			userExtJSON:  `{"eids":[{"source":"criteo.com","uids":[{"id":"X"}]}]}`,
			want:         nil,
		},
		{
			name:         "fallback to user.id when no eids and ext doesn't carry one",
			preserveEids: []string{"liveramp.com"},
			userExtJSON:  ``,
			userID:       "pub-uid-9",
			want:         []IdentityToken{{UIDType: "publisher_user_id", UserToken: "pub-uid-9"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var user *openrtb2.User
			if tc.userExtJSON != "" || tc.userID != "" {
				user = &openrtb2.User{ID: tc.userID}
				if tc.userExtJSON != "" {
					user.Ext = json.RawMessage(tc.userExtJSON)
				}
			}
			got := extractIdentities(user, tc.preserveEids)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMaskBidRequest_StripsSensitiveFields(t *testing.T) {
	cfg := MaskingConfig{
		Enabled: true,
		Geo:     GeoMaskingConfig{PreserveMetro: true, PreserveZip: true, PreserveCity: false, LatLongPrecision: 2},
		User:    UserMaskingConfig{PreserveEids: []string{"liveramp.com"}},
		Device:  DeviceMaskingConfig{PreserveMobileIds: false},
	}
	lat := 40.7128
	lon := -74.0059
	br := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP:    "73.158.22.41",
			IPv6:  "2001:db8::1",
			UA:    "ua",
			OS:    "macOS",
			IFA:   "A1B2-C3D4",
			Geo:   &openrtb2.Geo{Country: "USA", Region: "NY", City: "NYC", Metro: "501", ZIP: "10001", Lat: &lat, Lon: &lon, Accuracy: 5},
		},
		User: &openrtb2.User{
			ID:       "uid",
			BuyerUID: "bid",
			Yob:      1985,
			Gender:   "M",
			Keywords: "kw",
			Ext: []byte(`{"eids":[
				{"source":"liveramp.com","uids":[{"id":"keep"}]},
				{"source":"criteo.com","uids":[{"id":"drop"}]}
			]}`),
		},
	}

	masked := maskBidRequest(br, cfg)
	require.NotNil(t, masked)

	require.Empty(t, masked.Device.IP)
	require.Empty(t, masked.Device.IPv6)
	require.Empty(t, masked.Device.IFA)
	require.NotEmpty(t, masked.Device.UA)
	require.NotEmpty(t, masked.Device.OS)

	require.Equal(t, "USA", masked.Device.Geo.Country)
	require.Equal(t, "NY", masked.Device.Geo.Region)
	require.Empty(t, masked.Device.Geo.City)
	require.Equal(t, "501", masked.Device.Geo.Metro)
	require.Equal(t, "10001", masked.Device.Geo.ZIP)
	require.InDelta(t, 40.71, *masked.Device.Geo.Lat, 0.001)
	require.InDelta(t, -74.01, *masked.Device.Geo.Lon, 0.001)
	require.Zero(t, masked.Device.Geo.Accuracy)

	require.Empty(t, masked.User.ID)
	require.Empty(t, masked.User.BuyerUID)
	require.Zero(t, masked.User.Yob)
	require.Empty(t, masked.User.Gender)
	require.Empty(t, masked.User.Keywords)
}

func TestMaskBidRequest_DisabledIsPassthrough(t *testing.T) {
	cfg := MaskingConfig{Enabled: false}
	br := &openrtb2.BidRequest{Device: &openrtb2.Device{IP: "73.158.22.41"}}
	masked := maskBidRequest(br, cfg)
	require.Equal(t, "73.158.22.41", masked.Device.IP)
}

func TestFilterEIDs_AllAllowed(t *testing.T) {
	eids := []openrtb2.EID{
		{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "R1"}}},
		{Source: "uidapi.com", UIDs: []openrtb2.UID{{ID: "U2"}}},
	}
	allow := []string{"liveramp.com", "uidapi.com"}
	got := filterEIDs(eids, allow)
	require.Len(t, got, 2)
	require.Equal(t, "liveramp.com", got[0].Source)
	require.Equal(t, "uidapi.com", got[1].Source)
}

func TestFilterEIDs_PartialAllowed(t *testing.T) {
	eids := []openrtb2.EID{
		{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "R1"}}},
		{Source: "criteo.com", UIDs: []openrtb2.UID{{ID: "C1"}}},
		{Source: "uidapi.com", UIDs: []openrtb2.UID{{ID: "U2"}}},
	}
	allow := []string{"liveramp.com", "uidapi.com"}
	got := filterEIDs(eids, allow)
	require.Len(t, got, 2)
	require.Equal(t, "liveramp.com", got[0].Source)
	require.Equal(t, "uidapi.com", got[1].Source)
}

func TestFilterEIDs_EmptyAllow(t *testing.T) {
	eids := []openrtb2.EID{
		{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "R1"}}},
	}
	got := filterEIDs(eids, []string{})
	require.Nil(t, got)
}

func TestTruncateLatLong_PositiveValues(t *testing.T) {
	tests := []struct {
		value     float64
		precision int
		want      float64
	}{
		{40.7128, 2, 40.71},
		{-74.0059, 2, -74.01},
		{12.3456, 1, 12.3},
		{12.3456, 3, 12.346},
		{12.3456, 4, 12.3456},
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			got := truncateLatLong(tc.value, tc.precision)
			require.InDelta(t, tc.want, got, 0.00001)
		})
	}
}

func TestTruncateLatLong_EdgeCases(t *testing.T) {
	require.Equal(t, 0.0, truncateLatLong(40.7128, 0))
	require.Equal(t, 0.0, truncateLatLong(40.7128, -1))
}

func TestExtractIdentities_MalformedJSON(t *testing.T) {
	// Malformed JSON in user.Ext should not panic; should treat as no EIDs
	user := &openrtb2.User{
		Ext: []byte(`{invalid json}`),
	}
	got := extractIdentities(user, []string{"liveramp.com"})
	require.Nil(t, got)
}

func TestExtractIdentities_EmptyEIDList(t *testing.T) {
	// EIDs exists but is empty
	user := &openrtb2.User{
		ID:  "pub-123",
		Ext: []byte(`{"eids":[]}`),
	}
	got := extractIdentities(user, []string{"liveramp.com"})
	require.Equal(t, []IdentityToken{{UIDType: "publisher_user_id", UserToken: "pub-123"}}, got)
}

func TestExtractIdentities_DuplicateSources(t *testing.T) {
	// Same source twice in eids; should use first
	user := &openrtb2.User{
		Ext: []byte(`{"eids":[
			{"source":"liveramp.com","uids":[{"id":"FIRST"}]},
			{"source":"liveramp.com","uids":[{"id":"SECOND"}]}
		]}`),
	}
	got := extractIdentities(user, []string{"liveramp.com"})
	require.Len(t, got, 1)
	require.Equal(t, "FIRST", got[0].UserToken)
}

func TestMaskDevice_PreserveMobileIds(t *testing.T) {
	br := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IFA:       "A1B2-C3D4",
			DPIDMD5:   "abc123",
			MACMD5:    "def456",
			DIDMD5:    "ghi789",
		},
	}
	// With PreserveMobileIds=true, these should remain
	cfg := DeviceMaskingConfig{PreserveMobileIds: true}
	maskDevice(br, cfg)
	require.Equal(t, "A1B2-C3D4", br.Device.IFA)
	require.Equal(t, "abc123", br.Device.DPIDMD5)
	require.Equal(t, "def456", br.Device.MACMD5)
	require.Equal(t, "ghi789", br.Device.DIDMD5)
}

func TestMaskGeo_NoDevice(t *testing.T) {
	br := &openrtb2.BidRequest{} // No device
	cfg := GeoMaskingConfig{PreserveMetro: true}
	maskGeo(br, cfg)
	// Should not panic; device is nil
	require.Nil(t, br.Device)
}

func TestMaskGeo_NoGeo(t *testing.T) {
	br := &openrtb2.BidRequest{
		Device: &openrtb2.Device{}, // No geo
	}
	cfg := GeoMaskingConfig{PreserveMetro: true}
	maskGeo(br, cfg)
	// Should not panic; geo is nil
	require.Nil(t, br.Device.Geo)
}

func TestMaskUser_NilUser(t *testing.T) {
	br := &openrtb2.BidRequest{} // No user
	cfg := UserMaskingConfig{PreserveEids: []string{}}
	maskUser(br, cfg)
	// Should not panic
	require.Nil(t, br.User)
}
