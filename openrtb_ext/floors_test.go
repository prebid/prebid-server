package openrtb_ext

import (
	"testing"
)

func getIntPtr(val int) *int {
	return &val
}

func TestPriceFloorRulesDeepCopy(t *testing.T) {
	type fields struct {
		FloorMin           float64
		FloorMinCur        string
		SkipRate           int
		Location           *PriceFloorEndpoint
		Data               *PriceFloorData
		Enforcement        *PriceFloorEnforcement
		Enabled            *bool
		Skipped            *bool
		FloorProvider      string
		FetchStatus        string
		PriceFloorLocation string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "DeepCopy does not share same reference",
			fields: fields{
				FloorMin:    10,
				FloorMinCur: "INR",
				SkipRate:    0,
				Location: &PriceFloorEndpoint{
					URL: "https://test/floors",
				},
				Data: &PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []PriceFloorModelGroup{
						{
							Currency:    "INR",
							ModelWeight: getIntPtr(1),
							SkipRate:    0,
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := &PriceFloorRules{
				FloorMin:           tt.fields.FloorMin,
				FloorMinCur:        tt.fields.FloorMinCur,
				SkipRate:           tt.fields.SkipRate,
				Location:           tt.fields.Location,
				Data:               tt.fields.Data,
				Enforcement:        tt.fields.Enforcement,
				Enabled:            tt.fields.Enabled,
				Skipped:            tt.fields.Skipped,
				FloorProvider:      tt.fields.FloorProvider,
				FetchStatus:        tt.fields.FetchStatus,
				PriceFloorLocation: tt.fields.PriceFloorLocation,
			}
			got := pf.DeepCopy()
			if got == pf {
				t.Errorf("Rules reference are same")
			}
			if got.Data == pf.Data {
				t.Errorf("Floor data reference is same")
			}
		})
	}
}
