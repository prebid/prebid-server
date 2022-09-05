package floors

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	DEFAULT_DELIMITER      string = "|"
	CATCH_ALL              string = "*"
	SKIP_RATE_MIN          int    = 0
	SKIP_RATE_MAX          int    = 100
	MODEL_WEIGHT_MAX_VALUE int    = 100
	MODEL_WEIGHT_MIN_VALUE int    = 0
	ENFORCE_RATE_MIN       int    = 0
	ENFORCE_RATE_MAX       int    = 100
)

// ModifyImpsWithFloors will validate floor rules, based on request and rules prepares various combinations
// to match with floor rules and selects appripariate floor rule and update imp.bidfloor and imp.bidfloorcur
func ModifyImpsWithFloors(floorExt *openrtb_ext.PriceFloorRules, request *openrtb2.BidRequest, conversions currency.Conversions) []error {
	var (
		floorErrList      []error
		floorModelErrList []error
		floorVal          float64
	)
	floorData := floorExt.Data
	if floorData == nil {
		return nil
	}

	floorModelErrList = validateFloorSkipRates(floorExt)
	if len(floorModelErrList) > 0 {
		return floorModelErrList
	}

	floorData.ModelGroups, floorModelErrList = selectValidFloorModelGroups(floorData.ModelGroups)
	if len(floorData.ModelGroups) == 0 {
		return floorModelErrList
	} else if len(floorData.ModelGroups) > 1 {
		floorData.ModelGroups = selectFloorModelGroup(floorData.ModelGroups, rand.Intn)
	}

	modelGroup := floorData.ModelGroups[0]
	if modelGroup.Schema.Delimiter == "" {
		modelGroup.Schema.Delimiter = DEFAULT_DELIMITER
	}

	floorExt.Skipped = new(bool)
	if shouldSkipFloors(floorExt.Data.ModelGroups[0].SkipRate, floorExt.Data.SkipRate, floorExt.SkipRate, rand.Intn) {
		*floorExt.Skipped = true
		floorData.ModelGroups = nil
		return floorModelErrList
	}

	floorErrList = validateFloorRules(modelGroup.Schema, modelGroup.Schema.Delimiter, modelGroup.Values)
	if len(modelGroup.Values) > 0 {
		for i := 0; i < len(request.Imp); i++ {
			desiredRuleKey := createRuleKey(modelGroup.Schema, request, request.Imp[i])
			matchedRule, isRuleMatched := findRule(modelGroup.Values, modelGroup.Schema.Delimiter, desiredRuleKey, len(modelGroup.Schema.Fields))

			floorVal = modelGroup.Default
			if isRuleMatched {
				floorVal = modelGroup.Values[matchedRule]
			}

			floorMinVal, floorCur, err := getMinFloorValue(floorExt, conversions)
			if err == nil {
				if floorMinVal > 0.0 && floorVal < floorMinVal {
					request.Imp[i].BidFloor = math.Round(floorMinVal*10000) / 10000
				} else {
					request.Imp[i].BidFloor = math.Round(floorVal*10000) / 10000
				}
				request.Imp[i].BidFloorCur = floorCur
				updateImpExtWithFloorDetails(matchedRule, &request.Imp[i], floorVal)
			} else {
				floorModelErrList = append(floorModelErrList, fmt.Errorf("Error in getting FloorMin value : '%v'", err.Error()))
			}

		}
	}
	floorModelErrList = append(floorModelErrList, floorErrList...)
	return floorModelErrList
}
