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
