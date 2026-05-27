package tmp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveAuctionIdentifiers(t *testing.T) {
	moduleCfg := Config{RouterURL: "https://router", SellerAgentURL: "https://us"}

	tests := []struct {
		name        string
		accountJSON string
		extJSON     string
		impTagID    string
		wantRID     string
		wantPType   PropertyType
		wantPlace   string
		wantSeller  string
		wantErr     string
	}{
		{
			name:        "all from account",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"01916f3a","property_type":"website","placements":{"header":"header_728x90"}}}}`,
			extJSON:     `{}`,
			impTagID:    "header",
			wantRID:     "01916f3a",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "header_728x90",
			wantSeller:  "https://us",
		},
		{
			name:        "ext overrides property_rid",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"acct","property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"override"}}}}}`,
			impTagID:    "h",
			wantRID:     "override",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "h1",
			wantSeller:  "https://us",
		},
		{
			name:        "ext placement_id overrides per-imp lookup",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{"prebid":{"modules":{"scope3":{"tmp":{"placement_id":"test_slot"}}}}}`,
			impTagID:    "h",
			wantRID:     "r",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "test_slot",
			wantSeller:  "https://us",
		},
		{
			name:        "account overrides seller_agent_url",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"h1"},"seller_agent_url":"https://alt"}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantRID:     "r",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "h1",
			wantSeller:  "https://alt",
		},
		{
			name:        "missing property_rid is error",
			accountJSON: `{"scope3":{"tmp":{"property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantErr:     "property_rid is required",
		},
		{
			name:        "missing property_type is error",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantErr:     "property_type is required",
		},
		{
			name:        "unknown tagid yields empty placement_id (caller decides to skip)",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"website","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "unknown_tagid",
			wantRID:     "r",
			wantPType:   PropertyTypeWebsite,
			wantPlace:   "",
			wantSeller:  "https://us",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := accountResolver{
				accountConfig: json.RawMessage(tc.accountJSON),
				requestExt:    json.RawMessage(tc.extJSON),
				moduleCfg:     moduleCfg,
			}
			ids, err := r.resolveAuction()
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantRID, ids.PropertyRID)
			require.Equal(t, tc.wantPType, ids.PropertyType)
			require.Equal(t, tc.wantSeller, ids.SellerAgentURL)
			place, _ := r.resolvePlacement(tc.impTagID)
			require.Equal(t, tc.wantPlace, place)
		})
	}
}
