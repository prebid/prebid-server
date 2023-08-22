package privacy

import (
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/config"
)

// Enforcement represents the privacy policies to enforce for an OpenRTB bid request.
type Enforcement struct {
	CCPA    bool
	COPPA   bool
	GDPRGeo bool
	GDPRID  bool
	LMT     bool

	// activities
	UFPD       bool
	Eids       bool
	PreciseGeo bool
	TID        bool
}

// Any returns true if at least one privacy policy requires enforcement.
func (e Enforcement) AnyLegacy() bool {
	return e.CCPA || e.COPPA || e.GDPRGeo || e.GDPRID || e.LMT
}

func (e Enforcement) AnyActivities() bool {
	return e.UFPD || e.PreciseGeo || e.Eids || e.TID
}

// Apply cleans personally identifiable information from an OpenRTB bid request.
func (e Enforcement) Apply(bidRequest *openrtb2.BidRequest, privacy config.AccountPrivacy) {
	e.apply(bidRequest, NewScrubber(privacy.IPv6Config, privacy.IPv4Config))
}

func (e Enforcement) apply(bidRequest *openrtb2.BidRequest, scrubber Scrubber) {
	if bidRequest != nil {
		if e.AnyActivities() {
			bidRequest = scrubber.ScrubRequest(bidRequest, e)
		}
		if e.AnyLegacy() && !(e.UFPD && e.PreciseGeo && e.Eids) {
			bidRequest.User = scrubber.ScrubUser(bidRequest.User, e.getUserScrubStrategy(), e.getGeoScrubStrategy())
		}
		if e.AnyLegacy() && !(e.UFPD && e.PreciseGeo) {
			bidRequest.Device = scrubber.ScrubDevice(bidRequest.Device, e.getDeviceIDScrubStrategy(), e.getIPv4ScrubStrategy(), e.getIPv6ScrubStrategy(), e.getGeoScrubStrategy())
		}
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
	if e.GDPRGeo || e.CCPA || e.LMT || e.COPPA {
		return ScrubStrategyIPV6Subnet
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
	if e.COPPA || e.CCPA || e.LMT || e.GDPRID {
		return ScrubStrategyUserIDAndDemographic
	}

	return ScrubStrategyUserNone
}
