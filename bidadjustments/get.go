package bidadjustments

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func GetAndApplyAdjustmentArray(ruleToAdjustments map[string][]openrtb_ext.Adjustments, bidInfo *adapters.TypedBid, bidderName openrtb_ext.BidderName, currency string, reqInfo *adapters.ExtraRequestInfo) (float64, string) {
	adjArray := []openrtb_ext.Adjustments{}
	if ruleToAdjustments != nil {
		adjArray = getAdjustmentArray(ruleToAdjustments, string(bidInfo.BidType), string(bidderName), bidInfo.Bid.DealID)
	} else {
		return bidInfo.Bid.Price, currency
	}
	return applyAdjustmentArray(adjArray, bidInfo.Bid.Price, currency, reqInfo)
}

// Given the bid response information of bidType, bidderName, and dealID, we create the same format of combinations that's present in the key of the ruleToAdjustments map
// There's a max of 8 combinations that can be made with those pieces of bid information, so after discussion with team member, we decided to generate each combo, and check if it's present in the map
// The order of the array is ordered by priority from highest to lowest, so as soon as we find a match, that's the adjustment array we want to return
func getAdjustmentArray(ruleToAdjustments map[string][]openrtb_ext.Adjustments, bidType string, bidderName string, dealID string) []openrtb_ext.Adjustments {
	var priorityRules []string
	if dealID != "" {
		priorityRules = append(priorityRules, bidType+PipeDelimiter+bidderName+PipeDelimiter+dealID)
		priorityRules = append(priorityRules, bidType+PipeDelimiter+bidderName+PipeDelimiter+WildCard)
		priorityRules = append(priorityRules, bidType+PipeDelimiter+WildCard+PipeDelimiter+dealID)
		priorityRules = append(priorityRules, WildCard+PipeDelimiter+bidderName+PipeDelimiter+dealID)
		priorityRules = append(priorityRules, bidType+PipeDelimiter+WildCard+PipeDelimiter+WildCard)
		priorityRules = append(priorityRules, WildCard+PipeDelimiter+bidderName+PipeDelimiter+WildCard)
		priorityRules = append(priorityRules, WildCard+PipeDelimiter+WildCard+PipeDelimiter+dealID)
		priorityRules = append(priorityRules, WildCard+PipeDelimiter+WildCard+PipeDelimiter+WildCard)
	} else {
		priorityRules = append(priorityRules, bidType+PipeDelimiter+bidderName+PipeDelimiter+WildCard)
		priorityRules = append(priorityRules, bidType+PipeDelimiter+WildCard+PipeDelimiter+WildCard)
		priorityRules = append(priorityRules, WildCard+PipeDelimiter+bidderName+PipeDelimiter+WildCard)
		priorityRules = append(priorityRules, WildCard+PipeDelimiter+WildCard+PipeDelimiter+WildCard)
	}

	for _, rule := range priorityRules {
		if _, ok := ruleToAdjustments[rule]; ok {
			return ruleToAdjustments[rule]
		}
	}
	return nil
}
