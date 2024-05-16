package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestValidatePMP(t *testing.T) {
	tests := []struct {
		name      string
		pmp       *openrtb2.PMP
		wantError bool
	}{
		{
			name:      "nil",
			pmp:       nil,
			wantError: false,
		},
		{
			name: "nil_deals",
			pmp: &openrtb2.PMP{
				Deals: nil,
			},
			wantError: false,
		},
		{
			name: "empty_deals",
			pmp: &openrtb2.PMP{
				Deals: []openrtb2.Deal{},
			},
			wantError: false,
		},
		{
			name: "one_deal",
			pmp: &openrtb2.PMP{
				Deals: []openrtb2.Deal{
					{
						ID: "deal1",
					},
				},
			},
			wantError: false,
		},
		{
			name: "one_deal_no_id",
			pmp: &openrtb2.PMP{
				Deals: []openrtb2.Deal{
					{
						ID: "",
					},
				},
			},
			wantError: true,
		},
		{
			name: "multiple_deals",
			pmp: &openrtb2.PMP{
				Deals: []openrtb2.Deal{
					{
						ID: "deal1",
					},
					{
						ID: "deal2",
					},
					{
						ID: "deal3",
					},
				},
			},
			wantError: false,
		},
		{
			name: "multiple_deals_no_id",
			pmp: &openrtb2.PMP{
				Deals: []openrtb2.Deal{
					{
						ID: "deal1",
					},
					{
						ID: "",
					},
					{
						ID: "deal3",
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePmp(tt.pmp, 1)
			if tt.wantError {
				assert.Error(t, result)
			} else {
				assert.NoError(t, result)
			}
		})
	}
}
