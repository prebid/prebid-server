package floors

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Enforce does floors enforcement for bids from all bidders based on floors provided in request, account level floors config
func Enforce(bidRequestWrapper *openrtb_ext.RequestWrapper, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, account config.Account, conversions currency.Conversions) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, []error, []*entities.PbsOrtbSeatBid) {
	rejectionErrs := []error{}

	rejectedBids := []*entities.PbsOrtbSeatBid{}

	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err != nil {
		return seatBids, []error{errors.New("Error in getting request extension")}, rejectedBids
	}

	if !isPriceFloorsEnabled(account, bidRequestWrapper) {
		return seatBids, nil, rejectedBids
	}

	if isSignalingSkipped(requestExt) || !isValidImpBidFloorPresent(bidRequestWrapper.BidRequest.Imp) {
		return seatBids, nil, rejectedBids
	}

	enforceFloors := isSatisfiedByEnforceRate(requestExt, account.PriceFloors.EnforceFloorsRate, rand.Intn)
	if updateEnforcePBS(enforceFloors, requestExt) {
		err := bidRequestWrapper.RebuildRequest()
		if err != nil {
			return seatBids, []error{err}, rejectedBids
		}
	}
	updateBidExt(bidRequestWrapper, seatBids)
	if enforceFloors {
		enforceDealFloors := account.PriceFloors.EnforceDealFloors && getEnforceDealsFlag(requestExt)
		seatBids, rejectionErrs, rejectedBids = enforceFloorToBids(bidRequestWrapper, seatBids, conversions, enforceDealFloors)
	}
	return seatBids, rejectionErrs, rejectedBids
}

// updateEnforcePBS updates prebid extension in request if enforcePBS needs to be updated
func updateEnforcePBS(enforceFloors bool, requestExt *openrtb_ext.RequestExt) bool {
	updateReqExt := false

	prebidExt := requestExt.GetPrebid()
	if prebidExt == nil {
		prebidExt = new(openrtb_ext.ExtRequestPrebid)
	}

	if prebidExt.Floors == nil {
		prebidExt.Floors = new(openrtb_ext.PriceFloorRules)
	}
	floorExt := prebidExt.Floors

	if floorExt.Enforcement == nil {
		floorExt.Enforcement = new(openrtb_ext.PriceFloorEnforcement)
	}

	if floorExt.Enforcement.EnforcePBS == nil {
		updateReqExt = true
		floorExt.Enforcement.EnforcePBS = new(bool)
		*floorExt.Enforcement.EnforcePBS = enforceFloors
	} else {
		oldEnforcePBS := *floorExt.Enforcement.EnforcePBS
		*floorExt.Enforcement.EnforcePBS = enforceFloors && *floorExt.Enforcement.EnforcePBS
		updateReqExt = oldEnforcePBS != *floorExt.Enforcement.EnforcePBS
	}

	if updateReqExt {
		requestExt.SetPrebid(prebidExt)
	}

	return updateReqExt
}

// updateBidExt updates bid extension for floors related details
func updateBidExt(bidRequestWrapper *openrtb_ext.RequestWrapper, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) {
	impMap := make(map[string]*openrtb_ext.ImpWrapper, bidRequestWrapper.LenImp())
	for _, imp := range bidRequestWrapper.GetImp() {
		impMap[imp.ID] = imp
	}

	for _, seatBid := range seatBids {
		for _, bid := range seatBid.Bids {
			reqImp, ok := impMap[bid.Bid.ImpID]
			if ok {
				updateBidExtWithFloors(reqImp, bid, reqImp.BidFloorCur)
			}
		}
	}
}

// enforceFloorToBids function does floors enforcement for each bid,
// The bids returned by each partner below bid floor price are rejected and remaining eligible bids are considered for further processing
func enforceFloorToBids(bidRequestWrapper *openrtb_ext.RequestWrapper, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, conversions currency.Conversions, enforceDealFloors bool) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, []error, []*entities.PbsOrtbSeatBid) {
	errs := []error{}
	rejectedBids := []*entities.PbsOrtbSeatBid{}
	impMap := make(map[string]*openrtb_ext.ImpWrapper, bidRequestWrapper.LenImp())

	for _, v := range bidRequestWrapper.GetImp() {
		impMap[v.ID] = v
	}

	for bidderName, seatBid := range seatBids {
		eligibleBids := make([]*entities.PbsOrtbBid, 0, len(seatBid.Bids))
		for _, bid := range seatBid.Bids {

			reqImp, ok := impMap[bid.Bid.ImpID]
			if !ok {
				continue
			}

			requestExt, err := bidRequestWrapper.GetRequestExt()
			if err != nil {
				errs = append(errs, fmt.Errorf("error in getting req extension = %v", err.Error()))
				continue
			}

			if isEnforcementEnabled(requestExt) {
				if hasDealID(bid) && !enforceDealFloors {
					eligibleBids = append(eligibleBids, bid)
					continue
				}

				rate, err := getCurrencyConversionRate(seatBid.Currency, reqImp.BidFloorCur, conversions)
				if err != nil {
					errs = append(errs, fmt.Errorf("error in rate conversion from = %s to %s with bidder %s for impression id %s and bid id %s error = %v", seatBid.Currency, reqImp.BidFloorCur, bidderName, bid.Bid.ImpID, bid.Bid.ID, err.Error()))
					continue
				}

				bidPrice := rate * bid.Bid.Price
				if (bidPrice + floorPrecision) < reqImp.BidFloor {
					rejectedBid := &entities.PbsOrtbSeatBid{
						Currency: seatBid.Currency,
						Seat:     seatBid.Seat,
						Bids:     []*entities.PbsOrtbBid{bid},
					}
					rejectedBids = append(rejectedBids, rejectedBid)
					continue
				}
			}
			eligibleBids = append(eligibleBids, bid)
		}
		seatBids[bidderName].Bids = eligibleBids
	}
	return seatBids, errs, rejectedBids
}

// isEnforcementEnabled check for floors enforcement enabled in request
func isEnforcementEnabled(requestExt *openrtb_ext.RequestExt) bool {
	if floorsExt := getFloorsExt(requestExt); floorsExt != nil {
		return floorsExt.GetEnforcePBS()
	}
	return true
}

// isSignalingSkipped check for floors signalling is skipped due to skip rate
func isSignalingSkipped(requestExt *openrtb_ext.RequestExt) bool {
	if floorsExt := getFloorsExt(requestExt); floorsExt != nil {
		return floorsExt.GetFloorsSkippedFlag()
	}
	return false
}

// getEnforceRateRequest returns enforceRate provided in request
func getEnforceRateRequest(requestExt *openrtb_ext.RequestExt) int {
	if floorsExt := getFloorsExt(requestExt); floorsExt != nil {
		return floorsExt.GetEnforceRate()
	}
	return 0
}

// getEnforceDealsFlag returns FloorDeals flag from req.ext.prebid.floors.enforcement
func getEnforceDealsFlag(requestExt *openrtb_ext.RequestExt) bool {
	if floorsExt := getFloorsExt(requestExt); floorsExt != nil {
		return floorsExt.GetEnforceDealsFlag()
	}
	return false
}

// getFloorsExt returns req.ext.prebid.floors
func getFloorsExt(requestExt *openrtb_ext.RequestExt) *openrtb_ext.PriceFloorRules {
	if requestExt != nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil {
			return prebidExt.Floors
		}
	}
	return nil
}

// isValidImpBidFloorPresent checks if non zero imp.bidfloor is present in request
func isValidImpBidFloorPresent(imp []openrtb2.Imp) bool {
	for i := range imp {
		if imp[i].BidFloor > 0 {
			return true
		}
	}
	return false
}

// isSatisfiedByEnforceRate check enforcements should be done or not based on enforceRate in config and in request
func isSatisfiedByEnforceRate(requestExt *openrtb_ext.RequestExt, configEnforceRate int, f func(int) int) bool {
	requestEnforceRate := getEnforceRateRequest(requestExt)
	enforceRate := f(enforceRateMax)
	satisfiedByRequest := requestEnforceRate == 0 || enforceRate < requestEnforceRate
	satisfiedByAccount := configEnforceRate == 0 || enforceRate < configEnforceRate
	shouldEnforce := satisfiedByRequest && satisfiedByAccount

	return shouldEnforce
}

// hasDealID checks for dealID presence in bid
func hasDealID(bid *entities.PbsOrtbBid) bool {
	if bid != nil && bid.Bid != nil && bid.Bid.DealID != "" {
		return true
	}
	return false
}

// getCurrencyConversionRate gets conversion rate in case floor currency and seatBid currency are not same
func getCurrencyConversionRate(seatBidCur, reqImpCur string, conversions currency.Conversions) (float64, error) {
	rate := 1.0
	if seatBidCur != reqImpCur {
		return conversions.GetRate(seatBidCur, reqImpCur)
	} else {
		return rate, nil
	}
}

// updateBidExtWithFloors updates floors related details in bid extension
func updateBidExtWithFloors(reqImp *openrtb_ext.ImpWrapper, bid *entities.PbsOrtbBid, floorCurrency string) {
	impExt, err := reqImp.GetImpExt()
	if err != nil {
		return
	}

	var bidExtFloors openrtb_ext.ExtBidPrebidFloors
	prebidExt := impExt.GetPrebid()
	if prebidExt == nil || prebidExt.Floors == nil {
		if reqImp.BidFloor > 0 {
			bidExtFloors.FloorValue = reqImp.BidFloor
			bidExtFloors.FloorCurrency = reqImp.BidFloorCur
			bid.BidFloors = &bidExtFloors
		}
	} else {
		bidExtFloors.FloorRule = prebidExt.Floors.FloorRule
		bidExtFloors.FloorRuleValue = prebidExt.Floors.FloorRuleValue
		bidExtFloors.FloorValue = prebidExt.Floors.FloorValue
		bidExtFloors.FloorCurrency = floorCurrency
		bid.BidFloors = &bidExtFloors
	}
}
