package vast

import (
	"testing"
)

func TestMergeCTVVastConfig_NilInputs(t *testing.T) {
	result := MergeCTVVastConfig(nil, nil, nil)
	
	if result.Enabled {
		t.Error("Expected Enabled to be false for nil inputs")
	}
	if result.Receiver != "" {
		t.Errorf("Expected empty Receiver, got %s", result.Receiver)
	}
}

func TestMergeCTVVastConfig_HostOnly(t *testing.T) {
	host := &CTVVastConfig{
		Enabled:            true,
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "USD",
		VastVersionDefault: "3.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "SINGLE",
		CollisionPolicy:    "VAST_WINS",
		EnableDebug:        false,
	}
	
	result := MergeCTVVastConfig(host, nil, nil)
	
	if !result.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if result.Receiver != "GAM_SSU" {
		t.Errorf("Expected Receiver GAM_SSU, got %s", result.Receiver)
	}
	if result.DefaultCurrency != "USD" {
		t.Errorf("Expected DefaultCurrency USD, got %s", result.DefaultCurrency)
	}
	if result.MaxAdsInPod != 5 {
		t.Errorf("Expected MaxAdsInPod 5, got %d", result.MaxAdsInPod)
	}
}

func TestMergeCTVVastConfig_AccountOverridesHost(t *testing.T) {
	host := &CTVVastConfig{
		Enabled:            true,
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "USD",
		VastVersionDefault: "3.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "SINGLE",
		CollisionPolicy:    "VAST_WINS",
	}
	
	account := &CTVVastConfig{
		Receiver:          "GENERIC",
		MaxAdsInPod:       8,
		SelectionStrategy: "TOP_N",
	}
	
	result := MergeCTVVastConfig(host, account, nil)
	
	// Account should override
	if result.Receiver != "GENERIC" {
		t.Errorf("Expected Receiver GENERIC, got %s", result.Receiver)
	}
	if result.MaxAdsInPod != 8 {
		t.Errorf("Expected MaxAdsInPod 8, got %d", result.MaxAdsInPod)
	}
	if result.SelectionStrategy != "TOP_N" {
		t.Errorf("Expected SelectionStrategy TOP_N, got %s", result.SelectionStrategy)
	}
	
	// Host values should be preserved where account didn't override
	if result.DefaultCurrency != "USD" {
		t.Errorf("Expected DefaultCurrency USD, got %s", result.DefaultCurrency)
	}
	if result.VastVersionDefault != "3.0" {
		t.Errorf("Expected VastVersionDefault 3.0, got %s", result.VastVersionDefault)
	}
}

func TestMergeCTVVastConfig_ProfileOverridesAll(t *testing.T) {
	host := &CTVVastConfig{
		Enabled:            true,
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "USD",
		VastVersionDefault: "3.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "SINGLE",
	}
	
	account := &CTVVastConfig{
		Receiver:    "GENERIC",
		MaxAdsInPod: 8,
	}
	
	profile := &CTVVastConfig{
		Receiver:           "GAM_SSU",
		VastVersionDefault: "4.0",
		MaxAdsInPod:        3,
		EnableDebug:        true,
	}
	
	result := MergeCTVVastConfig(host, account, profile)
	
	// Profile should override
	if result.Receiver != "GAM_SSU" {
		t.Errorf("Expected Receiver GAM_SSU from profile, got %s", result.Receiver)
	}
	if result.VastVersionDefault != "4.0" {
		t.Errorf("Expected VastVersionDefault 4.0 from profile, got %s", result.VastVersionDefault)
	}
	if result.MaxAdsInPod != 3 {
		t.Errorf("Expected MaxAdsInPod 3 from profile, got %d", result.MaxAdsInPod)
	}
	if !result.EnableDebug {
		t.Error("Expected EnableDebug true from profile")
	}
	
	// Host values should be preserved where not overridden
	if result.DefaultCurrency != "USD" {
		t.Errorf("Expected DefaultCurrency USD from host, got %s", result.DefaultCurrency)
	}
}

func TestMergeCTVVastConfig_PlacementRules(t *testing.T) {
	host := &CTVVastConfig{
		PlacementRules: &PlacementRulesConfig{
			PricingPlacement:    "INLINE",
			AdvertiserPlacement: "INLINE",
			CategoriesPlacement: "EXTENSIONS",
			DebugPlacement:      "EXTENSIONS",
		},
	}
	
	account := &CTVVastConfig{
		PlacementRules: &PlacementRulesConfig{
			PricingPlacement: "EXTENSIONS",
			// Other fields not set, should inherit from host
		},
	}
	
	result := MergeCTVVastConfig(host, account, nil)
	
	if result.PlacementRules == nil {
		t.Fatal("Expected PlacementRules to be set")
	}
	
	// Account should override pricing
	if result.PlacementRules.PricingPlacement != "EXTENSIONS" {
		t.Errorf("Expected PricingPlacement EXTENSIONS, got %s", result.PlacementRules.PricingPlacement)
	}
	
	// Other fields should inherit from host
	if result.PlacementRules.AdvertiserPlacement != "INLINE" {
		t.Errorf("Expected AdvertiserPlacement INLINE, got %s", result.PlacementRules.AdvertiserPlacement)
	}
	if result.PlacementRules.CategoriesPlacement != "EXTENSIONS" {
		t.Errorf("Expected CategoriesPlacement EXTENSIONS, got %s", result.PlacementRules.CategoriesPlacement)
	}
}

func TestReceiverConfig_Defaults(t *testing.T) {
	cfg := CTVVastConfig{}
	
	rc := cfg.ReceiverConfig()
	
	if rc.Receiver != "GENERIC" {
		t.Errorf("Expected default Receiver GENERIC, got %s", rc.Receiver)
	}
	if rc.DefaultCurrency != "USD" {
		t.Errorf("Expected default Currency USD, got %s", rc.DefaultCurrency)
	}
	if rc.VastVersionDefault != "3.0" {
		t.Errorf("Expected default VastVersion 3.0, got %s", rc.VastVersionDefault)
	}
	if rc.MaxAdsInPod != 10 {
		t.Errorf("Expected default MaxAdsInPod 10, got %d", rc.MaxAdsInPod)
	}
	if rc.SelectionStrategy != "SINGLE" {
		t.Errorf("Expected default SelectionStrategy SINGLE, got %s", rc.SelectionStrategy)
	}
	if rc.CollisionPolicy != CollisionPolicyVastWins {
		t.Errorf("Expected default CollisionPolicy VAST_WINS, got %s", rc.CollisionPolicy)
	}
}

func TestReceiverConfig_CustomValues(t *testing.T) {
	cfg := CTVVastConfig{
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "EUR",
		VastVersionDefault: "4.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  "TOP_N",
		CollisionPolicy:    "OPENRTB_WINS",
		EnableDebug:        true,
	}
	
	rc := cfg.ReceiverConfig()
	
	if rc.Receiver != "GAM_SSU" {
		t.Errorf("Expected Receiver GAM_SSU, got %s", rc.Receiver)
	}
	if rc.DefaultCurrency != "EUR" {
		t.Errorf("Expected Currency EUR, got %s", rc.DefaultCurrency)
	}
	if rc.VastVersionDefault != "4.0" {
		t.Errorf("Expected VastVersion 4.0, got %s", rc.VastVersionDefault)
	}
	if rc.MaxAdsInPod != 5 {
		t.Errorf("Expected MaxAdsInPod 5, got %d", rc.MaxAdsInPod)
	}
	if rc.SelectionStrategy != "TOP_N" {
		t.Errorf("Expected SelectionStrategy TOP_N, got %s", rc.SelectionStrategy)
	}
	if rc.CollisionPolicy != CollisionPolicyOpenRTBWins {
		t.Errorf("Expected CollisionPolicy OPENRTB_WINS, got %s", rc.CollisionPolicy)
	}
	if !rc.EnableDebug {
		t.Error("Expected EnableDebug true")
	}
}

func TestReceiverConfig_PlacementRulesDefaults(t *testing.T) {
	cfg := CTVVastConfig{}
	
	rc := cfg.ReceiverConfig()
	
	if rc.PlacementRules.PricingPlacement != PlacementInline {
		t.Errorf("Expected default PricingPlacement INLINE, got %s", rc.PlacementRules.PricingPlacement)
	}
	if rc.PlacementRules.AdvertiserPlacement != PlacementInline {
		t.Errorf("Expected default AdvertiserPlacement INLINE, got %s", rc.PlacementRules.AdvertiserPlacement)
	}
	if rc.PlacementRules.CategoriesPlacement != PlacementExtensions {
		t.Errorf("Expected default CategoriesPlacement EXTENSIONS, got %s", rc.PlacementRules.CategoriesPlacement)
	}
	if rc.PlacementRules.DebugPlacement != PlacementExtensions {
		t.Errorf("Expected default DebugPlacement EXTENSIONS, got %s", rc.PlacementRules.DebugPlacement)
	}
}

func TestReceiverConfig_PlacementRulesCustom(t *testing.T) {
	cfg := CTVVastConfig{
		PlacementRules: &PlacementRulesConfig{
			PricingPlacement:    "SKIP",
			AdvertiserPlacement: "EXTENSIONS",
			CategoriesPlacement: "INLINE",
			DebugPlacement:      "SKIP",
		},
	}
	
	rc := cfg.ReceiverConfig()
	
	if rc.PlacementRules.PricingPlacement != PlacementSkip {
		t.Errorf("Expected PricingPlacement SKIP, got %s", rc.PlacementRules.PricingPlacement)
	}
	if rc.PlacementRules.AdvertiserPlacement != PlacementExtensions {
		t.Errorf("Expected AdvertiserPlacement EXTENSIONS, got %s", rc.PlacementRules.AdvertiserPlacement)
	}
	if rc.PlacementRules.CategoriesPlacement != PlacementInline {
		t.Errorf("Expected CategoriesPlacement INLINE, got %s", rc.PlacementRules.CategoriesPlacement)
	}
	if rc.PlacementRules.DebugPlacement != PlacementSkip {
		t.Errorf("Expected DebugPlacement SKIP, got %s", rc.PlacementRules.DebugPlacement)
	}
}

func TestMergeCTVVastConfig_ComplexScenario(t *testing.T) {
	// Host sets base configuration
	host := &CTVVastConfig{
		Enabled:            true,
		Receiver:           "GAM_SSU",
		DefaultCurrency:    "USD",
		VastVersionDefault: "3.0",
		MaxAdsInPod:        10,
		SelectionStrategy:  "SINGLE",
		CollisionPolicy:    "VAST_WINS",
		EnableDebug:        false,
		PlacementRules: &PlacementRulesConfig{
			PricingPlacement:    "INLINE",
			AdvertiserPlacement: "INLINE",
			CategoriesPlacement: "EXTENSIONS",
			DebugPlacement:      "EXTENSIONS",
		},
	}
	
	// Account overrides some fields
	account := &CTVVastConfig{
		VastVersionDefault: "4.0",
		MaxAdsInPod:        8,
		PlacementRules: &PlacementRulesConfig{
			CategoriesPlacement: "INLINE",
		},
	}
	
	// Profile overrides more fields
	profile := &CTVVastConfig{
		MaxAdsInPod:        5,
		EnableDebug:        true,
		CollisionPolicy:    "OPENRTB_WINS",
		PlacementRules: &PlacementRulesConfig{
			DebugPlacement: "INLINE",
		},
	}
	
	result := MergeCTVVastConfig(host, account, profile)
	
	// Verify final merged values
	if !result.Enabled {
		t.Error("Expected Enabled true from host")
	}
	if result.Receiver != "GAM_SSU" {
		t.Errorf("Expected Receiver GAM_SSU from host, got %s", result.Receiver)
	}
	if result.DefaultCurrency != "USD" {
		t.Errorf("Expected DefaultCurrency USD from host, got %s", result.DefaultCurrency)
	}
	if result.VastVersionDefault != "4.0" {
		t.Errorf("Expected VastVersionDefault 4.0 from account, got %s", result.VastVersionDefault)
	}
	if result.MaxAdsInPod != 5 {
		t.Errorf("Expected MaxAdsInPod 5 from profile, got %d", result.MaxAdsInPod)
	}
	if result.SelectionStrategy != "SINGLE" {
		t.Errorf("Expected SelectionStrategy SINGLE from host, got %s", result.SelectionStrategy)
	}
	if result.CollisionPolicy != "OPENRTB_WINS" {
		t.Errorf("Expected CollisionPolicy OPENRTB_WINS from profile, got %s", result.CollisionPolicy)
	}
	if !result.EnableDebug {
		t.Error("Expected EnableDebug true from profile")
	}
	
	// Verify placement rules merged correctly
	if result.PlacementRules == nil {
		t.Fatal("Expected PlacementRules to be set")
	}
	if result.PlacementRules.PricingPlacement != "INLINE" {
		t.Errorf("Expected PricingPlacement INLINE from host, got %s", result.PlacementRules.PricingPlacement)
	}
	if result.PlacementRules.AdvertiserPlacement != "INLINE" {
		t.Errorf("Expected AdvertiserPlacement INLINE from host, got %s", result.PlacementRules.AdvertiserPlacement)
	}
	if result.PlacementRules.CategoriesPlacement != "INLINE" {
		t.Errorf("Expected CategoriesPlacement INLINE from account, got %s", result.PlacementRules.CategoriesPlacement)
	}
	if result.PlacementRules.DebugPlacement != "INLINE" {
		t.Errorf("Expected DebugPlacement INLINE from profile, got %s", result.PlacementRules.DebugPlacement)
	}
}
