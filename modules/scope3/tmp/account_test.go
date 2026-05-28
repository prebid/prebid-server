package tmp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveAuction_PropertyRIDFromExt(t *testing.T) {
	tests := []struct {
		name       string
		extJSON    string
		wantRID    string
		wantErr    string
	}{
		{
			name:    "property_rid present in ext",
			extJSON: `{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"01916f3a"}}}}}`,
			wantRID: "01916f3a",
		},
		{
			name:    "property_rid missing from ext",
			extJSON: `{}`,
			wantErr: "property_rid is required in request ext",
		},
		{
			name:    "ext is nil",
			extJSON: ``,
			wantErr: "property_rid is required in request ext",
		},
		{
			name:    "property_rid overridden in ext",
			extJSON: `{"prebid":{"modules":{"scope3":{"tmp":{"property_rid":"override-rid"}}}}}`,
			wantRID: "override-rid",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := accountResolver{
				requestExt: json.RawMessage(tc.extJSON),
				moduleCfg:  Config{RouterURL: "https://router"},
			}
			ids, err := r.resolveAuction()
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantRID, ids.PropertyRID)
			require.Equal(t, "https://router", ids.RouterURL)
		})
	}
}

func TestResolvePlacement_FromImpExt(t *testing.T) {
	tests := []struct {
		name      string
		impExtJSON string
		wantPlace string
		wantFound bool
	}{
		{
			name:       "placement_id present in imp ext",
			impExtJSON: `{"prebid":{"modules":{"scope3":{"tmp":{"placement_id":"header_728x90"}}}}}`,
			wantPlace:  "header_728x90",
			wantFound:  true,
		},
		{
			name:       "placement_id missing from imp ext",
			impExtJSON: `{}`,
			wantPlace:  "",
			wantFound:  false,
		},
		{
			name:       "imp ext is nil",
			impExtJSON: ``,
			wantPlace:  "",
			wantFound:  false,
		},
		{
			name:       "other fields in imp ext, no placement_id",
			impExtJSON: `{"prebid":{"modules":{"scope3":{"tmp":{}}}}}`,
			wantPlace:  "",
			wantFound:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := accountResolver{
				requestExt: json.RawMessage(`{}`),
				moduleCfg:  Config{},
			}
			place, ok := r.resolvePlacement(json.RawMessage(tc.impExtJSON))
			require.Equal(t, tc.wantFound, ok)
			require.Equal(t, tc.wantPlace, place)
		})
	}
}
