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
			name:        "invalid property_type rejected",
			accountJSON: `{"scope3":{"tmp":{"property_rid":"r","property_type":"made_up_type","placements":{"h":"h1"}}}}`,
			extJSON:     `{}`,
			impTagID:    "h",
			wantErr:     `property_type "made_up_type" is not a valid`,
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

func TestResolveAuctionIdentifiers_MissingSellerAgentURL(t *testing.T) {
	moduleCfg := Config{RouterURL: "https://router"} // No SellerAgentURL
	r := accountResolver{
		accountConfig: json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website"}}}`),
		requestExt:    json.RawMessage(`{}`),
		moduleCfg:     moduleCfg,
	}
	_, err := r.resolveAuction()
	require.Error(t, err)
	require.Contains(t, err.Error(), "seller_agent_url is required")
}

func TestResolveAuctionIdentifiers_MissingRouterURL(t *testing.T) {
	moduleCfg := Config{SellerAgentURL: "https://us"} // No RouterURL
	r := accountResolver{
		accountConfig: json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website"}}}`),
		requestExt:    json.RawMessage(`{}`),
		moduleCfg:     moduleCfg,
	}
	_, err := r.resolveAuction()
	require.Error(t, err)
	require.Contains(t, err.Error(), "router_url is required")
}

func TestResolveAuctionIdentifiers_AccountOverridesModuleDefaults(t *testing.T) {
	moduleCfg := Config{
		RouterURL:      "https://router-module",
		SellerAgentURL: "https://us-module",
	}
	r := accountResolver{
		accountConfig: json.RawMessage(`{"scope3":{"tmp":{"property_rid":"r","property_type":"website","router_url":"https://router-account","seller_agent_url":"https://us-account"}}}`),
		requestExt:    json.RawMessage(`{}`),
		moduleCfg:     moduleCfg,
	}
	ids, err := r.resolveAuction()
	require.NoError(t, err)
	require.Equal(t, "https://router-account", ids.RouterURL)
	require.Equal(t, "https://us-account", ids.SellerAgentURL)
}

func TestResolvePlacement_ExtPlacementIDTakesPrecedence(t *testing.T) {
	moduleCfg := Config{}
	r := accountResolver{
		accountConfig: json.RawMessage(`{"scope3":{"tmp":{"placements":{"h":"acct-placement"}}}}`),
		requestExt:    json.RawMessage(`{"prebid":{"modules":{"scope3":{"tmp":{"placement_id":"ext-placement"}}}}}`),
		moduleCfg:     moduleCfg,
	}
	place, ok := r.resolvePlacement("h")
	require.True(t, ok)
	require.Equal(t, "ext-placement", place)
}

func TestResolvePlacement_AccountFallback(t *testing.T) {
	moduleCfg := Config{}
	r := accountResolver{
		accountConfig: json.RawMessage(`{"scope3":{"tmp":{"placements":{"h":"acct-placement"}}}}`),
		requestExt:    json.RawMessage(`{}`),
		moduleCfg:     moduleCfg,
	}
	place, ok := r.resolvePlacement("h")
	require.True(t, ok)
	require.Equal(t, "acct-placement", place)
}

func TestResolvePlacement_NotFound(t *testing.T) {
	moduleCfg := Config{}
	r := accountResolver{
		accountConfig: json.RawMessage(`{"scope3":{"tmp":{"placements":{"other":"acct-placement"}}}}`),
		requestExt:    json.RawMessage(`{}`),
		moduleCfg:     moduleCfg,
	}
	place, ok := r.resolvePlacement("h")
	require.False(t, ok)
	require.Equal(t, "", place)
}
