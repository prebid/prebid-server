package floors

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
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

type FloorConfig struct {
	FloorEnabled      bool
	EnforceRate       int
	EnforceDealFloors bool
}

func NewFloorConfig(priceFloor config.PriceFloors) *FloorConfig {

	floorConfig := FloorConfig{
		FloorEnabled:      priceFloor.Enabled,
		EnforceRate:       priceFloor.EnforceFloorsRate,
		EnforceDealFloors: priceFloor.EnforceDealFloors,
	}

	return &floorConfig
}

func (fc *FloorConfig) Enabled() bool {
	return fc.FloorEnabled
}

func (fc *FloorConfig) GetEnforceRate() int {
	return fc.EnforceRate
}

func (fc *FloorConfig) EnforceDealFloor() bool {
	return fc.EnforceDealFloors
}

type Floor interface {
	Enabled() bool
	GetEnforceRate() int
	EnforceDealFloor() bool
}

// IsRequestEnabledWithFloor will check if floors is enabled in request
func IsRequestEnabledWithFloor(Floors *openrtb_ext.PriceFloorRules) bool {
	if Floors == nil {
		return false
	}

	if Floors.Enabled != nil && !*Floors.Enabled {
		return *Floors.Enabled
	}

	return true
}

// UpdateImpsWithFloors will validate floor rules, based on request and rules prepares various combinations
// to match with floor rules and selects appripariate floor rule and update imp.bidfloor and imp.bidfloorcur
func UpdateImpsWithFloors(floorExt *openrtb_ext.PriceFloorRules, request *openrtb2.BidRequest, conversions currency.Conversions) []error {
	var (
		floorErrList      []error
		floorModelErrList []error
		floorVal          float64
	)
	floorData := floorExt.Data

	floorData.ModelGroups, floorModelErrList = validateFloorModelGroups(floorData.ModelGroups)
	if len(floorData.ModelGroups) == 0 {
		return floorModelErrList
	} else if len(floorData.ModelGroups) > 1 {
		floorData.ModelGroups = selectFloorModelGroup(floorData.ModelGroups, rand.Intn)
	}

	if floorData.ModelGroups[0].Schema.Delimiter == "" {
		floorData.ModelGroups[0].Schema.Delimiter = DEFAULT_DELIMITER
	}

	floorExt.Skipped = new(bool)
	if shouldSkipFloors(floorExt.Data.ModelGroups[0].SkipRate, floorExt.Data.SkipRate, floorExt.SkipRate, rand.Intn) {
		*floorExt.Skipped = true
		floorData.ModelGroups = nil
		return floorModelErrList
	}

	floorErrList = validateFloorRules(floorData.ModelGroups[0].Schema, floorData.ModelGroups[0].Schema.Delimiter, floorData.ModelGroups[0].Values)
	if len(floorData.ModelGroups[0].Values) > 0 {
		for i := 0; i < len(request.Imp); i++ {
			desiredRuleKey := createRuleKey(floorData.ModelGroups[0].Schema, request, request.Imp[i])
			matchedRule := findRule(floorData.ModelGroups[0].Values, floorData.ModelGroups[0].Schema.Delimiter, desiredRuleKey, len(floorData.ModelGroups[0].Schema.Fields))

			floorVal = floorData.ModelGroups[0].Default
			if matchedRule != "" {
				floorVal = floorData.ModelGroups[0].Values[matchedRule]
			}

			if floorVal > 0.0 {
				request.Imp[i].BidFloor = math.Round(floorVal*10000) / 10000
				floorMinVal, floorCur, err := getMinFloorValue(floorExt, conversions)
				if err == nil {
					if floorMinVal > 0.0 && floorVal < floorMinVal {
						request.Imp[i].BidFloor = math.Round(floorMinVal*10000) / 10000
					}
					request.Imp[i].BidFloorCur = floorCur
					updateImpExtWithFloorDetails(matchedRule, &request.Imp[i], floorVal)
				} else {
					floorModelErrList = append(floorModelErrList, fmt.Errorf("error in Currency Conversion  = '%v'", err.Error()))
				}
			}
		}
	}
	floorModelErrList = append(floorModelErrList, floorErrList...)

	return floorModelErrList
}
