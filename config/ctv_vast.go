package config

// CTVVast holds CTV VAST endpoint configuration
type CTVVast struct {
	Enabled               bool           `mapstructure:"enabled" json:"enabled"`
	Receiver              string         `mapstructure:"receiver" json:"receiver"` // "GAM_SSU" or "GENERIC"
	VastVersionDefault    string         `mapstructure:"vast_version_default" json:"vast_version_default"`
	DefaultCurrency       string         `mapstructure:"default_currency" json:"default_currency"`
	MaxAdsInPod           int            `mapstructure:"max_ads_in_pod" json:"max_ads_in_pod"`
	SelectionStrategy     string         `mapstructure:"selection_strategy" json:"selection_strategy"` // "SINGLE" or "TOP_N"
	CollisionPolicy       string         `mapstructure:"collision_policy" json:"collision_policy"`     // "VAST_WINS" or "OPENRTB_WINS"
	PlacementRules        PlacementRules `mapstructure:"placement_rules" json:"placement_rules"`
	MacroConfig           MacroConfig    `mapstructure:"macro_config" json:"macro_config"`
	IncludeDebugIDs       bool           `mapstructure:"include_debug_ids" json:"include_debug_ids"`
	StoredRequestsEnabled bool           `mapstructure:"stored_requests_enabled" json:"stored_requests_enabled"`
}

// PlacementRules defines where OpenRTB fields should be placed in VAST
type PlacementRules struct {
	Price      string `mapstructure:"price" json:"price"`           // "INLINE", "EXTENSIONS", or "SKIP"
	Currency   string `mapstructure:"currency" json:"currency"`     // "INLINE", "EXTENSIONS", or "SKIP"
	Advertiser string `mapstructure:"advertiser" json:"advertiser"` // "INLINE", "EXTENSIONS", or "SKIP"
	Categories string `mapstructure:"categories" json:"categories"` // "INLINE", "EXTENSIONS", or "SKIP"
	Duration   string `mapstructure:"duration" json:"duration"`     // "INLINE", "EXTENSIONS", or "SKIP"
	IDs        string `mapstructure:"ids" json:"ids"`               // "INLINE", "EXTENSIONS", or "SKIP"
	DealID     string `mapstructure:"deal_id" json:"deal_id"`       // "INLINE", "EXTENSIONS", or "SKIP"
}

// MacroConfig defines macro expansion configuration
type MacroConfig struct {
	Enabled            bool                    `mapstructure:"enabled" json:"enabled"`
	UnknownMacroPolicy string                  `mapstructure:"unknown_macro_policy" json:"unknown_macro_policy"` // "KEEP", "REMOVE", or "ERROR"
	Mappings           map[string]MacroMapping `mapstructure:"mappings" json:"mappings"`
}

// MacroMapping defines how a macro should be expanded
type MacroMapping struct {
	Source       string `mapstructure:"source" json:"source"`               // "query", "header", "context", or "default"
	Key          string `mapstructure:"key" json:"key"`                     // Key to lookup in source
	DefaultValue string `mapstructure:"default_value" json:"default_value"` // Default if not found
}

// CTVVastDefaults returns default CTV VAST configuration
func CTVVastDefaults() CTVVast {
	return CTVVast{
		Enabled:            false, // Disabled by default
		Receiver:           "GAM_SSU",
		VastVersionDefault: "3.0",
		DefaultCurrency:    "USD",
		MaxAdsInPod:        1,
		SelectionStrategy:  "SINGLE",
		CollisionPolicy:    "VAST_WINS",
		PlacementRules: PlacementRules{
			Price:      "INLINE",
			Currency:   "INLINE",
			Advertiser: "INLINE",
			Categories: "EXTENSIONS",
			Duration:   "INLINE",
			IDs:        "EXTENSIONS",
			DealID:     "EXTENSIONS",
		},
		MacroConfig: MacroConfig{
			Enabled:            true,
			UnknownMacroPolicy: "KEEP",
			Mappings:           make(map[string]MacroMapping),
		},
		IncludeDebugIDs:       false,
		StoredRequestsEnabled: true,
	}
}

// MergeCTVVastConfig merges CTV VAST configurations with priority: profile > account > host
func MergeCTVVastConfig(host, account, profile *CTVVast) CTVVast {
	// Start with host config (or defaults if nil)
	result := CTVVastDefaults()
	if host != nil {
		result = *host
	}

	// Apply account overrides
	if account != nil {
		if account.Receiver != "" {
			result.Receiver = account.Receiver
		}
		if account.VastVersionDefault != "" {
			result.VastVersionDefault = account.VastVersionDefault
		}
		if account.DefaultCurrency != "" {
			result.DefaultCurrency = account.DefaultCurrency
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

		// Merge placement rules (account overrides host)
		mergePlacementRules(&result.PlacementRules, &account.PlacementRules)

		// Merge macro config
		mergeMacroConfig(&result.MacroConfig, &account.MacroConfig)
	}

	// Apply profile overrides (highest priority)
	if profile != nil {
		if profile.Receiver != "" {
			result.Receiver = profile.Receiver
		}
		if profile.VastVersionDefault != "" {
			result.VastVersionDefault = profile.VastVersionDefault
		}
		if profile.DefaultCurrency != "" {
			result.DefaultCurrency = profile.DefaultCurrency
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

		// Merge placement rules (profile overrides account)
		mergePlacementRules(&result.PlacementRules, &profile.PlacementRules)

		// Merge macro config
		mergeMacroConfig(&result.MacroConfig, &profile.MacroConfig)
	}

	return result
}

// mergePlacementRules merges placement rules with override priority
func mergePlacementRules(base *PlacementRules, override *PlacementRules) {
	if override.Price != "" {
		base.Price = override.Price
	}
	if override.Currency != "" {
		base.Currency = override.Currency
	}
	if override.Advertiser != "" {
		base.Advertiser = override.Advertiser
	}
	if override.Categories != "" {
		base.Categories = override.Categories
	}
	if override.Duration != "" {
		base.Duration = override.Duration
	}
	if override.IDs != "" {
		base.IDs = override.IDs
	}
	if override.DealID != "" {
		base.DealID = override.DealID
	}
}

// mergeMacroConfig merges macro configurations
func mergeMacroConfig(base *MacroConfig, override *MacroConfig) {
	if override.UnknownMacroPolicy != "" {
		base.UnknownMacroPolicy = override.UnknownMacroPolicy
	}

	// Merge mappings
	if override.Mappings != nil {
		if base.Mappings == nil {
			base.Mappings = make(map[string]MacroMapping)
		}
		for k, v := range override.Mappings {
			base.Mappings[k] = v
		}
	}
}
