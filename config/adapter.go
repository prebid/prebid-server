package config

type Adapter struct {
	Endpoint         string
	ExtraAdapterInfo string

	// needed for Rubicon
	XAPI AdapterXAPI `mapstructure:"xapi"`

	// needed for Facebook
	PlatformID string `mapstructure:"platform_id"`
	AppSecret  string `mapstructure:"app_secret"`
}
