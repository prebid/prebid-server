package ortb

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/ptrutil"
)

const (
	DefaultPriceGranularityPrecision  = 2
	DefaultTargetingIncludeWinners    = true
	DefaultTargetingIncludeBidderKeys = true
	DefaultSecure                     = int8(1)
)

func SetDefaults(r *openrtb_ext.RequestWrapper) error {
	requestExt, err := r.GetRequestExt()
	if err != nil {
		return err
	}

	requestExtPrebid := requestExt.GetPrebid()
	if requestExtPrebid != nil {
		modified := setDefaultsTargeting(requestExtPrebid.Targeting)

		if modified {
			requestExt.SetPrebid(requestExtPrebid)
		}
	}

	imps := r.GetImp()
	if len(imps) > 0 {
		modified := setDefaultsImp(imps)

		if modified {
			r.SetImp(imps)
		}
	}

	return nil
}

func setDefaultsTargeting(targeting *openrtb_ext.ExtRequestTargeting) bool {
	if targeting == nil {
		return false
	}

	modified := false

	if targeting.PriceGranularity == nil || len(targeting.PriceGranularity.Ranges) == 0 {
		targeting.PriceGranularity = ptrutil.ToPtr(openrtb_ext.NewPriceGranularityDefault())
		modified = true
	} else if setDefaultsPriceGranularity(targeting.PriceGranularity) {
		modified = true
	}

	if targeting.IncludeWinners == nil {
		targeting.IncludeWinners = ptrutil.ToPtr(DefaultTargetingIncludeWinners)
		modified = true
	}

	if targeting.IncludeBidderKeys == nil {
		targeting.IncludeBidderKeys = ptrutil.ToPtr(DefaultTargetingIncludeBidderKeys)
		modified = true
	}

	return modified
}

func setDefaultsPriceGranularity(pg *openrtb_ext.PriceGranularity) bool {
	modified := false

	if pg.Precision == nil {
		pg.Precision = ptrutil.ToPtr(DefaultPriceGranularityPrecision)
		modified = true
	}

	if setDefaultsPriceGranularityRange(pg.Ranges) {
		modified = true
	}

	return modified
}

func setDefaultsPriceGranularityRange(ranges []openrtb_ext.GranularityRange) bool {
	modified := false

	var prevMax float64 = 0
	for i, r := range ranges {
		if ranges[i].Min != prevMax {
			ranges[i].Min = prevMax
			modified = true
		}
		prevMax = r.Max
	}

	return modified
}

func setDefaultsImp(imps []*openrtb_ext.ImpWrapper) bool {
	modified := false

	for _, i := range imps {
		if i != nil && i.Imp != nil && i.Secure == nil {
			i.Secure = ptrutil.ToPtr(DefaultSecure)
			modified = true
		}
	}

	return modified
}
