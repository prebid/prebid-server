package privacy

import "github.com/mxmCherry/openrtb/v15/openrtb2"

// Enforcement represents the privacy policies to enforce for an OpenRTB bid request.
type Enforcement struct {
	CCPA    bool
	COPPA   bool
	GDPRGeo bool
	GDPRID  bool
	LMT     bool
}

// Any returns true if at least one privacy policy requires enforcement.
func (e Enforcement) Any() bool {
	return e.CCPA || e.COPPA || e.GDPRGeo || e.GDPRID || e.LMT
}

// Apply cleans personally identifiable information from an OpenRTB bid request.
func (e Enforcement) Apply(bidRequest *openrtb2.BidRequest) {
	e.apply(bidRequest, NewScrubber())
}

func (e Enforcement) apply(bidRequest *openrtb2.BidRequest, scrubber Scrubber) {
	if bidRequest != nil && e.Any() {
		bidRequest.Device = scrubber.ScrubDevice(bidRequest.Device, e.getDeviceIDScrubStrategy(), e.getIPv4ScrubStrategy(), e.getIPv6ScrubStrategy(), e.getGeoScrubStrategy())
		bidRequest.User = scrubber.ScrubUser(bidRequest.User, e.getUserScrubStrategy(), e.getGeoScrubStrategy())
	}
}

func (e Enforcement) getDeviceIDScrubStrategy() ScrubStrategyDeviceID {
	if e.COPPA || e.GDPRID || e.CCPA || e.LMT {
		return ScrubStrategyDeviceIDAll
	}

	return ScrubStrategyDeviceIDNone
}

func (e Enforcement) getIPv4ScrubStrategy() ScrubStrategyIPV4 {
	if e.COPPA || e.GDPRGeo || e.CCPA || e.LMT {
		return ScrubStrategyIPV4Lowest8
	}

	return ScrubStrategyIPV4None
}

func (e Enforcement) getIPv6ScrubStrategy() ScrubStrategyIPV6 {
	if e.COPPA {
		return ScrubStrategyIPV6Lowest32
	}

	if e.GDPRGeo || e.CCPA || e.LMT {
		return ScrubStrategyIPV6Lowest16
	}

	return ScrubStrategyIPV6None
}

func (e Enforcement) getGeoScrubStrategy() ScrubStrategyGeo {
	if e.COPPA {
		return ScrubStrategyGeoFull
	}

	if e.GDPRGeo || e.CCPA || e.LMT {
		return ScrubStrategyGeoReducedPrecision
	}

	return ScrubStrategyGeoNone
}

func (e Enforcement) getUserScrubStrategy() ScrubStrategyUser {
	if e.COPPA {
		return ScrubStrategyUserIDAndDemographic
	}

	if e.CCPA || e.LMT {
		return ScrubStrategyUserID
	}

	if e.GDPRID {
		return ScrubStrategyUserID
	}

	return ScrubStrategyUserNone
}
