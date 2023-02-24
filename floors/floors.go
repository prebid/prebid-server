package floors

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Price struct {
	FloorMin    float64
	FloorMinCur string
}

const (
	defaultDelimiter string = "|"
	catchAll         string = "*"
	skipRateMin      int    = 0
	skipRateMax      int    = 100
	modelWeightMax   int    = 100
	modelWeightMin   int    = 1
	enforceRateMin   int    = 0
	enforceRateMax   int    = 100
)

// EnrichWithPriceFloors checks for floors enabled in account and request and selects floors data from dynamic fetched floors JSON if present
// else selects floors JSON from req.ext.prebid.floors and update request with selected floors details
func EnrichWithPriceFloors(bidRequestWrapper *openrtb_ext.RequestWrapper, account config.Account, conversions currency.Conversions) []error {

	if bidRequestWrapper == nil || bidRequestWrapper.BidRequest == nil {
		return []error{errors.New("Empty bidrequest")}
	}

	if isPriceFloorsDisabled(account, bidRequestWrapper) {
		return []error{errors.New("Floors feature is disabled at account level or request")}
	}

	floors, err := resolveFloors(account, bidRequestWrapper, conversions)

	updateReqErrs := updateBidRequestWithFloors(floors, bidRequestWrapper, conversions)
	updateFloorsInRequest(bidRequestWrapper, floors)
	return append(err, updateReqErrs...)
}

// updateBidRequestWithFloors will update imp.bidfloor and imp.bidfloorcur based on rules matching
func updateBidRequestWithFloors(extFloorRules *openrtb_ext.PriceFloorRules, request *openrtb_ext.RequestWrapper, conversions currency.Conversions) []error {
	var (
		floorErrList []error
		floorVal     float64
	)

	if extFloorRules == nil || extFloorRules.Data == nil || len(extFloorRules.Data.ModelGroups) == 0 {
		return []error{}
	}

	modelGroup := extFloorRules.Data.ModelGroups[0]
	if modelGroup.Schema.Delimiter == "" {
		modelGroup.Schema.Delimiter = defaultDelimiter
	}

	extFloorRules.Skipped = new(bool)
	if shouldSkipFloors(modelGroup.SkipRate, extFloorRules.Data.SkipRate, extFloorRules.SkipRate, rand.Intn) {
		*extFloorRules.Skipped = true
		return []error{}
	}

	floorErrList = validateFloorRulesAndLowerValidRuleKey(modelGroup.Schema, modelGroup.Schema.Delimiter, modelGroup.Values)
	if len(modelGroup.Values) > 0 {
		for i, imp := range request.GetImp() {
			desiredRuleKey := createRuleKey(modelGroup.Schema, request.BidRequest, request.Imp[i])
			matchedRule, isRuleMatched := findRule(modelGroup.Values, modelGroup.Schema.Delimiter, desiredRuleKey, len(modelGroup.Schema.Fields))

			floorVal = modelGroup.Default
			if isRuleMatched {
				floorVal = modelGroup.Values[matchedRule]
			}

			floorMinVal, floorCur, err := getMinFloorValue(extFloorRules, request.Imp[i], conversions)
			if err == nil {
				floorVal = math.Round(floorVal*10000) / 10000
				bidFloor := floorVal
				if floorMinVal > 0.0 && floorVal < floorMinVal {
					bidFloor = floorMinVal
				}

				if bidFloor > 0.0 {
					imp.BidFloor = math.Round(bidFloor*10000) / 10000
					imp.BidFloorCur = floorCur
				}
				if isRuleMatched {
					updateImpExtWithFloorDetails(imp, matchedRule, floorVal, imp.BidFloor)
				}
			} else {
				floorErrList = append(floorErrList, fmt.Errorf("Error in getting FloorMin value : '%v'", err.Error()))
			}
		}
		err := request.RebuildImp()
		if err != nil {
			return append(floorErrList, err)
		}
	}
	return floorErrList
}

// isPriceFloorsDisabled check for floors are disabled at account or request level
func isPriceFloorsDisabled(account config.Account, bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	return isPriceFloorsDisabledForAccount(account) || isPriceFloorsDisabledForRequest(bidRequestWrapper)
}

// isPriceFloorsDisabledForAccount check for floors are disabled at account
func isPriceFloorsDisabledForAccount(account config.Account) bool {
	return !account.PriceFloors.Enabled
}

// isPriceFloorsDisabledForRequest check for floors are disabled at request
func isPriceFloorsDisabledForRequest(bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil && !prebidExt.Floors.GetEnabled() {
			return true
		}
	}
	return false
}

// resolveFloors does selection of floors fields from requet JSON and dynamic fetched floors JSON if dynamic fetch is enabled
func resolveFloors(account config.Account, bidRequestWrapper *openrtb_ext.RequestWrapper, conversions currency.Conversions) (*openrtb_ext.PriceFloorRules, []error) {
	var errList []error
	var floorsJson *openrtb_ext.PriceFloorRules

	reqFloor := extractFloorsFromRequest(bidRequestWrapper)
	if reqFloor != nil {
		floorsJson, errList = createFloorsFrom(reqFloor, openrtb_ext.FetchNone, openrtb_ext.RequestLocation)
	} else {
		floorsJson, errList = createFloorsFrom(nil, openrtb_ext.FetchNone, openrtb_ext.NoDataLocation)
	}
	return floorsJson, errList
}

// createFloorsFrom does preparation of floors data which shall be used for further processing
func createFloorsFrom(floors *openrtb_ext.PriceFloorRules, fetchStatus, floorLocation string) (*openrtb_ext.PriceFloorRules, []error) {
	var floorModelErrList []error
	finalFloors := new(openrtb_ext.PriceFloorRules)

	if floors != nil {
		floorValidationErr := validateFloorParams(floors)
		if floorValidationErr != nil {
			finalFloors.FetchStatus = fetchStatus
			finalFloors.PriceFloorLocation = floorLocation
			return finalFloors, append(floorModelErrList, floorValidationErr)
		}

		finalFloors.Enforcement = floors.Enforcement
		if floors.Data != nil {
			validModelGroups, floorModelErrList := selectValidFloorModelGroups(floors.Data.ModelGroups)
			if len(validModelGroups) == 0 {
				finalFloors.FetchStatus = fetchStatus
				finalFloors.PriceFloorLocation = floorLocation
				return finalFloors, floorModelErrList
			} else {
				*finalFloors = *floors
				finalFloors.Data = new(openrtb_ext.PriceFloorData)
				*finalFloors.Data = *floors.Data
				if len(validModelGroups) > 1 {
					validModelGroups = selectFloorModelGroup(validModelGroups, rand.Intn)
				}
				finalFloors.Data.ModelGroups = []openrtb_ext.PriceFloorModelGroup{validModelGroups[0].Copy()}
			}
		}
	}
	finalFloors.FetchStatus = fetchStatus
	finalFloors.PriceFloorLocation = floorLocation
	return finalFloors, floorModelErrList
}

// resolveFloorMin gets floorMin valud from request and dynamic fetched data
func resolveFloorMin(reqFloors *openrtb_ext.PriceFloorRules, fetchFloors openrtb_ext.PriceFloorRules, conversions currency.Conversions) Price {
	var floorCur, reqFloorMinCur string
	var reqFloorMin float64
	if reqFloors != nil {
		floorCur = getFloorCurrency(reqFloors)
		reqFloorMin = reqFloors.FloorMin
		reqFloorMinCur = reqFloors.FloorMinCur
	}

	if len(reqFloorMinCur) == 0 && fetchFloors.Data == nil {
		reqFloorMinCur = floorCur
	}

	provFloorMinCur := fetchFloors.FloorMinCur
	provFloorMin := fetchFloors.FloorMin

	if len(reqFloorMinCur) > 0 {
		if reqFloorMin > 0.0 {
			return Price{FloorMin: reqFloorMin, FloorMinCur: reqFloorMinCur}
		} else if provFloorMin > 0.0 {
			if len(provFloorMinCur) == 0 || strings.Compare(reqFloorMinCur, provFloorMinCur) == 0 {
				return Price{FloorMin: provFloorMin, FloorMinCur: reqFloorMinCur}
			}
			rate, err := conversions.GetRate(provFloorMinCur, reqFloorMinCur)
			if err == nil {
				return Price{FloorMinCur: reqFloorMinCur,
					FloorMin: math.Round(rate*provFloorMin*10000) / 10000}
			}
		}
	}
	if len(provFloorMinCur) == 0 {
		provFloorMinCur = getFloorCurrency(&fetchFloors)
	}
	if len(provFloorMinCur) > 0 {
		if provFloorMin > 0.0 {
			return Price{FloorMin: provFloorMin, FloorMinCur: provFloorMinCur}
		} else if reqFloorMin > 0.0 {
			return Price{FloorMin: reqFloorMin, FloorMinCur: provFloorMinCur}
		}
	}
	return Price{FloorMin: 0.0, FloorMinCur: floorCur}
}

// extractFloorsFromRequest gets floors data from req.ext.prebid.floors
func extractFloorsFromRequest(bidRequestWrapper *openrtb_ext.RequestWrapper) *openrtb_ext.PriceFloorRules {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		prebidExt := requestExt.GetPrebid()
		if prebidExt != nil && prebidExt.Floors != nil {
			return prebidExt.Floors
		}
	}
	return nil
}

// updateFloorsInRequest updates floors data into req.ext.prebid.floors
func updateFloorsInRequest(bidRequestWrapper *openrtb_ext.RequestWrapper, priceFloors *openrtb_ext.PriceFloorRules) {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		prebidExt := requestExt.GetPrebid()
		if prebidExt != nil {
			prebidExt.Floors = priceFloors
			requestExt.SetPrebid(prebidExt)
			bidRequestWrapper.RebuildRequestExt()
		}
	}
}
