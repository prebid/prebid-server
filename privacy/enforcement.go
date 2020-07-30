package privacy

import (
	"github.com/mxmCherry/openrtb"
)

// Enforcement represents the privacy policies to enforce for an OpenRTB bid request.
type Enforcement struct {
	CCPA    bool
	COPPA   bool
	GDPR    bool
	GDPRGeo bool
	LMT     bool
}

// Any returns true if at least one privacy policy requires enforcement.
func (e Enforcement) Any() bool {
	return e.CCPA || e.COPPA || e.GDPR || e.GDPRGeo || e.LMT
}

// Apply cleans personally identifiable information from an OpenRTB bid request.
func (e Enforcement) Apply(bidRequest *openrtb.BidRequest, ampGDPRException bool) {
	e.apply(bidRequest, ampGDPRException, NewScrubber())
}

func (e Enforcement) apply(bidRequest *openrtb.BidRequest, ampGDPRException bool, scrubber Scrubber) {
	if bidRequest != nil && e.Any() {
		bidRequest.Device = scrubber.ScrubDevice(bidRequest.Device, e.getIPv6ScrubStrategy(), e.getGeoScrubStrategy())
		bidRequest.User = scrubber.ScrubUser(bidRequest.User, e.getUserScrubStrategy(ampGDPRException), e.getGeoScrubStrategy())
	}
}

func (e Enforcement) getIPv6ScrubStrategy() ScrubStrategyIPV6 {
	if e.COPPA {
		return ScrubStrategyIPV6Lowest32
	}

	if e.GDPR || e.CCPA || e.LMT {
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

func (e Enforcement) getUserScrubStrategy(ampGDPRException bool) ScrubStrategyUser {
	if e.COPPA {
		return ScrubStrategyUserIDAndDemographic
	}

	if e.GDPR && ampGDPRException {
		return ScrubStrategyUserNone
	}

	// If no user scrubbing is needed, then return none, else scrub ID (COPPA checked above)
	if e.CCPA || e.GDPR || e.LMT {
		return ScrubStrategyUserID
	}

	return ScrubStrategyUserNone
}
