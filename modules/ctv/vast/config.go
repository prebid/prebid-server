package vast

// CTVVastConfig contains configuration for the CTV VAST module
// It supports layered configuration: host -> account -> profile
type CTVVastConfig struct {
	// Enabled controls whether the CTV VAST module is active
	Enabled bool `json:"enabled"`
	// Receiver identifies the target player (e.g., "GAM_SSU", "GENERIC")
	Receiver string `json:"receiver"`
	// DefaultCurrency is used when bid currency is not specified
	DefaultCurrency string `json:"default_currency"`
	// VastVersionDefault is the VAST version to use (e.g., "3.0", "4.0")
	VastVersionDefault string `json:"vast_version_default"`
	// MaxAdsInPod is the maximum number of ads to include in a pod
	MaxAdsInPod int `json:"max_ads_in_pod"`
	// SelectionStrategy determines how bids are selected ("SINGLE", "TOP_N")
	SelectionStrategy string `json:"selection_strategy"`
	// CollisionPolicy handles conflicts when both VAST and OpenRTB have same field
	CollisionPolicy string `json:"collision_policy"`
	// EnableDebug includes debug information in VAST extensions
	EnableDebug bool `json:"enable_debug"`
	// AllowSkeletonVast allows bids without adm field (skeleton VAST)
	AllowSkeletonVast bool `json:"allow_skeleton_vast"`
	// PlacementRules controls where metadata is placed in VAST
	PlacementRules *PlacementRulesConfig `json:"placement_rules,omitempty"`
}

// PlacementRulesConfig controls where specific metadata fields are placed in VAST
type PlacementRulesConfig struct {
	// PricingPlacement controls where pricing data goes
	PricingPlacement string `json:"pricing_placement"`
	// AdvertiserPlacement controls where advertiser domain goes
	AdvertiserPlacement string `json:"advertiser_placement"`
	// CategoriesPlacement controls where IAB categories go
	CategoriesPlacement string `json:"categories_placement"`
	// DebugPlacement controls where debug info goes
	DebugPlacement string `json:"debug_placement"`
}

// MergeCTVVastConfig merges configuration with PBS-style precedence
// Non-zero values in profile override account, account overrides host
// Returns merged configuration with proper precedence
func MergeCTVVastConfig(host, account, profile *CTVVastConfig) CTVVastConfig {
	// Start with defaults
	result := CTVVastConfig{
		Enabled:            false,
		Receiver:           "",
		DefaultCurrency:    "",
		VastVersionDefault: "",
		MaxAdsInPod:        0,
		SelectionStrategy:  "",
		CollisionPolicy:    "",
		EnableDebug:        false,
		AllowSkeletonVast:  false,
		PlacementRules:     nil,
	}

	// Apply host config
	if host != nil {
		if host.Enabled {
			result.Enabled = host.Enabled
		}
		if host.Receiver != "" {
			result.Receiver = host.Receiver
		}
		if host.DefaultCurrency != "" {
			result.DefaultCurrency = host.DefaultCurrency
		}
		if host.VastVersionDefault != "" {
			result.VastVersionDefault = host.VastVersionDefault
		}
		if host.MaxAdsInPod > 0 {
			result.MaxAdsInPod = host.MaxAdsInPod
		}
		if host.SelectionStrategy != "" {
			result.SelectionStrategy = host.SelectionStrategy
		}
		if host.CollisionPolicy != "" {
			result.CollisionPolicy = host.CollisionPolicy
		}
		if host.EnableDebug {
			result.EnableDebug = host.EnableDebug
		}
		if host.AllowSkeletonVast {
			result.AllowSkeletonVast = host.AllowSkeletonVast
		}
		if host.PlacementRules != nil {
			result.PlacementRules = copyPlacementRules(host.PlacementRules)
		}
	}

	// Apply account config (overrides host)
	if account != nil {
		if account.Enabled {
			result.Enabled = account.Enabled
		}
		if account.Receiver != "" {
			result.Receiver = account.Receiver
		}
		if account.DefaultCurrency != "" {
			result.DefaultCurrency = account.DefaultCurrency
		}
		if account.VastVersionDefault != "" {
			result.VastVersionDefault = account.VastVersionDefault
		}
		if account.MaxAdsInPod > 0 {
			result.MaxAdsInPod = account.MaxAdsInPod
		}
		if account.SelectionStrategy != "" {
			result.SelectionStrategy = account.SelectionStrategy
		}
		if account.CollisionPolicy != "" {
			result.CollisionPolicy = account.CollisionPolicy
		}
		if account.EnableDebug {
			result.EnableDebug = account.EnableDebug
		}
		if account.AllowSkeletonVast {
			result.AllowSkeletonVast = account.AllowSkeletonVast
		}
		if account.PlacementRules != nil {
			result.PlacementRules = mergePlacementRules(result.PlacementRules, account.PlacementRules)
		}
	}

	// Apply profile config (overrides account)
	if profile != nil {
		if profile.Enabled {
			result.Enabled = profile.Enabled
		}
		if profile.Receiver != "" {
			result.Receiver = profile.Receiver
		}
		if profile.DefaultCurrency != "" {
			result.DefaultCurrency = profile.DefaultCurrency
		}
		if profile.VastVersionDefault != "" {
			result.VastVersionDefault = profile.VastVersionDefault
		}
		if profile.MaxAdsInPod > 0 {
			result.MaxAdsInPod = profile.MaxAdsInPod
		}
		if profile.SelectionStrategy != "" {
			result.SelectionStrategy = profile.SelectionStrategy
		}
		if profile.CollisionPolicy != "" {
			result.CollisionPolicy = profile.CollisionPolicy
		}
		if profile.EnableDebug {
			result.EnableDebug = profile.EnableDebug
		}
		if profile.AllowSkeletonVast {
			result.AllowSkeletonVast = profile.AllowSkeletonVast
		}
		if profile.PlacementRules != nil {
			result.PlacementRules = mergePlacementRules(result.PlacementRules, profile.PlacementRules)
		}
	}

	return result
}

// ReceiverConfig converts CTVVastConfig to ReceiverConfig with proper defaults
func (cfg CTVVastConfig) ReceiverConfig() ReceiverConfig {
	// Apply defaults
	receiver := cfg.Receiver
	if receiver == "" {
		receiver = "GENERIC"
	}

	defaultCurrency := cfg.DefaultCurrency
	if defaultCurrency == "" {
		defaultCurrency = "USD"
	}

	vastVersion := cfg.VastVersionDefault
	if vastVersion == "" {
		vastVersion = "3.0"
	}

	maxAdsInPod := cfg.MaxAdsInPod
	if maxAdsInPod == 0 {
		maxAdsInPod = 10
	}

	selectionStrategy := cfg.SelectionStrategy
	if selectionStrategy == "" {
		selectionStrategy = "SINGLE"
	}

	collisionPolicy := CollisionPolicy(cfg.CollisionPolicy)
	if collisionPolicy == "" {
		collisionPolicy = CollisionPolicyVastWins
	}

	// Build placement rules with defaults
	placementRules := buildPlacementRules(cfg.PlacementRules)

	return ReceiverConfig{
		Receiver:           receiver,
		DefaultCurrency:    defaultCurrency,
		VastVersionDefault: vastVersion,
		MaxAdsInPod:        maxAdsInPod,
		SelectionStrategy:  selectionStrategy,
		CollisionPolicy:    collisionPolicy,
		PlacementRules:     placementRules,
		EnableDebug:        cfg.EnableDebug,
		AllowSkeletonVast:  cfg.AllowSkeletonVast,
	}
}

// copyPlacementRules creates a deep copy of placement rules
func copyPlacementRules(src *PlacementRulesConfig) *PlacementRulesConfig {
	if src == nil {
		return nil
	}
	return &PlacementRulesConfig{
		PricingPlacement:    src.PricingPlacement,
		AdvertiserPlacement: src.AdvertiserPlacement,
		CategoriesPlacement: src.CategoriesPlacement,
		DebugPlacement:      src.DebugPlacement,
	}
}

// mergePlacementRules merges placement rules with override precedence
func mergePlacementRules(base, override *PlacementRulesConfig) *PlacementRulesConfig {
	if override == nil {
		return base
	}

	result := &PlacementRulesConfig{}
	if base != nil {
		result.PricingPlacement = base.PricingPlacement
		result.AdvertiserPlacement = base.AdvertiserPlacement
		result.CategoriesPlacement = base.CategoriesPlacement
		result.DebugPlacement = base.DebugPlacement
	}

	if override.PricingPlacement != "" {
		result.PricingPlacement = override.PricingPlacement
	}
	if override.AdvertiserPlacement != "" {
		result.AdvertiserPlacement = override.AdvertiserPlacement
	}
	if override.CategoriesPlacement != "" {
		result.CategoriesPlacement = override.CategoriesPlacement
	}
	if override.DebugPlacement != "" {
		result.DebugPlacement = override.DebugPlacement
	}

	return result
}

// buildPlacementRules converts PlacementRulesConfig to PlacementRules with defaults
func buildPlacementRules(cfg *PlacementRulesConfig) PlacementRules {
	rules := PlacementRules{
		PricingPlacement:    PlacementInline,
		AdvertiserPlacement: PlacementInline,
		CategoriesPlacement: PlacementExtensions,
		DebugPlacement:      PlacementExtensions,
	}

	if cfg != nil {
		if cfg.PricingPlacement != "" {
			rules.PricingPlacement = Placement(cfg.PricingPlacement)
		}
		if cfg.AdvertiserPlacement != "" {
			rules.AdvertiserPlacement = Placement(cfg.AdvertiserPlacement)
		}
		if cfg.CategoriesPlacement != "" {
			rules.CategoriesPlacement = Placement(cfg.CategoriesPlacement)
		}
		if cfg.DebugPlacement != "" {
			rules.DebugPlacement = Placement(cfg.DebugPlacement)
		}
	}

	return rules
}
