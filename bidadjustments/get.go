package bidadjustments

import "github.com/prebid/prebid-server/openrtb_ext"

func GetAdjustmentArray(bidAdj *openrtb_ext.ExtRequestPrebidBidAdjustments, bidType openrtb_ext.BidType, bidderName openrtb_ext.BidderName, dealID string) []openrtb_ext.Adjustments {
	if bidAdj.MediaType.Banner != nil && bidType == openrtb_ext.BidTypeBanner {
		if adjArray := getAdjustmentArrayForMediaType(bidAdj.MediaType.Banner, bidderName.String(), dealID); adjArray != nil {
			return adjArray
		}
	}
	if bidAdj.MediaType.Video != nil && bidType == openrtb_ext.BidTypeVideo {
		if adjArray := getAdjustmentArrayForMediaType(bidAdj.MediaType.Video, bidderName.String(), dealID); adjArray != nil {
			return adjArray
		}
	}
	if bidAdj.MediaType.Audio != nil && bidType == openrtb_ext.BidTypeAudio {
		if adjArray := getAdjustmentArrayForMediaType(bidAdj.MediaType.Audio, bidderName.String(), dealID); adjArray != nil {
			return adjArray
		}

	}
	if bidAdj.MediaType.Native != nil && bidType == openrtb_ext.BidTypeNative {
		if adjArray := getAdjustmentArrayForMediaType(bidAdj.MediaType.Native, bidderName.String(), dealID); adjArray != nil {
			return adjArray
		}
	}
	if bidAdj.MediaType.WildCard != nil {
		if adjArray := getAdjustmentArrayForMediaType(bidAdj.MediaType.WildCard, bidderName.String(), dealID); adjArray != nil {
			return adjArray
		}
	}
	return nil
}

// Priority For Returning Adjustment Array Based on Passed BidderName and DealID
// #1: Are able to match bidderName and dealID
// #2: Are able to match bidderName and dealID field is WildCard
// #3: Bidder field is WildCard and are able to match DealID
// #4: Wildcard bidder and wildcard dealID
func getAdjustmentArrayForMediaType(bidAdjMap map[string]map[string][]openrtb_ext.Adjustments, bidderName string, dealID string) []openrtb_ext.Adjustments {
	if _, ok := bidAdjMap[bidderName]; ok {
		if _, ok := bidAdjMap[bidderName][dealID]; ok {
			return bidAdjMap[bidderName][dealID]
		} else if _, ok := bidAdjMap[bidderName][AdjWildCard]; ok {
			return bidAdjMap[bidderName][AdjWildCard]
		}
	} else if _, ok := bidAdjMap[AdjWildCard]; ok {
		if _, ok := bidAdjMap[AdjWildCard][dealID]; ok {
			return bidAdjMap[AdjWildCard][dealID]
		} else if _, ok := bidAdjMap[AdjWildCard][AdjWildCard]; ok {
			return bidAdjMap[AdjWildCard][AdjWildCard]
		}
	}
	return nil
}
