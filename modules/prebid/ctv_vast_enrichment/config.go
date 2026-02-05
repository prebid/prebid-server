package vast

// CTVVastConfig represents the configuration for CTV VAST processing.
// It supports PBS-style layered configuration where profile overrides account,
// and account overrides host-level settings.
type CTVVastConfig struct {
	// Enabled controls whether CTV VAST processing is active.
	Enabled *bool `json:"enabled,omitempty" mapstructure:"enabled"`
	// Receiver identifies the downstream ad receiver type (e.g., "GAM_SSU", "GENERIC").
	Receiver string `json:"receiver,omitempty" mapstructure:"receiver"`
	// DefaultCurrency is the currency to use when not specified (default: "USD").
	DefaultCurrency string `json:"default_currency,omitempty" mapstructure:"default_currency"`
	// VastVersionDefault is the default VAST version to output (default: "3.0").
	VastVersionDefault string `json:"vast_version_default,omitempty" mapstructure:"vast_version_default"`
	// MaxAdsInPod is the maximum number of ads allowed in a pod (default: 10).
	MaxAdsInPod int `json:"max_ads_in_pod,omitempty" mapstructure:"max_ads_in_pod"`
	// SelectionStrategy defines how bids are selected (e.g., "SINGLE", "TOP_N").
	SelectionStrategy string `json:"selection_strategy,omitempty" mapstructure:"selection_strategy"`
	// CollisionPolicy defines how competitive separation is handled (default: "VAST_WINS").
	CollisionPolicy string `json:"collision_policy,omitempty" mapstructure:"collision_policy"`
	// AllowSkeletonVast allows bids without AdM content (skeleton VAST).
	AllowSkeletonVast *bool `json:"allow_skeleton_vast,omitempty" mapstructure:"allow_skeleton_vast"`
	// Placement contains placement-specific rules.
	Placement *PlacementRulesConfig `json:"placement,omitempty" mapstructure:"placement"`
	// Debug enables debug mode with additional output.
	Debug *bool `json:"debug,omitempty" mapstructure:"debug"`
}

// PlacementRulesConfig contains rules for validating and filtering bids.
type PlacementRulesConfig struct {
	// Pricing contains price floor and ceiling rules.
	Pricing *PricingRulesConfig `json:"pricing,omitempty" mapstructure:"pricing"`
	// Advertiser contains advertiser-based filtering rules.
	Advertiser *AdvertiserRulesConfig `json:"advertiser,omitempty" mapstructure:"advertiser"`
	// Categories contains category-based filtering rules.
	Categories *CategoryRulesConfig `json:"categories,omitempty" mapstructure:"categories"`
	// PricingPlacement defines where to place pricing: "VAST_PRICING" or "EXTENSION".
	PricingPlacement string `json:"pricing_placement,omitempty" mapstructure:"pricing_placement"`
	// AdvertiserPlacement defines where to place advertiser: "ADVERTISER_TAG" or "EXTENSION".
	AdvertiserPlacement string `json:"advertiser_placement,omitempty" mapstructure:"advertiser_placement"`
	// Debug enables debug output for placement rules.
	Debug *bool `json:"debug,omitempty" mapstructure:"debug"`
}

// PricingRulesConfig defines pricing constraints for bid selection.
type PricingRulesConfig struct {
	// FloorCPM is the minimum CPM allowed.
	FloorCPM *float64 `json:"floor_cpm,omitempty" mapstructure:"floor_cpm"`
	// CeilingCPM is the maximum CPM allowed (0 = no ceiling).
	CeilingCPM *float64 `json:"ceiling_cpm,omitempty" mapstructure:"ceiling_cpm"`
	// Currency is the currency for floor/ceiling values.
	Currency string `json:"currency,omitempty" mapstructure:"currency"`
}

// AdvertiserRulesConfig defines advertiser-based filtering.
type AdvertiserRulesConfig struct {
	// BlockedDomains is a list of advertiser domains to reject.
	BlockedDomains []string `json:"blocked_domains,omitempty" mapstructure:"blocked_domains"`
	// AllowedDomains is a whitelist of allowed domains (empty = allow all).
	AllowedDomains []string `json:"allowed_domains,omitempty" mapstructure:"allowed_domains"`
}

// CategoryRulesConfig defines category-based filtering.
type CategoryRulesConfig struct {
	// BlockedCategories is a list of IAB categories to reject.
	BlockedCategories []string `json:"blocked_categories,omitempty" mapstructure:"blocked_categories"`
	// AllowedCategories is a whitelist of allowed categories (empty = allow all).
	AllowedCategories []string `json:"allowed_categories,omitempty" mapstructure:"allowed_categories"`
}

// Default values for CTVVastConfig.
const (
	DefaultVastVersion       = "3.0"
	DefaultCurrency          = "USD"
	DefaultMaxAdsInPod       = 10
	DefaultCollisionPolicy   = "VAST_WINS"
	DefaultReceiver          = "GAM_SSU"
	DefaultSelectionStrategy = "max_revenue"

	// Placement constants for pricing
	PlacementVastPricing = "VAST_PRICING"
	PlacementExtension   = "EXTENSION"

	// Placement constants for advertiser
	PlacementAdvertiserTag = "ADVERTISER_TAG"
	// PlacementExtension is also used for advertiser
)

// MergeCTVVastConfig merges configuration from host, account, and profile layers.
// The precedence order is: profile > account > host (profile values override account, which overrides host).
// Only non-zero values override; nil pointers and empty strings are considered "not set".
func MergeCTVVastConfig(host, account, profile *CTVVastConfig) CTVVastConfig {
	result := CTVVastConfig{}

	// Start with host config
	if host != nil {
		result = mergeIntoConfig(result, *host)
	}

	// Override with account config
	if account != nil {
		result = mergeIntoConfig(result, *account)
	}

	// Override with profile config (highest precedence)
	if profile != nil {
		result = mergeIntoConfig(result, *profile)
	}

	return result
}

// mergeIntoConfig merges src into dst, where non-zero values in src override dst.
func mergeIntoConfig(dst, src CTVVastConfig) CTVVastConfig {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if src.Receiver != "" {
		dst.Receiver = src.Receiver
	}
	if src.DefaultCurrency != "" {
		dst.DefaultCurrency = src.DefaultCurrency
	}
	if src.VastVersionDefault != "" {
		dst.VastVersionDefault = src.VastVersionDefault
	}
	if src.MaxAdsInPod != 0 {
		dst.MaxAdsInPod = src.MaxAdsInPod
	}
	if src.SelectionStrategy != "" {
		dst.SelectionStrategy = src.SelectionStrategy
	}
	if src.CollisionPolicy != "" {
		dst.CollisionPolicy = src.CollisionPolicy
	}
	if src.AllowSkeletonVast != nil {
		dst.AllowSkeletonVast = src.AllowSkeletonVast
	}
	if src.Debug != nil {
		dst.Debug = src.Debug
	}

	// Merge placement rules
	if src.Placement != nil {
		if dst.Placement == nil {
			dst.Placement = &PlacementRulesConfig{}
		}
		dst.Placement = mergePlacementRules(dst.Placement, src.Placement)
	}

	return dst
}

// mergePlacementRules merges placement rules from src into dst.
func mergePlacementRules(dst, src *PlacementRulesConfig) *PlacementRulesConfig {
	if dst == nil {
		dst = &PlacementRulesConfig{}
	}
	if src == nil {
		return dst
	}

	if src.Debug != nil {
		dst.Debug = src.Debug
	}

	// Merge pricing rules
	if src.Pricing != nil {
		if dst.Pricing == nil {
			dst.Pricing = &PricingRulesConfig{}
		}
		dst.Pricing = mergePricingRules(dst.Pricing, src.Pricing)
	}

	// Merge advertiser rules
	if src.Advertiser != nil {
		if dst.Advertiser == nil {
			dst.Advertiser = &AdvertiserRulesConfig{}
		}
		dst.Advertiser = mergeAdvertiserRules(dst.Advertiser, src.Advertiser)
	}

	// Merge category rules
	if src.Categories != nil {
		if dst.Categories == nil {
			dst.Categories = &CategoryRulesConfig{}
		}
		dst.Categories = mergeCategoryRules(dst.Categories, src.Categories)
	}

	return dst
}

// mergePricingRules merges pricing rules from src into dst.
func mergePricingRules(dst, src *PricingRulesConfig) *PricingRulesConfig {
	if src.FloorCPM != nil {
		dst.FloorCPM = src.FloorCPM
	}
	if src.CeilingCPM != nil {
		dst.CeilingCPM = src.CeilingCPM
	}
	if src.Currency != "" {
		dst.Currency = src.Currency
	}
	return dst
}

// mergeAdvertiserRules merges advertiser rules from src into dst.
func mergeAdvertiserRules(dst, src *AdvertiserRulesConfig) *AdvertiserRulesConfig {
	if len(src.BlockedDomains) > 0 {
		dst.BlockedDomains = src.BlockedDomains
	}
	if len(src.AllowedDomains) > 0 {
		dst.AllowedDomains = src.AllowedDomains
	}
	return dst
}

// mergeCategoryRules merges category rules from src into dst.
func mergeCategoryRules(dst, src *CategoryRulesConfig) *CategoryRulesConfig {
	if len(src.BlockedCategories) > 0 {
		dst.BlockedCategories = src.BlockedCategories
	}
	if len(src.AllowedCategories) > 0 {
		dst.AllowedCategories = src.AllowedCategories
	}
	return dst
}

// ReceiverConfig converts CTVVastConfig to ReceiverConfig with defaults applied.
// Default values:
//   - VastVersionDefault: "3.0"
//   - DefaultCurrency: "USD"
//   - MaxAdsInPod: 10
//   - CollisionPolicy: "VAST_WINS"
//   - Receiver: "GAM_SSU"
//   - SelectionStrategy: "max_revenue"
func (cfg CTVVastConfig) ReceiverConfig() ReceiverConfig {
	rc := ReceiverConfig{}

	// Apply receiver with default
	if cfg.Receiver != "" {
		rc.Receiver = ReceiverType(cfg.Receiver)
	} else {
		rc.Receiver = ReceiverType(DefaultReceiver)
	}

	// Apply currency with default
	if cfg.DefaultCurrency != "" {
		rc.DefaultCurrency = cfg.DefaultCurrency
	} else {
		rc.DefaultCurrency = DefaultCurrency
	}

	// Apply VAST version with default
	if cfg.VastVersionDefault != "" {
		rc.VastVersionDefault = cfg.VastVersionDefault
	} else {
		rc.VastVersionDefault = DefaultVastVersion
	}

	// Apply max ads in pod with default
	if cfg.MaxAdsInPod != 0 {
		rc.MaxAdsInPod = cfg.MaxAdsInPod
	} else {
		rc.MaxAdsInPod = DefaultMaxAdsInPod
	}

	// Apply selection strategy with default
	if cfg.SelectionStrategy != "" {
		rc.SelectionStrategy = SelectionStrategy(cfg.SelectionStrategy)
	} else {
		rc.SelectionStrategy = SelectionStrategy(DefaultSelectionStrategy)
	}

	// Apply collision policy with default
	if cfg.CollisionPolicy != "" {
		rc.CollisionPolicy = CollisionPolicy(cfg.CollisionPolicy)
	} else {
		rc.CollisionPolicy = CollisionPolicy(DefaultCollisionPolicy)
	}

	// Apply allow skeleton vast flag
	if cfg.AllowSkeletonVast != nil {
		rc.AllowSkeletonVast = *cfg.AllowSkeletonVast
	}

	// Apply debug flag
	if cfg.Debug != nil {
		rc.Debug = *cfg.Debug
	}

	// Apply placement rules
	rc.Placement = cfg.buildPlacementRules()

	return rc
}

// buildPlacementRules converts PlacementRulesConfig to PlacementRules.
func (cfg CTVVastConfig) buildPlacementRules() PlacementRules {
	pr := PlacementRules{}

	if cfg.Placement == nil {
		return pr
	}

	if cfg.Placement.Debug != nil {
		pr.Debug = *cfg.Placement.Debug
	}

	// Set placement locations with defaults
	pr.PricingPlacement = cfg.Placement.PricingPlacement
	if pr.PricingPlacement == "" {
		pr.PricingPlacement = PlacementVastPricing
	}
	pr.AdvertiserPlacement = cfg.Placement.AdvertiserPlacement
	if pr.AdvertiserPlacement == "" {
		pr.AdvertiserPlacement = PlacementAdvertiserTag
	}

	// Build pricing rules
	if cfg.Placement.Pricing != nil {
		pr.Pricing = PricingRules{
			Currency: cfg.Placement.Pricing.Currency,
		}
		if cfg.Placement.Pricing.FloorCPM != nil {
			pr.Pricing.FloorCPM = *cfg.Placement.Pricing.FloorCPM
		}
		if cfg.Placement.Pricing.CeilingCPM != nil {
			pr.Pricing.CeilingCPM = *cfg.Placement.Pricing.CeilingCPM
		}
		if pr.Pricing.Currency == "" {
			pr.Pricing.Currency = DefaultCurrency
		}
	}

	// Build advertiser rules
	if cfg.Placement.Advertiser != nil {
		pr.Advertiser = AdvertiserRules{
			BlockedDomains: cfg.Placement.Advertiser.BlockedDomains,
			AllowedDomains: cfg.Placement.Advertiser.AllowedDomains,
		}
	}

	// Build category rules
	if cfg.Placement.Categories != nil {
		pr.Categories = CategoryRules{
			BlockedCategories: cfg.Placement.Categories.BlockedCategories,
			AllowedCategories: cfg.Placement.Categories.AllowedCategories,
		}
	}

	return pr
}

// IsEnabled returns true if the config is enabled. Returns false if Enabled is nil or false.
func (cfg CTVVastConfig) IsEnabled() bool {
	return cfg.Enabled != nil && *cfg.Enabled
}

// boolPtr is a helper function to create a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// float64Ptr is a helper function to create a pointer to a float64 value.
func float64Ptr(f float64) *float64 {
	return &f
}
