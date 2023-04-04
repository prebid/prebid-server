package config

import "github.com/prebid/prebid-server/util/randomutil"

type Adapter struct {
	Endpoint         string
	ExtraAdapterInfo string

	// needed for AppNexus
	RandomGenerator randomutil.RandomGenerator

	// needed for Rubicon
	XAPI AdapterXAPI

	// needed for Facebook
	PlatformID string
	AppSecret  string
}
