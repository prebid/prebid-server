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

const maxNumOfCombos = 8
const pricePrecision float64 = 10000 // Rounds to 4 Decimal Places

func Apply(ruleToAdjustments map[string][]openrtb_ext.Adjustment, bidInfo *adapters.TypedBid, bidderName openrtb_ext.BidderName, currency string, reqInfo *adapters.ExtraRequestInfo) (float64, string) {
	adjustments := []openrtb_ext.Adjustment{}
	if ruleToAdjustments != nil {
		adjustments = get(ruleToAdjustments, string(bidInfo.BidType), string(bidderName), bidInfo.Bid.DealID)
	} else {
		return bidInfo.Bid.Price, currency
	}
	return apply(adjustments, bidInfo.Bid.Price, currency, reqInfo)
}

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

// get() should return the highest priority slice of adjustments from the map that we can match with the given bid info
// given the bid info, we create the same format of combinations that's present in the key of the ruleToAdjustments map
// the slice is ordered by priority from highest to lowest, as soon as we find a match, we return that slice
func get(ruleToAdjustments map[string][]openrtb_ext.Adjustment, bidType, bidderName, dealID string) []openrtb_ext.Adjustment {
	priorityRules := [maxNumOfCombos]string{}
	if dealID != "" {
		priorityRules[0] = bidType + Delimiter + bidderName + Delimiter + dealID
		priorityRules[1] = bidType + Delimiter + bidderName + Delimiter + WildCard
		priorityRules[2] = bidType + Delimiter + WildCard + Delimiter + dealID
		priorityRules[3] = WildCard + Delimiter + bidderName + Delimiter + dealID
		priorityRules[4] = bidType + Delimiter + WildCard + Delimiter + WildCard
		priorityRules[5] = WildCard + Delimiter + bidderName + Delimiter + WildCard
		priorityRules[6] = WildCard + Delimiter + WildCard + Delimiter + dealID
		priorityRules[7] = WildCard + Delimiter + WildCard + Delimiter + WildCard
	} else {
		priorityRules[0] = bidType + Delimiter + bidderName + Delimiter + WildCard
		priorityRules[1] = bidType + Delimiter + WildCard + Delimiter + WildCard
		priorityRules[2] = WildCard + Delimiter + bidderName + Delimiter + WildCard
		priorityRules[3] = WildCard + Delimiter + WildCard + Delimiter + WildCard
	}

	for _, rule := range priorityRules {
		if _, ok := ruleToAdjustments[rule]; ok {
			return ruleToAdjustments[rule]
		}
	}
	return nil
}
