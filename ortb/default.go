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
		return err
	}

	requestExtPrebid := requestExt.GetPrebid()
	if requestExtPrebid != nil {
		hasChanges := setDefaultsTargeting(requestExtPrebid.Targeting)

		if hasChanges {
			requestExt.SetPrebid(requestExtPrebid)
		}
	}

	return nil
}

func setDefaultsTargeting(targeting *openrtb_ext.ExtRequestTargeting) bool {
	if targeting == nil {
		return false
	}

	hasChanges := false

	if targeting.PriceGranularity == nil {
		targeting.PriceGranularity = ptrutil.ToPtr(openrtb_ext.NewPriceGranularityDefault())
		hasChanges = true
	} else {
		if targeting.PriceGranularity.Precision == nil {
			targeting.PriceGranularity.Precision = ptrutil.ToPtr(DefaultPriceGranularityPrecision)
			hasChanges = true
		}
		hasChanges = hasChanges || setDefaultsPriceGranularityRange(targeting.PriceGranularity.Ranges)
	}

	if targeting.IncludeWinners == nil {
		targeting.IncludeWinners = ptrutil.ToPtr(DefaultTargetingIncludeWinners)
		hasChanges = true
	}

	if targeting.IncludeBidderKeys == nil {
		targeting.IncludeBidderKeys = ptrutil.ToPtr(DefaultTargetingIncludeBidderKeys)
		hasChanges = true
	}

	return hasChanges
}

func setDefaultsPriceGranularityRange(ranges []openrtb_ext.GranularityRange) bool {
	hasChanges := false

	var prevMax float64 = 0
	for i, r := range ranges {
		if ranges[i].Min != prevMax {
			ranges[i].Min = prevMax
			hasChanges = true
		}
		prevMax = r.Max
	}

	return hasChanges
}
