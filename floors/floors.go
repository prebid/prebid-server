package floors

import (
	"errors"
	"math"
	"math/rand"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Price struct {
	FloorMin    float64
	FloorMinCur string
}

const (
	defaultCurrency  string  = "USD"
	defaultDelimiter string  = "|"
	catchAll         string  = "*"
	skipRateMin      int     = 0
	skipRateMax      int     = 100
	modelWeightMax   int     = 100
	modelWeightMin   int     = 1
	enforceRateMin   int     = 0
	enforceRateMax   int     = 100
	floorPrecision   float64 = 0.01
)

// EnrichWithPriceFloors checks for floors enabled in account and request and selects floors data from dynamic fetched if present
// else selects floors data from req.ext.prebid.floors and update request with selected floors details
func EnrichWithPriceFloors(bidRequestWrapper *openrtb_ext.RequestWrapper, account config.Account, conversions currency.Conversions) []error {

	if bidRequestWrapper == nil || bidRequestWrapper.BidRequest == nil {
		return []error{errors.New("Empty bidrequest")}
	}

	if !isPriceFloorsEnabled(account, bidRequestWrapper) {
		return []error{errors.New("Floors feature is disabled at account or in the request")}
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
		for _, imp := range request.GetImp() {
			desiredRuleKey := createRuleKey(modelGroup.Schema, request, imp)
			matchedRule, isRuleMatched := findRule(modelGroup.Values, modelGroup.Schema.Delimiter, desiredRuleKey)
			floorVal = modelGroup.Default
			if isRuleMatched {
				floorVal = modelGroup.Values[matchedRule]
			}

			// No rule is matched or no default value provided or non-zero bidfloor not provided
			if floorVal == 0.0 {
				continue
			}

			floorMinVal, floorCur, err := getMinFloorValue(extFloorRules, imp, conversions)
			if err == nil {
				floorVal = roundToFourDecimals(floorVal)
				bidFloor := floorVal
				if floorMinVal > 0.0 && floorVal < floorMinVal {
					bidFloor = floorMinVal
				}

				imp.BidFloor = bidFloor
				imp.BidFloorCur = floorCur

				if isRuleMatched {
					err = updateImpExtWithFloorDetails(imp, matchedRule, floorVal, imp.BidFloor)
					if err != nil {
						floorErrList = append(floorErrList, err)
					}
				}
			} else {
				floorErrList = append(floorErrList, err)
			}
		}
	}
	return floorErrList
}

// roundToFourDecimals retuns given value to 4 decimal points
func roundToFourDecimals(in float64) float64 {
	return math.Round(in*10000) / 10000
}

// isPriceFloorsEnabled check for floors are enabled at account and request level
func isPriceFloorsEnabled(account config.Account, bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	return isPriceFloorsEnabledForAccount(account) && isPriceFloorsEnabledForRequest(bidRequestWrapper)
}

// isPriceFloorsEnabledForAccount check for floors enabled flag in account config
func isPriceFloorsEnabledForAccount(account config.Account) bool {
	return account.PriceFloors.Enabled
}

// isPriceFloorsEnabledForRequest check for floors are enabled flag in request
func isPriceFloorsEnabledForRequest(bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil {
			return prebidExt.Floors.GetEnabled()
		}
	}
	return true
}

// resolveFloors does selection of floors fields from request data and dynamic fetched data if dynamic fetch is enabled
func resolveFloors(account config.Account, bidRequestWrapper *openrtb_ext.RequestWrapper, conversions currency.Conversions) (*openrtb_ext.PriceFloorRules, []error) {
	var errList []error
	var floorRules *openrtb_ext.PriceFloorRules

	reqFloor := extractFloorsFromRequest(bidRequestWrapper)
	if reqFloor != nil {
		floorRules, errList = createFloorsFrom(reqFloor, account, openrtb_ext.FetchNone, openrtb_ext.RequestLocation)
	} else {
		floorRules, errList = createFloorsFrom(nil, account, openrtb_ext.FetchNone, openrtb_ext.NoDataLocation)
	}
	return floorRules, errList
}

// createFloorsFrom does preparation of floors data which shall be used for further processing
func createFloorsFrom(floors *openrtb_ext.PriceFloorRules, account config.Account, fetchStatus, floorLocation string) (*openrtb_ext.PriceFloorRules, []error) {
	var floorModelErrList []error
	finalFloors := &openrtb_ext.PriceFloorRules{
		FetchStatus:        fetchStatus,
		PriceFloorLocation: floorLocation,
	}

	if floors != nil {
		floorValidationErr := validateFloorParams(floors)
		if floorValidationErr != nil {
			return finalFloors, append(floorModelErrList, floorValidationErr)
		}

		finalFloors.Enforcement = floors.Enforcement
		if floors.Data != nil {
			validModelGroups, floorModelErrList := selectValidFloorModelGroups(floors.Data.ModelGroups, account)
			if len(validModelGroups) == 0 {
				return finalFloors, floorModelErrList
			} else {
				*finalFloors = *floors
				finalFloors.Data = new(openrtb_ext.PriceFloorData)
				*finalFloors.Data = *floors.Data
				finalFloors.PriceFloorLocation = floorLocation
				finalFloors.FetchStatus = fetchStatus
				if len(validModelGroups) > 1 {
					validModelGroups = selectFloorModelGroup(validModelGroups, rand.Intn)
				}
				finalFloors.Data.ModelGroups = []openrtb_ext.PriceFloorModelGroup{validModelGroups[0].Copy()}
			}
		}
	}

	return finalFloors, floorModelErrList
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

// updateFloorsInRequest updates req.ext.prebid.floors with floors data
func updateFloorsInRequest(bidRequestWrapper *openrtb_ext.RequestWrapper, priceFloors *openrtb_ext.PriceFloorRules) {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		prebidExt := requestExt.GetPrebid()
		if prebidExt == nil {
			prebidExt = &openrtb_ext.ExtRequestPrebid{}
		}
		prebidExt.Floors = priceFloors
		requestExt.SetPrebid(prebidExt)
		bidRequestWrapper.RebuildRequest()
	}
}
