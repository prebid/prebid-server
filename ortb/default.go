package ortb

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/ptrutil"
)

const (
	DefaultPriceGranularityPrecision  = 2
	DefaultTargetingIncludeWinners    = true
	DefaultTargetingIncludeBidderKeys = true
)

func SetDefaults(r *openrtb_ext.RequestWrapper) error {
	requestExt, err := r.GetRequestExt()
	if err != nil {
		return nil
	}

	requestExtPrebid := requestExt.GetPrebid()
	if requestExtPrebid != nil {
		setDefaultsTargeting(requestExtPrebid.Targeting)
	}

	return nil
}

func setDefaultsTargeting(targeting *openrtb_ext.ExtRequestTargeting) {
	if targeting == nil {
		return
	}

	if targeting.PriceGranularity == nil {
		targeting.PriceGranularity = ptrutil.ToPtr(openrtb_ext.NewPriceGranularityDefault())
	} else {
		if targeting.PriceGranularity.Precision == nil {
			targeting.PriceGranularity.Precision = ptrutil.ToPtr(DefaultPriceGranularityPrecision)
		}
		setDefaultsPriceGranularityRange(targeting.PriceGranularity.Ranges)
	}

	if targeting.IncludeWinners == nil {
		targeting.IncludeWinners = ptrutil.ToPtr(DefaultTargetingIncludeWinners)
	}

	if targeting.IncludeBidderKeys == nil {
		targeting.IncludeBidderKeys = ptrutil.ToPtr(DefaultTargetingIncludeBidderKeys)
	}

}

func setDefaultsPriceGranularityRange(ranges []openrtb_ext.GranularityRange) {
	var prevMax float64 = 0
	for i, r := range ranges {
		ranges[i].Min = prevMax
		prevMax = r.Max
	}
}
