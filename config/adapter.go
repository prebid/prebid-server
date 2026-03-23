package config

type Adapter struct {
	Endpoint         string
	ExtraAdapterInfo string

	// needed for Rubicon
	XAPI AdapterXAPI

	// needed for AppNexus and Facebook
	PlatformID string

	// nededed for Facebook
	AppSecret string
}
