package bidadjustment

import (
	"math"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	AdjustmentTypeCpm        = "cpm"
	AdjustmentTypeMultiplier = "multiplier"
	AdjustmentTypeStatic     = "static"
	WildCard                 = "*"
	Delimiter                = "|"
)

const pricePrecision float64 = 10000 // Rounds to 4 Decimal Places

func apply(adjustments []openrtb_ext.Adjustment, bidPrice float64, currency string, reqInfo *adapters.ExtraRequestInfo) (float64, string) {
	if adjustments == nil {
		return bidPrice, currency
	}
	originalBidPrice := bidPrice
	originalCurrency := currency

	for _, adjustment := range adjustments {
		switch adjustment.Type {
		case AdjustmentTypeMultiplier:
			bidPrice = bidPrice * adjustment.Value
		case AdjustmentTypeCpm:
			convertedVal, err := reqInfo.ConvertCurrency(adjustment.Value, adjustment.Currency, currency) // Convert Adjustment to Bid Currency
			if err != nil {
				return originalBidPrice, currency
			}
			bidPrice = bidPrice - convertedVal
		case AdjustmentTypeStatic:
			bidPrice = adjustment.Value
			currency = adjustment.Currency
		}
	}
	roundedBidPrice := math.Round(bidPrice*pricePrecision) / pricePrecision

	if roundedBidPrice <= 0 {
		return originalBidPrice, originalCurrency
	}
	return roundedBidPrice, currency
}

func Process(req *openrtb_ext.RequestWrapper, acctBidAdjs *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
	mergedBidAdj, err := merge(req, acctBidAdjs)
	if err != nil {
		return nil, err
	}
	if !Validate(mergedBidAdj) {
		mergedBidAdj = nil
	}
	return mergedBidAdj, err
}

func merge(req *openrtb_ext.RequestWrapper, acctBidAdjs *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return nil, err
	}
	extPrebid := reqExt.GetPrebid()

	if extPrebid == nil && acctBidAdjs == nil {
		return nil, nil
	}
	if extPrebid == nil && acctBidAdjs != nil {
		return acctBidAdjs, nil
	}
	if extPrebid != nil && acctBidAdjs == nil {
		return extPrebid.BidAdjustments, nil
	}

	if extPrebid.BidAdjustments.MediaType.Banner != nil && acctBidAdjs.MediaType.Banner != nil {
		extPrebid.BidAdjustments.MediaType.Banner = mergeAdjustmentsForMediaType(extPrebid.BidAdjustments.MediaType.Banner, acctBidAdjs.MediaType.Banner)
	} else if acctBidAdjs.MediaType.Banner != nil {
		extPrebid.BidAdjustments.MediaType.Banner = acctBidAdjs.MediaType.Banner
	}

	if extPrebid.BidAdjustments.MediaType.Video != nil && acctBidAdjs.MediaType.Video != nil {
		extPrebid.BidAdjustments.MediaType.Video = mergeAdjustmentsForMediaType(extPrebid.BidAdjustments.MediaType.Video, acctBidAdjs.MediaType.Video)
	} else if acctBidAdjs.MediaType.Video != nil {
		extPrebid.BidAdjustments.MediaType.Video = acctBidAdjs.MediaType.Video
	}

	if extPrebid.BidAdjustments.MediaType.Native != nil && acctBidAdjs.MediaType.Native != nil {
		extPrebid.BidAdjustments.MediaType.Native = mergeAdjustmentsForMediaType(extPrebid.BidAdjustments.MediaType.Native, acctBidAdjs.MediaType.Native)
	} else if acctBidAdjs.MediaType.Native != nil {
		extPrebid.BidAdjustments.MediaType.Native = acctBidAdjs.MediaType.Native
	}

	if extPrebid.BidAdjustments.MediaType.Audio != nil && acctBidAdjs.MediaType.Audio != nil {
		extPrebid.BidAdjustments.MediaType.Audio = mergeAdjustmentsForMediaType(extPrebid.BidAdjustments.MediaType.Audio, acctBidAdjs.MediaType.Audio)
	} else if acctBidAdjs.MediaType.Audio != nil {
		extPrebid.BidAdjustments.MediaType.Audio = acctBidAdjs.MediaType.Audio
	}

	if extPrebid.BidAdjustments.MediaType.WildCard != nil && acctBidAdjs.MediaType.WildCard != nil {
		extPrebid.BidAdjustments.MediaType.WildCard = mergeAdjustmentsForMediaType(extPrebid.BidAdjustments.MediaType.WildCard, acctBidAdjs.MediaType.WildCard)
	} else if acctBidAdjs.MediaType.WildCard != nil {
		extPrebid.BidAdjustments.MediaType.WildCard = acctBidAdjs.MediaType.WildCard
	}
	return extPrebid.BidAdjustments, nil
}

func mergeAdjustmentsForMediaType(reqAdjMap map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID, accountAdjMap map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID) map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID {
	for bidderName, dealIdToAdjustmentsMap := range accountAdjMap {
		if _, ok := reqAdjMap[bidderName]; ok {
			for dealID, acctAdjustmentsArray := range accountAdjMap[bidderName] {
				if _, okay := reqAdjMap[bidderName][dealID]; !okay {
					reqAdjMap[bidderName][dealID] = acctAdjustmentsArray
				}
			}
		} else {
			reqAdjMap[bidderName] = dealIdToAdjustmentsMap
		}
	}
	return reqAdjMap
}

func GenerateMap(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments) map[string][]openrtb_ext.Adjustment {
	if bidAdjustments == nil {
		return nil
	}
	ruleToAdjustmentMap := make(map[string][]openrtb_ext.Adjustment)
	ruleToAdjustmentMap = populateMapForMediaType(bidAdjustments.MediaType.Banner, string(openrtb_ext.BidTypeBanner), ruleToAdjustmentMap)
	ruleToAdjustmentMap = populateMapForMediaType(bidAdjustments.MediaType.Video, string(openrtb_ext.BidTypeVideo), ruleToAdjustmentMap)
	ruleToAdjustmentMap = populateMapForMediaType(bidAdjustments.MediaType.Audio, string(openrtb_ext.BidTypeAudio), ruleToAdjustmentMap)
	ruleToAdjustmentMap = populateMapForMediaType(bidAdjustments.MediaType.Native, string(openrtb_ext.BidTypeNative), ruleToAdjustmentMap)
	ruleToAdjustmentMap = populateMapForMediaType(bidAdjustments.MediaType.WildCard, WildCard, ruleToAdjustmentMap)

	return ruleToAdjustmentMap
}

func populateMapForMediaType(bidAdj map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID, mediaType string, ruleToAdjustmentMap map[string][]openrtb_ext.Adjustment) map[string][]openrtb_ext.Adjustment {
	for bidderName := range bidAdj {
		for dealID, adjustments := range bidAdj[bidderName] {
			rule := mediaType + Delimiter + string(bidderName) + Delimiter + dealID
			ruleToAdjustmentMap[rule] = adjustments
		}
	}
	return ruleToAdjustmentMap
}
