package openrtb_ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getFlag(in bool) *bool {
	return &in
}

func TestPriceFloorRulesGetEnforcePBS(t *testing.T) {
	tests := []struct {
		name   string
		floors *PriceFloorRules
		want   bool
	}{
		{
			name: "EnforcePBS_Enabled",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Enforcement: &PriceFloorEnforcement{
					EnforcePBS: getFlag(true),
				},
			},
			want: true,
		},
		{
			name: "EnforcePBS_NotProvided",
			floors: &PriceFloorRules{
				Enabled:     getFlag(true),
				Enforcement: &PriceFloorEnforcement{},
			},
			want: true,
		},
		{
			name: "EnforcePBS_Disabled",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Enforcement: &PriceFloorEnforcement{
					EnforcePBS: getFlag(false),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.floors.GetEnforcePBS()
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestPriceFloorRulesGetFloorsSkippedFlag(t *testing.T) {
	tests := []struct {
		name   string
		floors *PriceFloorRules
		want   bool
	}{
		{
			name: "Skipped_true",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Skipped: getFlag(true),
			},
			want: true,
		},
		{
			name: "Skipped_false",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Skipped: getFlag(false),
			},
			want: false,
		},
		{
			name: "Skipped_NotProvided",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.floors.GetFloorsSkippedFlag()
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestPriceFloorRulesGetEnforceRate(t *testing.T) {
	tests := []struct {
		name   string
		floors *PriceFloorRules
		want   int
	}{
		{
			name: "EnforceRate_100",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Enforcement: &PriceFloorEnforcement{
					EnforcePBS:  getFlag(true),
					EnforceRate: 100,
				},
			},
			want: 100,
		},
		{
			name: "EnforceRate_0",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Enforcement: &PriceFloorEnforcement{
					EnforcePBS:  getFlag(true),
					EnforceRate: 0,
				},
			},
			want: 0,
		},
		{
			name: "EnforceRate_NotProvided",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.floors.GetEnforceRate()
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestPriceFloorRulesGetEnforceDealsFlag(t *testing.T) {
	tests := []struct {
		name   string
		floors *PriceFloorRules
		want   bool
	}{
		{
			name: "FloorDeals_true",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Enforcement: &PriceFloorEnforcement{
					EnforcePBS:  getFlag(true),
					EnforceRate: 0,
					FloorDeals:  getFlag(true),
				},
			},
			want: true,
		},
		{
			name: "FloorDeals_false",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
				Enforcement: &PriceFloorEnforcement{
					EnforcePBS: getFlag(true),
					FloorDeals: getFlag(false),
				},
				Skipped: getFlag(false),
			},
			want: false,
		},
		{
			name: "FloorDeals_NotProvided",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.floors.GetEnforceDealsFlag()
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestPriceFloorRulesGetEnabled(t *testing.T) {
	tests := []struct {
		name   string
		floors *PriceFloorRules
		want   bool
	}{
		{
			name: "Enabled_true",
			floors: &PriceFloorRules{
				Enabled: getFlag(true),
			},
			want: true,
		},
		{
			name: "Enabled_false",
			floors: &PriceFloorRules{
				Enabled: getFlag(false),
			},
			want: false,
		},
		{
			name:   "Enabled_NotProvided",
			floors: &PriceFloorRules{},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.floors.GetEnabled()
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
