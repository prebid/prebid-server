package config

// Account represents a publisher account configuration
type Account struct {
	ID            string      `mapstructure:"id" json:"id"`
	Disabled      bool        `mapstructure:"disabled" json:"disabled"`
	CacheTTL      DefaultTTLs `mapstructure:"cache_ttl" json:"cache_ttl"`
	EventsEnabled bool        `mapstructure:"events_enabled" json:"events_enabled"`
	GDPR          AccountGDPR `mapstructure:"gdpr" json:"gdpr"`
}

// AccountGDPR represents account-specific GDPR configuration
type AccountGDPR struct {
	Enabled            *bool                  `mapstructure:"enabled" json:"enabled,omitempty"`
	IntegrationEnabled AccountGDPRIntegration `mapstructure:"integration_enabled" json:"integration_enabled"`
}

// AccountGDPRIntegration indicates whether GDPR is enabled for each request type
type AccountGDPRIntegration struct {
	AMP   *bool `mapstructure:"amp"   json:"amp,omitempty"`
	App   *bool `mapstructure:"app"   json:"app,omitempty"`
	Video *bool `mapstructure:"video" json:"video,omitempty"`
	Web   *bool `mapstructure:"web"   json:"web,omitempty"`
}
