package floors

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// EnforceFloors does floors enforcement for bids from all bidders based on floors provided in request, account level floors config
func EnforceFloors(bidRequestWrapper *openrtb_ext.RequestWrapper, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, account config.Account, conversions currency.Conversions) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, []error, []*entities.PbsOrtbSeatBid) {
	rejectionErrs := []error{}
	rejectedBids := []*entities.PbsOrtbSeatBid{}

	if isPriceFloorsDisabled(account, bidRequestWrapper) {
		return seatBids, []error{errors.New("Floors feature is disabled at account or in the request")}, rejectedBids
	}

	if !isFloorsSignallingSkipped(bidRequestWrapper) && isValidImpBidFloorPresent(bidRequestWrapper.BidRequest) {
		if enforceFloors := isSatisfiedByEnforceRate(bidRequestWrapper, account.PriceFloors.EnforceFloorsRate, rand.Intn); enforceFloors {
			enforceDealFloors := account.PriceFloors.EnforceDealFloors && getEnforceDealsFlag(bidRequestWrapper)

			seatBids, rejectionErrs, rejectedBids = enforceFloorToBids(bidRequestWrapper, seatBids, conversions, enforceDealFloors)
		}
	}
	return seatBids, rejectionErrs, rejectedBids
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

			reqImpCur := reqImp.BidFloorCur
			if reqImpCur == "" {
				reqImpCur = "USD"
				if bidRequestWrapper.Cur != nil {
					reqImpCur = bidRequestWrapper.Cur[0]
				}
			}
			updateBidExtWithFloors(reqImp, bid, reqImpCur)

			if !isPriceFloorsEnforcementDisabled(bidRequestWrapper) {
				dealBid := checkDealBidForEnforcement(bid, enforceDealFloors)
				if dealBid != nil {
					eligibleBids = append(eligibleBids, dealBid)
					continue
				}

				rate, err := getCurrencyConversionRate(seatBid.Currency, reqImpCur, conversions)
				if err != nil {
					errs = append(errs, fmt.Errorf("error in rate conversion from = %s to %s with bidder %s for impression id %s and bid id %s error = %v", seatBid.Currency, reqImpCur, bidderName, bid.Bid.ImpID, bid.Bid.ID, err.Error()))
					continue
				}

				bidPrice := rate * bid.Bid.Price

				if reqImp.BidFloor > bidPrice {
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

// isPriceFloorsEnforcementDisabled check for floors are disabled at request using Floors.Enforcement.EnforcePBS flag
func isPriceFloorsEnforcementDisabled(bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil && !prebidExt.Floors.GetEnforcePBS() {
			return true
		}
	}
	return false
}

// isFloorsSignallingSkipped check for floors signalling is skipped due to skip rate
func isFloorsSignallingSkipped(bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil {
			return prebidExt.Floors.GetFloorsSkippedFlag()
		}
	}
	return false
}

// getEnforceRateRequest returns enforceRate provided in request
func getEnforceRateRequest(bidRequestWrapper *openrtb_ext.RequestWrapper) int {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil {
			return prebidExt.Floors.GetEnforceRate()
		}
	}
	return 0
}

// getEnforceDealsFlag returns FloorDeals flag from req.ext.prebid.floors.enforcement
func getEnforceDealsFlag(bidRequestWrapper *openrtb_ext.RequestWrapper) bool {
	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err == nil {
		if prebidExt := requestExt.GetPrebid(); prebidExt != nil && prebidExt.Floors != nil {
			return prebidExt.Floors.GetEnforceDealsFlag()
		}
	}
	return false
}

// isValidImpBidFloorPresent checks if non zero imp.bidfloor is present in request
func isValidImpBidFloorPresent(bidRequest *openrtb2.BidRequest) bool {
	for i := range bidRequest.Imp {
		if bidRequest.Imp[i].BidFloor > 0 {
			return true
		}
	}
	return false
}

// isSatisfiedByEnforceRate check enforcements should be done or not based on enforceRate in config and in request
func isSatisfiedByEnforceRate(bidRequestWrapper *openrtb_ext.RequestWrapper, configEnforceRate int, f func(int) int) bool {
	requestEnforceRate := getEnforceRateRequest(bidRequestWrapper)
	enforceRate := f(enforceRateMax)
	satisfiedByRequest := requestEnforceRate == 0 || enforceRate < requestEnforceRate
	satisfiedByAccount := configEnforceRate == 0 || enforceRate < configEnforceRate
	shouldEnforce := satisfiedByRequest && satisfiedByAccount

	return shouldEnforce
}

// checkDealBidForEnforcement checks for floors enforcement for deal bids
func checkDealBidForEnforcement(bid *entities.PbsOrtbBid, enforceDealFloors bool) *entities.PbsOrtbBid {
	if !enforceDealFloors && bid != nil && bid.Bid != nil && bid.Bid.DealID != "" {
		return bid
	}
	return nil
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

	var bidExtFloors openrtb_ext.ExtBidFloors
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
