package config

type Adapter struct {
	Endpoint         string
	ExtraAdapterInfo string

	// needed for Rubicon
	XAPI AdapterXAPI

	// needed for Facebook
	PlatformID string
	AppSecret  string
}
