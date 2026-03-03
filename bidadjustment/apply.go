package bidadjustment

import (
	"math"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const (
	AdjustmentTypeCPM        = "cpm"
	AdjustmentTypeMultiplier = "multiplier"
	AdjustmentTypeStatic     = "static"
	WildCard                 = "*"
	Delimiter                = "|"
)

const maxNumOfCombos = 8
const pricePrecision float64 = 10000 // Rounds to 4 Decimal Places
const minBid = 0.1

// Apply gets the highest priority adjustment slice given a map of rules, and applies those adjustments to a bid's price
func Apply(rules map[string][]openrtb_ext.Adjustment, bidInfo *adapters.TypedBid, bidderName openrtb_ext.BidderName, currency string, reqInfo *adapters.ExtraRequestInfo, bidType string) (float64, string) {
	var adjustments []openrtb_ext.Adjustment
	if len(rules) > 0 {
		adjustments = get(rules, bidType, string(bidderName), bidInfo.Bid.DealID)
	} else {
		return bidInfo.Bid.Price, currency
	}
	adjustedPrice, adjustedCurrency := apply(adjustments, bidInfo.Bid.Price, currency, reqInfo)

	if bidInfo.Bid.DealID != "" && adjustedPrice < 0 {
		return 0, currency
	}
	if bidInfo.Bid.DealID == "" && adjustedPrice <= 0 {
		return minBid, currency
	}
	return adjustedPrice, adjustedCurrency
}

func apply(adjustments []openrtb_ext.Adjustment, bidPrice float64, currency string, reqInfo *adapters.ExtraRequestInfo) (float64, string) {
	if len(adjustments) == 0 {
		return bidPrice, currency
	}
	originalBidPrice := bidPrice

	for _, adjustment := range adjustments {
		switch adjustment.Type {
		case AdjustmentTypeMultiplier:
			bidPrice = bidPrice * adjustment.Value
		case AdjustmentTypeCPM:
			convertedVal, err := reqInfo.ConvertCurrency(adjustment.Value, adjustment.Currency, currency)
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

	return roundedBidPrice, currency
}

// get() should return the highest priority slice of adjustments from the map that we can match with the given bid info
// given the bid info, we create the same format of combinations that's present in the key of the ruleToAdjustments map
// the slice is ordered by priority from highest to lowest, as soon as we find a match, we return that slice
func get(rules map[string][]openrtb_ext.Adjustment, bidType, bidderName, dealID string) []openrtb_ext.Adjustment {
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
		if _, ok := rules[rule]; ok {
			return rules[rule]
		}
	}
	return nil
}
