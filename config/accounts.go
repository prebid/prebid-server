package config

// Account represents a publisher account configuration
type Account struct {
	ID       string      `mapstructure:"id" json:"id"`
	Disabled bool        `mapstructure:"disabled" json:"disabled"`
	CacheTTL DefaultTTLs `mapstructure:"cache_ttl" json:"cache_ttl"`
}
