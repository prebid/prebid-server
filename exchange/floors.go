package exchange

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/floors"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// RejectedBid defines the contract for bid rejection errors due to floors enforcement
type RejectedBid struct {
	Bid             *openrtb2.Bid `json:"bid,omitempty"`
	RejectionReason int           `json:"rejectreason,omitempty"`
	BidderName      string        `json:"biddername,omitempty"`
}

// Check for Floors enforcement for deals,
// In case bid wit DealID present and enforceDealFloors = false then bid floor enforcement should be skipped
func checkDealsForEnforcement(bid *pbsOrtbBid, enforceDealFloors bool) *pbsOrtbBid {
	if bid != nil && bid.bid != nil && bid.bid.DealID != "" && !enforceDealFloors {
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
func enforceFloorToBids(bidRequest *openrtb2.BidRequest, seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, conversions currency.Conversions, enforceDealFloors bool) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, []error, []RejectedBid) {
	errs := []error{}
	rejectedBids := []RejectedBid{}
	impMap := make(map[string]openrtb2.Imp, len(bidRequest.Imp))

	//Maintaining BidRequest Impression Map
	for i := range bidRequest.Imp {
		impMap[bidRequest.Imp[i].ID] = bidRequest.Imp[i]
	}

	for bidderName, seatBid := range seatBids {
		eligibleBids := make([]*pbsOrtbBid, 0, len(seatBid.bids))
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
					if bidRequest.Cur != nil {
						reqImpCur = bidRequest.Cur[0]
					} else {
						reqImpCur = "USD"
					}
				}
				rate, err := getCurrencyConversionRate(seatBid.currency, reqImpCur, conversions)
				if err == nil {
					bidPrice := rate * bid.bid.Price
					if reqImp.BidFloor > bidPrice {
						rejectedBid := RejectedBid{
							Bid:             bid.bid,
							BidderName:      string(bidderName),
							RejectionReason: errortypes.BidRejectionFloorsErrorCode,
						}
						rejectedBids = append(rejectedBids, rejectedBid)
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
	return seatBids, errs, rejectedBids
}

// selectFloorsAndModifyImp function does singanlling of floors,
// Internally validation of floors parameters and validation of rules is done,
// Based on number of modelGroups and modelWeight, one model is selected and imp.bidfloor and imp.bidfloorcur is updated
func selectFloorsAndModifyImp(r *AuctionRequest, floor config.PriceFloors, conversions currency.Conversions, responseDebugAllow bool) []error {
	var errs []error
	if r == nil || r.BidRequestWrapper == nil {
		return errs
	}

	requestExt, err := r.BidRequestWrapper.GetRequestExt()
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	prebidExt := requestExt.GetPrebid()
	if floor.Enabled && prebidExt != nil && prebidExt.Floors != nil && prebidExt.Floors.GetEnabled() {
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

// getFloorsFlagFromReqExt returns floors enabled flag,
// if floors enabled flag is not provided in request extesion, by default treated as true
func getFloorsFlagFromReqExt(prebidExt *openrtb_ext.ExtRequestPrebid) bool {
	floorEnabled := true
	if prebidExt == nil || prebidExt.Floors == nil || prebidExt.Floors.Enabled == nil {
		return floorEnabled
	}
	return *prebidExt.Floors.Enabled
}

func getEnforceDealsFlag(Floors *openrtb_ext.PriceFloorRules) bool {
	return Floors != nil && Floors.Enforcement != nil && Floors.Enforcement.FloorDeals != nil && *Floors.Enforcement.FloorDeals
}

// eneforceFloors function does floors enforcement
func enforceFloors(r *AuctionRequest, seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, floor config.PriceFloors, conversions currency.Conversions, responseDebugAllow bool) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, []error, []RejectedBid) {

	rejectionsErrs := []error{}
	rejecteBids := []RejectedBid{}
	if r == nil || r.BidRequestWrapper == nil {
		return seatBids, rejectionsErrs, rejecteBids
	}

	requestExt, err := r.BidRequestWrapper.GetRequestExt()
	if err != nil {
		rejectionsErrs = append(rejectionsErrs, err)
		return seatBids, rejectionsErrs, rejecteBids
	}
	prebidExt := requestExt.GetPrebid()
	reqFloorEnable := getFloorsFlagFromReqExt(prebidExt)
	if floor.Enabled && reqFloorEnable {
		var enforceDealFloors bool
		var floorsEnfocement bool
		floorsEnfocement = floors.RequestHasFloors(r.BidRequestWrapper.BidRequest)
		if prebidExt != nil && floorsEnfocement {
			if floorsEnfocement = floors.ShouldEnforce(r.BidRequestWrapper.BidRequest, prebidExt.Floors, floor.EnforceFloorsRate, rand.Intn); floorsEnfocement {
				enforceDealFloors = floor.EnforceDealFloors && getEnforceDealsFlag(prebidExt.Floors)
			}
		}

		if floorsEnfocement {
			seatBids, rejectionsErrs, rejecteBids = enforceFloorToBids(r.BidRequestWrapper.BidRequest, seatBids, conversions, enforceDealFloors)
		}
		requestExt.SetPrebid(prebidExt)
		err = r.BidRequestWrapper.RebuildRequest()
		if err != nil {
			rejectionsErrs = append(rejectionsErrs, err)
			return seatBids, rejectionsErrs, rejecteBids
		}

		if responseDebugAllow {
			updatedBidReq, _ := json.Marshal(r.BidRequestWrapper.BidRequest)
			//save updated request after floors enforcement
			r.UpdatedBidRequest = updatedBidReq
		}
	}
	return seatBids, rejectionsErrs, rejecteBids
}
