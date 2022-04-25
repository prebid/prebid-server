package consentconstants

import base "github.com/prebid/go-gdpr/consentconstants"

// TCF 2.0 Special Features:
const (
	// Use precise geolocation data to select and deliver an ad in the moment, without storing it.
	Geolocation base.SpecialFeature = 1

	// Identify a device by actively scanning device characteristics in order to select an ad in the moment.
	DeviceScan base.SpecialFeature = 2
)
