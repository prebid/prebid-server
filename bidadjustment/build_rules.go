package bidadjustment

import (
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const (
	VideoInstream  = "video-instream"
	VideoOutstream = "video-outstream"
)

// BuildRules() will populate the rules map with a rule that's a combination of the mediaType, bidderName, and dealId for a particular adjustment
// The result will be a map that'll map a given rule with its adjustment
func BuildRules(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments) map[string][]openrtb_ext.Adjustment {
	if bidAdjustments == nil {
		return nil
	}
	rules := make(map[string][]openrtb_ext.Adjustment)

	buildRulesForMediaType(string(openrtb_ext.BidTypeBanner), bidAdjustments.MediaType.Banner, rules)
	buildRulesForMediaType(string(openrtb_ext.BidTypeAudio), bidAdjustments.MediaType.Audio, rules)
	buildRulesForMediaType(string(openrtb_ext.BidTypeNative), bidAdjustments.MediaType.Native, rules)
	buildRulesForMediaType(VideoInstream, bidAdjustments.MediaType.VideoInstream, rules)
	buildRulesForMediaType(VideoOutstream, bidAdjustments.MediaType.VideoOutstream, rules)
	buildRulesForMediaType(WildCard, bidAdjustments.MediaType.WildCard, rules)

	return rules
}

func buildRulesForMediaType(mediaType string, rulesByBidder map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID, rules map[string][]openrtb_ext.Adjustment) {
	for bidderName := range rulesByBidder {
		for dealID, adjustments := range rulesByBidder[bidderName] {
			rule := mediaType + Delimiter + string(bidderName) + Delimiter + dealID
			rules[rule] = adjustments
		}
	}
}

// Merge takes bid adjustments defined on the request and on the account, and combines/validates them, with the adjustments on the request taking precedence.
func Merge(req *openrtb_ext.RequestWrapper, acctBidAdjs *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
	mergedBidAdj, err := merge(req, acctBidAdjs)
	if err != nil {
		return nil, err
	}
	if !Validate(mergedBidAdj) {
		mergedBidAdj = nil
		err = &errortypes.Warning{
			WarningCode: errortypes.BidAdjustmentWarningCode,
			Message:     "bid adjustment on account was invalid",
		}
	}
	return mergedBidAdj, err
}

func merge(req *openrtb_ext.RequestWrapper, acct *openrtb_ext.ExtRequestPrebidBidAdjustments) (*openrtb_ext.ExtRequestPrebidBidAdjustments, error) {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return nil, err
	}
	extPrebid := reqExt.GetPrebid()

	if extPrebid == nil || extPrebid.BidAdjustments == nil {
		return acct, nil
	}

	if acct == nil {
		return extPrebid.BidAdjustments, nil
	}

	extPrebid.BidAdjustments.MediaType.Banner = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Banner, acct.MediaType.Banner)
	extPrebid.BidAdjustments.MediaType.Native = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Native, acct.MediaType.Native)
	extPrebid.BidAdjustments.MediaType.Audio = mergeForMediaType(extPrebid.BidAdjustments.MediaType.Audio, acct.MediaType.Audio)
	extPrebid.BidAdjustments.MediaType.VideoInstream = mergeForMediaType(extPrebid.BidAdjustments.MediaType.VideoInstream, acct.MediaType.VideoInstream)
	extPrebid.BidAdjustments.MediaType.VideoOutstream = mergeForMediaType(extPrebid.BidAdjustments.MediaType.VideoOutstream, acct.MediaType.VideoOutstream)
	extPrebid.BidAdjustments.MediaType.WildCard = mergeForMediaType(extPrebid.BidAdjustments.MediaType.WildCard, acct.MediaType.WildCard)

	return extPrebid.BidAdjustments, nil
}

func mergeForMediaType(reqAdj, acctAdj map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID) map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID {
	if reqAdj != nil && acctAdj == nil {
		return reqAdj
	}
	if reqAdj == nil && acctAdj != nil {
		return acctAdj
	}

	for bidderName, dealIDToAdjustments := range acctAdj {
		if _, ok := reqAdj[bidderName]; ok {
			for dealID, acctAdjustments := range acctAdj[bidderName] {
				if _, ok := reqAdj[bidderName][dealID]; !ok {
					reqAdj[bidderName][dealID] = acctAdjustments
				}
			}
		} else {
			reqAdj[bidderName] = dealIDToAdjustments
		}
	}
	return reqAdj
}
