package ortb

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	DefaultPriceGranularityPrecision  = 2
	DefaultTargetingIncludeWinners    = true
	DefaultTargetingIncludeBidderKeys = true
	DefaultSecure                     = int8(1)
)

func SetDefaults(r *openrtb_ext.RequestWrapper, defaultTmax int) error {
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

	if r.TMax == 0 {
		r.TMax = int64(defaultTmax)
	}

	return nil
}

func setDefaultsTargeting(targeting *openrtb_ext.ExtRequestTargeting) bool {
	if targeting == nil {
		return false
	}

	modified := false

	if newPG, updated := setDefaultsPriceGranularity(targeting.PriceGranularity); updated {
		modified = true
		targeting.PriceGranularity = newPG
	}

	// If price granularity is not specified in request then default one should be set.
	// Default price granularity can be overwritten for video, banner or native bid type
	// only in case targeting.MediaTypePriceGranularity.Video|Banner|Native != nil.

	if targeting.MediaTypePriceGranularity != nil {
		if targeting.MediaTypePriceGranularity.Video != nil {
			if newVideoPG, updated := setDefaultsPriceGranularity(targeting.MediaTypePriceGranularity.Video); updated {
				modified = true
				targeting.MediaTypePriceGranularity.Video = newVideoPG
			}
		}
		if targeting.MediaTypePriceGranularity.Banner != nil {
			if newBannerPG, updated := setDefaultsPriceGranularity(targeting.MediaTypePriceGranularity.Banner); updated {
				modified = true
				targeting.MediaTypePriceGranularity.Banner = newBannerPG
			}
		}
		if targeting.MediaTypePriceGranularity.Native != nil {
			if newNativePG, updated := setDefaultsPriceGranularity(targeting.MediaTypePriceGranularity.Native); updated {
				modified = true
				targeting.MediaTypePriceGranularity.Native = newNativePG
			}
		}
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

func setDefaultsPriceGranularity(pg *openrtb_ext.PriceGranularity) (*openrtb_ext.PriceGranularity, bool) {
	if pg == nil || len(pg.Ranges) == 0 {
		pg = ptrutil.ToPtr(openrtb_ext.NewPriceGranularityDefault())
		return pg, true
	}

	modified := false

	if pg.Precision == nil {
		pg.Precision = ptrutil.ToPtr(DefaultPriceGranularityPrecision)
		modified = true
	}

	if setDefaultsPriceGranularityRange(pg.Ranges) {
		modified = true
	}

	return pg, modified
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
