package privacy

import (
	"github.com/mxmCherry/openrtb"
)

// Enforcement represents the privacy policies to enforce for an OpenRTB bid request.
type Enforcement struct {
	CCPA  bool
	COPPA bool
	GDPR  bool
}

// Any returns true if at least one privacy policy requires enforcement.
func (e Enforcement) Any() bool {
	return e.CCPA || e.COPPA || e.GDPR
}

// Apply cleans personally identifiable information from an OpenRTB bid request.
func (e Enforcement) Apply(bidRequest *openrtb.BidRequest, isAMP bool) {
	e.apply(bidRequest, isAMP, NewScrubber())
}

func (e Enforcement) apply(bidRequest *openrtb.BidRequest, isAMP bool, scrubber Scrubber) {
	if bidRequest != nil && e.Any() {
		bidRequest.Device = scrubber.ScrubDevice(bidRequest.Device, e.getDeviceMacAndIFA(), e.getIPv6ScrubStrategy(), e.getGeoScrubStrategy())
		bidRequest.User = scrubber.ScrubUser(bidRequest.User, e.getUserScrubStrategy(isAMP), e.getGeoScrubStrategy())
	}
}

func (e Enforcement) getDeviceMacAndIFA() bool {
	return e.COPPA
}

func (e Enforcement) getIPv6ScrubStrategy() ScrubStrategyIPV6 {
	if e.COPPA {
		return ScrubStrategyIPV6Lowest32
	}

	if e.GDPR || e.CCPA {
		return ScrubStrategyIPV6Lowest16
	}

	return ScrubStrategyIPV6None
}

func (e Enforcement) getGeoScrubStrategy() ScrubStrategyGeo {
	if e.COPPA {
		return ScrubStrategyGeoFull
	}

	if e.GDPR || e.CCPA {
		return ScrubStrategyGeoReducedPrecision
	}

	return ScrubStrategyGeoNone
}

func (e Enforcement) getUserScrubStrategy(isAMP bool) ScrubStrategyUser {
	if e.COPPA {
		return ScrubStrategyUserFull
	}

	// There's no way for AMP to send a GDPR consent string yet so it's hard
	// to know if the vendor is consented or not and therefore for AMP requests
	// we keep the BuyerUID as is for GDPR.
	if e.GDPR && isAMP {
		return ScrubStrategyUserNone
	}

	if e.GDPR || e.CCPA {
		return ScrubStrategyUserBuyerIDOnly
	}

	return ScrubStrategyUserNone
}
