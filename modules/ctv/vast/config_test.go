package vast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeCTVVastConfig_NilInputs(t *testing.T) {
	result := MergeCTVVastConfig(nil, nil, nil)
	assert.Equal(t, CTVVastConfig{}, result)
}

func TestMergeCTVVastConfig_HostOnly(t *testing.T) {
	host := &CTVVastConfig{
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "EUR",
		VastVersionDefault: "4.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "balanced",
		CollisionPolicy:    "reject",
	}

	result := MergeCTVVastConfig(host, nil, nil)

	assert.Equal(t, "GAM_SSU", result.Receiver)
	assert.Equal(t, "EUR", result.DefaultCurrency)
	assert.Equal(t, "4.0", result.VastVersionDefault)
	assert.Equal(t, 5, result.MaxAdsInPod)
	assert.Equal(t, "balanced", result.SelectionStrategy)
	assert.Equal(t, "reject", result.CollisionPolicy)
}

func TestMergeCTVVastConfig_AccountOverridesHost(t *testing.T) {
	host := &CTVVastConfig{
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "EUR",
		VastVersionDefault: "4.0",
		MaxAdsInPod:        5,
	}
	account := &CTVVastConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     10,
	}

	result := MergeCTVVastConfig(host, account, nil)

	assert.Equal(t, "GAM_SSU", result.Receiver)       // from host
	assert.Equal(t, "USD", result.DefaultCurrency)    // overridden by account
	assert.Equal(t, "4.0", result.VastVersionDefault) // from host
	assert.Equal(t, 10, result.MaxAdsInPod)           // overridden by account
}

func TestMergeCTVVastConfig_ProfileOverridesAll(t *testing.T) {
	host := &CTVVastConfig{
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "EUR",
		VastVersionDefault: "4.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "max_revenue",
	}
	account := &CTVVastConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     10,
	}
	profile := &CTVVastConfig{
		VastVersionDefault: "4.2",
		MaxAdsInPod:        3,
		SelectionStrategy:  "min_duration",
	}

	result := MergeCTVVastConfig(host, account, profile)

	assert.Equal(t, "GAM_SSU", result.Receiver)               // from host
	assert.Equal(t, "USD", result.DefaultCurrency)            // from account
	assert.Equal(t, "4.2", result.VastVersionDefault)         // overridden by profile
	assert.Equal(t, 3, result.MaxAdsInPod)                    // overridden by profile
	assert.Equal(t, "min_duration", result.SelectionStrategy) // overridden by profile
}

func TestMergeCTVVastConfig_BoolPointers(t *testing.T) {
	trueVal := true
	falseVal := false

	host := &CTVVastConfig{
		Enabled: &trueVal,
		Debug:   &falseVal,
	}
	account := &CTVVastConfig{
		Debug: &trueVal,
	}
	profile := &CTVVastConfig{
		Enabled: &falseVal,
	}

	result := MergeCTVVastConfig(host, account, profile)

	assert.NotNil(t, result.Enabled)
	assert.False(t, *result.Enabled) // overridden by profile
	assert.NotNil(t, result.Debug)
	assert.True(t, *result.Debug) // from account (profile didn't set it)
}

func TestMergeCTVVastConfig_PlacementRules(t *testing.T) {
	floor := 1.5
	ceiling := 50.0
	profileFloor := 2.0

	host := &CTVVastConfig{
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM:   &floor,
				CeilingCPM: &ceiling,
				Currency:   "EUR",
			},
			Advertiser: &AdvertiserRulesConfig{
				BlockedDomains: []string{"blocked.com"},
			},
		},
	}
	account := &CTVVastConfig{
		Placement: &PlacementRulesConfig{
			Advertiser: &AdvertiserRulesConfig{
				BlockedDomains: []string{"account-blocked.com"},
			},
			Categories: &CategoryRulesConfig{
				BlockedCategories: []string{"IAB25"},
			},
		},
	}
	profile := &CTVVastConfig{
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM: &profileFloor,
			},
		},
	}

	result := MergeCTVVastConfig(host, account, profile)

	assert.NotNil(t, result.Placement)
	assert.NotNil(t, result.Placement.Pricing)
	assert.Equal(t, 2.0, *result.Placement.Pricing.FloorCPM)    // from profile
	assert.Equal(t, 50.0, *result.Placement.Pricing.CeilingCPM) // from host
	assert.Equal(t, "EUR", result.Placement.Pricing.Currency)   // from host

	assert.NotNil(t, result.Placement.Advertiser)
	assert.Equal(t, []string{"account-blocked.com"}, result.Placement.Advertiser.BlockedDomains) // from account

	assert.NotNil(t, result.Placement.Categories)
	assert.Equal(t, []string{"IAB25"}, result.Placement.Categories.BlockedCategories) // from account
}

func TestReceiverConfig_Defaults(t *testing.T) {
	cfg := CTVVastConfig{}
	rc := cfg.ReceiverConfig()

	assert.Equal(t, ReceiverType("GAM_SSU"), rc.Receiver)
	assert.Equal(t, "USD", rc.DefaultCurrency)
	assert.Equal(t, "3.0", rc.VastVersionDefault)
	assert.Equal(t, 10, rc.MaxAdsInPod)
	assert.Equal(t, SelectionStrategy("max_revenue"), rc.SelectionStrategy)
	assert.Equal(t, CollisionPolicy("VAST_WINS"), rc.CollisionPolicy)
	assert.False(t, rc.Debug)
}

func TestReceiverConfig_WithValues(t *testing.T) {
	debug := true
	cfg := CTVVastConfig{
		Receiver:           "GENERIC",
		DefaultCurrency:    "EUR",
		VastVersionDefault: "4.2",
		MaxAdsInPod:        7,
		SelectionStrategy:  "balanced",
		CollisionPolicy:    "warn",
		Debug:              &debug,
	}
	rc := cfg.ReceiverConfig()

	assert.Equal(t, ReceiverType("GENERIC"), rc.Receiver)
	assert.Equal(t, "EUR", rc.DefaultCurrency)
	assert.Equal(t, "4.2", rc.VastVersionDefault)
	assert.Equal(t, 7, rc.MaxAdsInPod)
	assert.Equal(t, SelectionStrategy("balanced"), rc.SelectionStrategy)
	assert.Equal(t, CollisionPolicy("warn"), rc.CollisionPolicy)
	assert.True(t, rc.Debug)
}

func TestReceiverConfig_PlacementRules(t *testing.T) {
	floor := 1.5
	ceiling := 100.0
	debug := true

	cfg := CTVVastConfig{
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM:   &floor,
				CeilingCPM: &ceiling,
				Currency:   "EUR",
			},
			Advertiser: &AdvertiserRulesConfig{
				BlockedDomains: []string{"blocked.com", "spam.com"},
				AllowedDomains: []string{"allowed.com"},
			},
			Categories: &CategoryRulesConfig{
				BlockedCategories: []string{"IAB25", "IAB26"},
				AllowedCategories: []string{"IAB1"},
			},
			Debug: &debug,
		},
	}
	rc := cfg.ReceiverConfig()

	assert.Equal(t, 1.5, rc.Placement.Pricing.FloorCPM)
	assert.Equal(t, 100.0, rc.Placement.Pricing.CeilingCPM)
	assert.Equal(t, "EUR", rc.Placement.Pricing.Currency)

	assert.Equal(t, []string{"blocked.com", "spam.com"}, rc.Placement.Advertiser.BlockedDomains)
	assert.Equal(t, []string{"allowed.com"}, rc.Placement.Advertiser.AllowedDomains)

	assert.Equal(t, []string{"IAB25", "IAB26"}, rc.Placement.Categories.BlockedCategories)
	assert.Equal(t, []string{"IAB1"}, rc.Placement.Categories.AllowedCategories)

	assert.True(t, rc.Placement.Debug)
}

func TestReceiverConfig_PlacementPricingDefaultCurrency(t *testing.T) {
	floor := 1.0
	cfg := CTVVastConfig{
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM: &floor,
				// Currency not set
			},
		},
	}
	rc := cfg.ReceiverConfig()

	assert.Equal(t, "USD", rc.Placement.Pricing.Currency)
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{
			name:     "nil returns false",
			enabled:  nil,
			expected: false,
		},
		{
			name:     "true returns true",
			enabled:  boolPtr(true),
			expected: true,
		},
		{
			name:     "false returns false",
			enabled:  boolPtr(false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CTVVastConfig{Enabled: tt.enabled}
			assert.Equal(t, tt.expected, cfg.IsEnabled())
		})
	}
}

func TestMergeCTVVastConfig_FullLayerPrecedence(t *testing.T) {
	// This test verifies the complete layering behavior:
	// profile > account > host

	host := &CTVVastConfig{
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "GBP",
		VastVersionDefault: "3.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "max_revenue",
		CollisionPolicy:    "reject",
		Enabled:            boolPtr(true),
		Debug:              boolPtr(false),
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM:   float64Ptr(1.0),
				CeilingCPM: float64Ptr(100.0),
				Currency:   "GBP",
			},
			Advertiser: &AdvertiserRulesConfig{
				BlockedDomains: []string{"host-blocked.com"},
			},
		},
	}

	account := &CTVVastConfig{
		DefaultCurrency: "EUR",
		MaxAdsInPod:     8,
		CollisionPolicy: "warn",
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM: float64Ptr(2.0),
				Currency: "EUR",
			},
			Categories: &CategoryRulesConfig{
				BlockedCategories: []string{"IAB25"},
			},
		},
	}

	profile := &CTVVastConfig{
		VastVersionDefault: "4.2",
		MaxAdsInPod:        3,
		Debug:              boolPtr(true),
		Placement: &PlacementRulesConfig{
			Pricing: &PricingRulesConfig{
				FloorCPM: float64Ptr(3.0),
			},
		},
	}

	result := MergeCTVVastConfig(host, account, profile)

	// Verify precedence
	assert.Equal(t, "GAM_SSU", result.Receiver)              // host (only set there)
	assert.Equal(t, "EUR", result.DefaultCurrency)           // account overrides host
	assert.Equal(t, "4.2", result.VastVersionDefault)        // profile overrides host
	assert.Equal(t, 3, result.MaxAdsInPod)                   // profile overrides account and host
	assert.Equal(t, "max_revenue", result.SelectionStrategy) // host (only set there)
	assert.Equal(t, "warn", result.CollisionPolicy)          // account overrides host
	assert.True(t, *result.Enabled)                          // host (only set there)
	assert.True(t, *result.Debug)                            // profile overrides host

	// Verify nested placement rules precedence
	assert.Equal(t, 3.0, *result.Placement.Pricing.FloorCPM)     // profile overrides account and host
	assert.Equal(t, 100.0, *result.Placement.Pricing.CeilingCPM) // host (only set there)
	assert.Equal(t, "EUR", result.Placement.Pricing.Currency)    // account overrides host

	assert.Equal(t, []string{"host-blocked.com"}, result.Placement.Advertiser.BlockedDomains) // host
	assert.Equal(t, []string{"IAB25"}, result.Placement.Categories.BlockedCategories)         // account
}

func TestMergeCTVVastConfig_EmptyStringsDoNotOverride(t *testing.T) {
	host := &CTVVastConfig{
		Receiver:        "GAM_SSU",
		DefaultCurrency: "EUR",
	}
	account := &CTVVastConfig{
		Receiver:        "", // empty string should not override
		DefaultCurrency: "USD",
	}

	result := MergeCTVVastConfig(host, account, nil)

	assert.Equal(t, "GAM_SSU", result.Receiver)    // empty string didn't override
	assert.Equal(t, "USD", result.DefaultCurrency) // non-empty string did override
}

func TestMergeCTVVastConfig_ZeroIntDoesNotOverride(t *testing.T) {
	host := &CTVVastConfig{
		MaxAdsInPod: 5,
	}
	account := &CTVVastConfig{
		MaxAdsInPod: 0, // zero should not override
	}

	result := MergeCTVVastConfig(host, account, nil)

	assert.Equal(t, 5, result.MaxAdsInPod) // zero didn't override
}

func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	falsePtr := boolPtr(false)

	assert.NotNil(t, truePtr)
	assert.True(t, *truePtr)
	assert.NotNil(t, falsePtr)
	assert.False(t, *falsePtr)
}

func TestFloat64Ptr(t *testing.T) {
	ptr := float64Ptr(1.5)
	assert.NotNil(t, ptr)
	assert.Equal(t, 1.5, *ptr)
}
