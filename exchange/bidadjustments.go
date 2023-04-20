package exchange

import (
	"math"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const roundTo float64 = 10000 // Rounds to 4 Decimal Places

func applyAdjustmentArray(adjArray []openrtb_ext.Adjustments, bidPrice float64, currency string, reqInfo *adapters.ExtraRequestInfo) (float64, string) {
	if adjArray == nil {
		return bidPrice, currency
	}
	originalBidPrice := bidPrice
	originalCurrency := currency

	for _, adjustment := range adjArray {
		if adjustment.AdjType == openrtb_ext.AdjTypeMultiplier {
			bidPrice = bidPrice * adjustment.Value
		} else if adjustment.AdjType == openrtb_ext.AdjTypeCpm {
			convertedVal, err := reqInfo.ConvertCurrency(adjustment.Value, adjustment.Currency, currency) // Convert Adjustment to Bid Currency
			if err != nil {
				return originalBidPrice, currency
			}
			bidPrice = bidPrice - convertedVal
		} else if adjustment.AdjType == openrtb_ext.AdjTypeStatic {
			bidPrice = adjustment.Value
			currency = adjustment.Currency
		}
	}
	roundedBidPrice := math.Round(bidPrice*roundTo) / roundTo // Returns Bid Price rounded to 4 decimal places

	if roundedBidPrice <= 0 {
		return originalBidPrice, originalCurrency
	}
	return roundedBidPrice, currency
}

func getAndApplyAdjustmentArray(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments, bidInfo *adapters.TypedBid, bidderName openrtb_ext.BidderName, currency string, reqInfo *adapters.ExtraRequestInfo) (float64, string) {
	adjArray := []openrtb_ext.Adjustments{}
	if bidAdjustments != nil {
		adjArray = bidAdjustments.GetAdjustmentArray(bidInfo.BidType, bidderName, bidInfo.Bid.DealID)
	} else {
		return bidInfo.Bid.Price, currency
	}
	return applyAdjustmentArray(adjArray, bidInfo.Bid.Price, currency, reqInfo)
}

func mergeBidAdjustments(req *openrtb_ext.RequestWrapper, acctBidAdjs *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
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
	return extPrebid.BidAdjustments, nil
}

func mergeAdjustmentsForMediaType(reqAdjMap map[string]map[string][]openrtb_ext.Adjustments, accountAdjMap map[string]map[string][]openrtb_ext.Adjustments) map[string]map[string][]openrtb_ext.Adjustments {
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

func processBidAdjustments(req *openrtb_ext.RequestWrapper, acctBidAdjs *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
	mergedBidAdj, err := mergeBidAdjustments(req, acctBidAdjs)
	if err != nil {
		return nil, err
	}
	if valid := mergedBidAdj.ValidateBidAdjustments(); !valid {
		mergedBidAdj = nil
	}
	return mergedBidAdj, err
}
