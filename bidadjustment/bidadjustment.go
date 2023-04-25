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
	if adjustments == nil || len(adjustments) == 0 {
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

func Merge(req *openrtb_ext.RequestWrapper, acctBidAdjs *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
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

	extPrebid.BidAdjustments.MediaType.Banner = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Banner, acctBidAdjs.MediaType.Banner)
	extPrebid.BidAdjustments.MediaType.Video = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Video, acctBidAdjs.MediaType.Video)
	extPrebid.BidAdjustments.MediaType.Native = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Native, acctBidAdjs.MediaType.Native)
	extPrebid.BidAdjustments.MediaType.Audio = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Audio, acctBidAdjs.MediaType.Audio)
	extPrebid.BidAdjustments.MediaType.WildCard = mergeForMediaType(extPrebid.BidAdjustments.MediaType.WildCard, acctBidAdjs.MediaType.WildCard)

	return extPrebid.BidAdjustments, nil
}

func mergeForMediaType(reqAdjMap map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID, accountAdjMap map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID) map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID {
	if reqAdjMap != nil && accountAdjMap == nil {
		return reqAdjMap
	}
	if reqAdjMap == nil && accountAdjMap != nil {
		return accountAdjMap
	}

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

func PopulateMap(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments, ruleToAdjustments map[string][]openrtb_ext.Adjustment) {
	if bidAdjustments == nil {
		return
	}
	populateMapForMediaType(bidAdjustments.MediaType.Banner, string(openrtb_ext.BidTypeBanner), ruleToAdjustments)
	populateMapForMediaType(bidAdjustments.MediaType.Video, string(openrtb_ext.BidTypeVideo), ruleToAdjustments)
	populateMapForMediaType(bidAdjustments.MediaType.Audio, string(openrtb_ext.BidTypeAudio), ruleToAdjustments)
	populateMapForMediaType(bidAdjustments.MediaType.Native, string(openrtb_ext.BidTypeNative), ruleToAdjustments)
	populateMapForMediaType(bidAdjustments.MediaType.WildCard, WildCard, ruleToAdjustments)
}

func populateMapForMediaType(bidAdj map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID, mediaType string, ruleToAdjustmentMap map[string][]openrtb_ext.Adjustment) {
	for bidderName := range bidAdj {
		for dealID, adjustments := range bidAdj[bidderName] {
			rule := mediaType + Delimiter + string(bidderName) + Delimiter + dealID
			ruleToAdjustmentMap[rule] = adjustments
		}
	}
}
