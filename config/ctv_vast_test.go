package config

import (
	"testing"
)

func TestCTVVastDefaults(t *testing.T) {
	defaults := CTVVastDefaults()

	if defaults.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if defaults.Receiver != "GAM_SSU" {
		t.Errorf("Expected receiver GAM_SSU, got %s", defaults.Receiver)
	}

	if defaults.VastVersionDefault != "3.0" {
		t.Errorf("Expected version 3.0, got %s", defaults.VastVersionDefault)
	}

	if defaults.DefaultCurrency != "USD" {
		t.Errorf("Expected currency USD, got %s", defaults.DefaultCurrency)
	}

	if defaults.MaxAdsInPod != 1 {
		t.Errorf("Expected max ads 1, got %d", defaults.MaxAdsInPod)
	}

	if defaults.SelectionStrategy != "SINGLE" {
		t.Errorf("Expected strategy SINGLE, got %s", defaults.SelectionStrategy)
	}

	if defaults.CollisionPolicy != "VAST_WINS" {
		t.Errorf("Expected collision policy VAST_WINS, got %s", defaults.CollisionPolicy)
	}

	if defaults.PlacementRules.Price != "INLINE" {
		t.Errorf("Expected price placement INLINE, got %s", defaults.PlacementRules.Price)
	}

	if defaults.MacroConfig.UnknownMacroPolicy != "KEEP" {
		t.Errorf("Expected unknown macro policy KEEP, got %s", defaults.MacroConfig.UnknownMacroPolicy)
	}
}

func TestMergeCTVVastConfig_HostOnly(t *testing.T) {
	host := CTVVastDefaults()
	host.MaxAdsInPod = 5
	host.DefaultCurrency = "EUR"

	result := MergeCTVVastConfig(&host, nil, nil)

	if result.MaxAdsInPod != 5 {
		t.Errorf("Expected max ads 5, got %d", result.MaxAdsInPod)
	}

	if result.DefaultCurrency != "EUR" {
		t.Errorf("Expected currency EUR, got %s", result.DefaultCurrency)
	}

	// Other fields should remain default
	if result.Receiver != "GAM_SSU" {
		t.Errorf("Expected receiver GAM_SSU, got %s", result.Receiver)
	}
}

func TestMergeCTVVastConfig_AccountOverridesHost(t *testing.T) {
	host := CTVVastDefaults()
	host.MaxAdsInPod = 5
	host.DefaultCurrency = "EUR"

	account := &CTVVast{
		MaxAdsInPod: 10,
		Receiver:    "GENERIC",
	}

	result := MergeCTVVastConfig(&host, account, nil)

	// Account overrides
	if result.MaxAdsInPod != 10 {
		t.Errorf("Expected max ads 10, got %d", result.MaxAdsInPod)
	}

	if result.Receiver != "GENERIC" {
		t.Errorf("Expected receiver GENERIC, got %s", result.Receiver)
	}

	// Host value preserved for non-overridden fields
	if result.DefaultCurrency != "EUR" {
		t.Errorf("Expected currency EUR, got %s", result.DefaultCurrency)
	}
}

func TestMergeCTVVastConfig_ProfileOverridesAll(t *testing.T) {
	host := CTVVastDefaults()
	host.MaxAdsInPod = 5
	host.DefaultCurrency = "EUR"

	account := &CTVVast{
		MaxAdsInPod: 10,
		Receiver:    "GENERIC",
	}

	profile := &CTVVast{
		MaxAdsInPod:     15,
		SelectionStrategy: "TOP_N",
	}

	result := MergeCTVVastConfig(&host, account, profile)

	// Profile has highest priority
	if result.MaxAdsInPod != 15 {
		t.Errorf("Expected max ads 15, got %d", result.MaxAdsInPod)
	}

	if result.SelectionStrategy != "TOP_N" {
		t.Errorf("Expected strategy TOP_N, got %s", result.SelectionStrategy)
	}

	// Account override preserved for fields not in profile
	if result.Receiver != "GENERIC" {
		t.Errorf("Expected receiver GENERIC, got %s", result.Receiver)
	}

	// Host value preserved for fields not overridden
	if result.DefaultCurrency != "EUR" {
		t.Errorf("Expected currency EUR, got %s", result.DefaultCurrency)
	}
}

func TestMergeCTVVastConfig_PlacementRules(t *testing.T) {
	host := CTVVastDefaults()
	host.PlacementRules.Price = "INLINE"
	host.PlacementRules.Advertiser = "INLINE"

	account := &CTVVast{
		PlacementRules: PlacementRules{
			Price:      "EXTENSIONS",
			Categories: "INLINE",
		},
	}

	profile := &CTVVast{
		PlacementRules: PlacementRules{
			Advertiser: "SKIP",
		},
	}

	result := MergeCTVVastConfig(&host, account, profile)

	// Profile overrides
	if result.PlacementRules.Advertiser != "SKIP" {
		t.Errorf("Expected advertiser SKIP, got %s", result.PlacementRules.Advertiser)
	}

	// Account overrides
	if result.PlacementRules.Price != "EXTENSIONS" {
		t.Errorf("Expected price EXTENSIONS, got %s", result.PlacementRules.Price)
	}

	if result.PlacementRules.Categories != "INLINE" {
		t.Errorf("Expected categories INLINE, got %s", result.PlacementRules.Categories)
	}

	// Host values preserved for non-overridden fields
	if result.PlacementRules.Duration != "INLINE" {
		t.Errorf("Expected duration INLINE from defaults, got %s", result.PlacementRules.Duration)
	}
}

func TestMergeCTVVastConfig_MacroMappings(t *testing.T) {
	host := CTVVastDefaults()
	host.MacroConfig.Mappings = map[string]MacroMapping{
		"MACRO1": {Source: "query", Key: "m1"},
		"MACRO2": {Source: "query", Key: "m2"},
	}

	account := &CTVVast{
		MacroConfig: MacroConfig{
			UnknownMacroPolicy: "REMOVE",
			Mappings: map[string]MacroMapping{
				"MACRO2": {Source: "header", Key: "X-M2"}, // Override
				"MACRO3": {Source: "query", Key: "m3"},    // New
			},
		},
	}

	result := MergeCTVVastConfig(&host, account, nil)

	// Policy override
	if result.MacroConfig.UnknownMacroPolicy != "REMOVE" {
		t.Errorf("Expected policy REMOVE, got %s", result.MacroConfig.UnknownMacroPolicy)
	}

	// Check mappings
	if len(result.MacroConfig.Mappings) != 3 {
		t.Errorf("Expected 3 mappings, got %d", len(result.MacroConfig.Mappings))
	}

	// MACRO1 preserved from host
	if m, ok := result.MacroConfig.Mappings["MACRO1"]; !ok || m.Source != "query" || m.Key != "m1" {
		t.Errorf("MACRO1 mapping incorrect: %+v", m)
	}

	// MACRO2 overridden by account
	if m, ok := result.MacroConfig.Mappings["MACRO2"]; !ok || m.Source != "header" || m.Key != "X-M2" {
		t.Errorf("MACRO2 mapping incorrect: %+v", m)
	}

	// MACRO3 added by account
	if m, ok := result.MacroConfig.Mappings["MACRO3"]; !ok || m.Source != "query" || m.Key != "m3" {
		t.Errorf("MACRO3 mapping incorrect: %+v", m)
	}
}

func TestMergeCTVVastConfig_NilInputs(t *testing.T) {
	// All nil should return defaults
	result := MergeCTVVastConfig(nil, nil, nil)

	defaults := CTVVastDefaults()
	
	if result.Receiver != defaults.Receiver {
		t.Errorf("Expected default receiver, got %s", result.Receiver)
	}

	if result.MaxAdsInPod != defaults.MaxAdsInPod {
		t.Errorf("Expected default max ads, got %d", result.MaxAdsInPod)
	}
}

func TestMergePlacementRules(t *testing.T) {
	base := PlacementRules{
		Price:      "INLINE",
		Advertiser: "INLINE",
		Categories: "EXTENSIONS",
	}

	override := PlacementRules{
		Price:    "EXTENSIONS",
		Duration: "SKIP",
	}

	mergePlacementRules(&base, &override)

	if base.Price != "EXTENSIONS" {
		t.Errorf("Expected price EXTENSIONS, got %s", base.Price)
	}

	if base.Duration != "SKIP" {
		t.Errorf("Expected duration SKIP, got %s", base.Duration)
	}

	// Non-overridden should remain
	if base.Advertiser != "INLINE" {
		t.Errorf("Expected advertiser INLINE, got %s", base.Advertiser)
	}

	if base.Categories != "EXTENSIONS" {
		t.Errorf("Expected categories EXTENSIONS, got %s", base.Categories)
	}
}

func TestMergeMacroConfig(t *testing.T) {
	base := MacroConfig{
		UnknownMacroPolicy: "KEEP",
		Mappings: map[string]MacroMapping{
			"M1": {Source: "query"},
		},
	}

	override := MacroConfig{
		UnknownMacroPolicy: "REMOVE",
		Mappings: map[string]MacroMapping{
			"M2": {Source: "header"},
		},
	}

	mergeMacroConfig(&base, &override)

	if base.UnknownMacroPolicy != "REMOVE" {
		t.Errorf("Expected policy REMOVE, got %s", base.UnknownMacroPolicy)
	}

	if len(base.Mappings) != 2 {
		t.Errorf("Expected 2 mappings, got %d", len(base.Mappings))
	}

	if _, ok := base.Mappings["M1"]; !ok {
		t.Error("Expected M1 mapping to be preserved")
	}

	if _, ok := base.Mappings["M2"]; !ok {
		t.Error("Expected M2 mapping to be added")
	}
}
