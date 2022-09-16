package exchange

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/floors"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Check for Floors enforcement for deals,
// In case bid wit DealID present and enforceDealFloors = false then bid floor enforcement should be skipped
func checkDealsForEnforcement(bid *pbsOrtbBid, enforceDealFloors bool) *pbsOrtbBid {
	if bid.bid.DealID != "" && !enforceDealFloors {
		return bid
	}
	return nil
}

// Get conversion rate in case floor currency and seatBid currency are not same
func getCurrencyConversionRate(seatBidCur, reqImpCur string, conversions currency.Conversions) (float64, error) {
	rate := 1.0
	if seatBidCur != reqImpCur {
		return conversions.GetRate(seatBidCur, reqImpCur)
	} else {
		return rate, nil
	}
}

// enforceFloorToBids function does floors enforcement for each bid.
//  The bids returned by each partner below bid floor price are rejected and remaining eligible bids are considered for further processing
func enforceFloorToBids(bidRequest *openrtb2.BidRequest, seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, conversions currency.Conversions, enforceDealFloors bool) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, []error) {
	errs := []error{}
	impMap := make(map[string]openrtb2.Imp, len(bidRequest.Imp))

	//Maintaining BidRequest Impression Map
	for i := range bidRequest.Imp {
		impMap[bidRequest.Imp[i].ID] = bidRequest.Imp[i]
	}

	for bidderName, seatBid := range seatBids {
		eligibleBids := make([]*pbsOrtbBid, 0)
		for _, bid := range seatBid.bids {
			retBid := checkDealsForEnforcement(bid, enforceDealFloors)
			if retBid != nil {
				eligibleBids = append(eligibleBids, retBid)
				continue
			}

			reqImp, ok := impMap[bid.bid.ImpID]
			if ok {
				reqImpCur := reqImp.BidFloorCur
				if reqImpCur == "" {
					reqImpCur = bidRequest.Cur[0]
				}
				rate, err := getCurrencyConversionRate(seatBid.currency, reqImpCur, conversions)
				if err == nil {
					bidPrice := rate * bid.bid.Price
					if reqImp.BidFloor > bidPrice {
						errs = append(errs, fmt.Errorf("bid rejected [bid ID: %s] reason: bid price value %.4f %s is less than bidFloor value %.4f %s for impression id %s bidder %s", bid.bid.ID, bidPrice, reqImpCur, reqImp.BidFloor, reqImpCur, bid.bid.ImpID, bidderName))
					} else {
						eligibleBids = append(eligibleBids, bid)
					}
				} else {
					errMsg := fmt.Errorf("Error in rate conversion from = %s to %s with bidder %s for impression id %s and bid id %s", seatBid.currency, reqImpCur, bidderName, bid.bid.ImpID, bid.bid.ID)
					glog.Errorf(errMsg.Error())
					errs = append(errs, errMsg)

				}
			}
		}
		seatBids[bidderName].bids = eligibleBids
	}
	return seatBids, errs
}

// selectFloorsAndModifyImp function does singanlling of floors,
// Internally validation of floors parameters and validation of rules is done,
// Based on number of modelGroups and modelWeight, one model is selected and imp.bidfloor and imp.bidfloorcur is updated
func selectFloorsAndModifyImp(r *AuctionRequest, floor config.PriceFloors, conversions currency.Conversions, responseDebugAllow bool) []error {

	var errs []error
	requestExt, err := r.BidRequestWrapper.GetRequestExt()
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	prebidExt := requestExt.GetPrebid()
	if floor.Enabled && prebidExt.Floors != nil && prebidExt.Floors.GetEnabled() {
		errs = floors.ModifyImpsWithFloors(prebidExt.Floors, r.BidRequestWrapper.BidRequest, conversions)
		requestExt.SetPrebid(prebidExt)
		err := r.BidRequestWrapper.RebuildRequest()
		if err != nil {
			errs = append(errs, err)
		}

		if responseDebugAllow {
			updatedBidReq, _ := json.Marshal(r.BidRequestWrapper.BidRequest)
			//save updated request after floors signalling
			r.UpdatedBidRequest = updatedBidReq
		}
	}
	return errs
}

// enforceFloors function does floors enforcement
func enforceFloors(r *AuctionRequest, seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, floor config.PriceFloors, conversions currency.Conversions, responseDebugAllow bool) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, []error) {

	rejectionsErrs := []error{}

	requestExt, err := r.BidRequestWrapper.GetRequestExt()
	if err != nil {
		rejectionsErrs = append(rejectionsErrs, err)
		return seatBids, rejectionsErrs
	}
	prebidExt := requestExt.GetPrebid()
	if floor.Enabled && prebidExt.Floors != nil && prebidExt.Floors.GetEnabled() {
		if floors.ShouldEnforce(r.BidRequestWrapper.BidRequest, prebidExt.Floors, floor.EnforceFloorsRate, rand.Intn) {
			var enforceDealFloors bool
			if prebidExt != nil && prebidExt.Floors != nil && prebidExt.Floors.Enforcement != nil && prebidExt.Floors.Enforcement.FloorDeals != nil {
				enforceDealFloors = *prebidExt.Floors.Enforcement.FloorDeals && floor.EnforceDealFloors
			}
			seatBids, rejectionsErrs = enforceFloorToBids(r.BidRequestWrapper.BidRequest, seatBids, conversions, enforceDealFloors)
		}
		requestExt.SetPrebid(prebidExt)
		err = r.BidRequestWrapper.RebuildRequest()
		if err != nil {
			rejectionsErrs = append(rejectionsErrs, err)
			return seatBids, rejectionsErrs
		}

		if responseDebugAllow {
			updatedBidReq, _ := json.Marshal(r.BidRequestWrapper.BidRequest)
			//save updated request after floors enforcement
			r.UpdatedBidRequest = updatedBidReq
		}
	}
	return seatBids, rejectionsErrs
}
