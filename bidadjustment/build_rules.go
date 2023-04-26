package bidadjustment

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

// BuildRules() will populate the rules map with a rule that's a combination of the mediaType, bidderName, and dealId for a particular adjustment
// The result will be a map that'll map a given rule with its adjustment
func BuildRules(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments, rules map[string][]openrtb_ext.Adjustment) {
	if bidAdjustments == nil {
		return
	}
	buildRulesForMediaType(string(openrtb_ext.BidTypeBanner), bidAdjustments.MediaType.Banner, rules)
	buildRulesForMediaType(string(openrtb_ext.BidTypeVideo), bidAdjustments.MediaType.Video, rules)
	buildRulesForMediaType(string(openrtb_ext.BidTypeAudio), bidAdjustments.MediaType.Audio, rules)
	buildRulesForMediaType(string(openrtb_ext.BidTypeNative), bidAdjustments.MediaType.Native, rules)
	buildRulesForMediaType(WildCard, bidAdjustments.MediaType.WildCard, rules)
}

func buildRulesForMediaType(mediaType string, rulesByBidder map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID, rules map[string][]openrtb_ext.Adjustment) {
	for bidderName := range rulesByBidder {
		for dealID, adjustments := range rulesByBidder[bidderName] {
			rule := mediaType + Delimiter + string(bidderName) + Delimiter + dealID
			rules[rule] = adjustments
		}
	}
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
