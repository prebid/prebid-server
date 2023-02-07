package floors

import (
	"fmt"
	"math/rand"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// RejectedBid defines the contract for bid rejection errors due to floors enforcement
type RejectedBid struct {
	Bid             *openrtb2.Bid `json:"bid,omitempty"`
	RejectionReason int           `json:"rejectreason,omitempty"`
	BidderName      string        `json:"biddername,omitempty"`
}

func IsImpBidfloorPresentInRequest(bidRequest *openrtb2.BidRequest) bool {
	for i := range bidRequest.Imp {
		if bidRequest.Imp[i].BidFloor > 0 {
			return true
		}
	}
	return false
}

func shouldEnforceFloors(bidRequest *openrtb2.BidRequest, floorExt *openrtb_ext.PriceFloorRules, configEnforceRate int, f func(int) int) (bool, bool) {

	updateReqExt := false
	if floorExt != nil && floorExt.Skipped != nil && *floorExt.Skipped {
		return !*floorExt.Skipped, updateReqExt
	}

	if floorExt != nil && floorExt.Enforcement != nil && floorExt.Enforcement.EnforcePBS != nil && !*floorExt.Enforcement.EnforcePBS {
		return *floorExt.Enforcement.EnforcePBS, updateReqExt
	}

	if floorExt != nil && floorExt.Enforcement != nil && floorExt.Enforcement.EnforceRate > 0 {
		configEnforceRate = floorExt.Enforcement.EnforceRate
	}

	shouldEnforce := configEnforceRate > f(enforceRateMax)
	if floorExt == nil {
		floorExt = new(openrtb_ext.PriceFloorRules)
	}

	if floorExt.Enforcement == nil {
		floorExt.Enforcement = new(openrtb_ext.PriceFloorEnforcement)
	}

	if floorExt.Enforcement.EnforcePBS == nil {
		updateReqExt = true
		floorExt.Enforcement.EnforcePBS = new(bool)
	}
	if *floorExt.Enforcement.EnforcePBS != shouldEnforce {
		updateReqExt = true
	}
	*floorExt.Enforcement.EnforcePBS = shouldEnforce
	return shouldEnforce, updateReqExt
}

// Check for Floors enforcement for deals,
// In case bid wit DealID present and enforceDealFloors = false then bid floor enforcement should be skipped
func checkDealsForEnforcement(bid *entities.PbsOrtbBid, enforceDealFloors bool) *entities.PbsOrtbBid {
	if bid != nil && bid.Bid != nil && bid.Bid.DealID != "" && !enforceDealFloors {
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

// enforceFloorToBids function does floors enforcement for each bid
// The bids returned by each partner below bid floor price are rejected and remaining eligible bids are considered for further processing
func enforceFloorToBids(bidRequest *openrtb2.BidRequest, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, conversions currency.Conversions, enforceDealFloors bool) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, []error, []RejectedBid) {
	errs := []error{}
	rejectedBids := []RejectedBid{}
	impMap := make(map[string]openrtb2.Imp, len(bidRequest.Imp))

	//Maintaining BidRequest Impression Map
	for i := range bidRequest.Imp {
		impMap[bidRequest.Imp[i].ID] = bidRequest.Imp[i]
	}

	for bidderName, seatBid := range seatBids {
		eligibleBids := make([]*entities.PbsOrtbBid, 0, len(seatBid.Bids))
		for _, bid := range seatBid.Bids {
			retBid := checkDealsForEnforcement(bid, enforceDealFloors)
			if retBid != nil {
				eligibleBids = append(eligibleBids, retBid)
				continue
			}

			reqImp, ok := impMap[bid.Bid.ImpID]
			if ok {
				reqImpCur := reqImp.BidFloorCur
				if reqImpCur == "" {
					if bidRequest.Cur != nil {
						reqImpCur = bidRequest.Cur[0]
					} else {
						reqImpCur = "USD"
					}
				}
				rate, err := getCurrencyConversionRate(seatBid.Currency, reqImpCur, conversions)
				if err == nil {
					bidPrice := rate * bid.Bid.Price
					if reqImp.BidFloor > bidPrice {
						rejectedBid := RejectedBid{
							Bid:             bid.Bid,
							BidderName:      string(bidderName),
							RejectionReason: errortypes.BidRejectionFloorsErrorCode,
						}
						rejectedBids = append(rejectedBids, rejectedBid)
						errs = append(errs, fmt.Errorf("bid rejected [bid ID: %s] reason: bid price value %.4f %s is less than bidFloor value %.4f %s for impression id %s bidder %s", bid.Bid.ID, bidPrice, reqImpCur, reqImp.BidFloor, reqImpCur, bid.Bid.ImpID, bidderName))
					} else {
						eligibleBids = append(eligibleBids, bid)
					}
				} else {
					errMsg := fmt.Errorf("Error in rate conversion from = %s to %s with bidder %s for impression id %s and bid id %s", seatBid.Currency, reqImpCur, bidderName, bid.Bid.ImpID, bid.Bid.ID)
					glog.Errorf(errMsg.Error())
					errs = append(errs, errMsg)

				}
			}
		}
		seatBids[bidderName].Bids = eligibleBids
	}
	return seatBids, errs, rejectedBids
}

// EnforceFloors function does floors enforcement
func EnforceFloors(bidRequestWrapper *openrtb_ext.RequestWrapper, bidRequest *openrtb2.BidRequest, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, priceFloorsCfg config.AccountPriceFloors, conversions currency.Conversions) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, []error, []RejectedBid) {

	rejectionsErrs := []error{}
	rejecteBids := []RejectedBid{}
	if bidRequestWrapper == nil || bidRequest == nil {
		return seatBids, rejectionsErrs, rejecteBids
	}

	requestExt, err := bidRequestWrapper.GetRequestExt()
	if err != nil {
		rejectionsErrs = append(rejectionsErrs, err)
		return seatBids, rejectionsErrs, rejecteBids
	}

	prebidExt := requestExt.GetPrebid()
	reqFloorEnable := getFloorsFlagFromReqExt(prebidExt)
	if reqFloorEnable {
		var enforceDealFloors bool
		var floorsEnfocement bool
		var updateReqExt bool
		floorsEnfocement = IsImpBidfloorPresentInRequest(bidRequest)
		if prebidExt != nil && floorsEnfocement {
			if floorsEnfocement, updateReqExt = shouldEnforceFloors(bidRequest, prebidExt.Floors, priceFloorsCfg.EnforceFloorRate, rand.Intn); floorsEnfocement {
				enforceDealFloors = priceFloorsCfg.EnforceDealFloors && getEnforceDealsFlag(prebidExt.Floors)
			}
		}

		if floorsEnfocement {
			seatBids, rejectionsErrs, rejecteBids = enforceFloorToBids(bidRequest, seatBids, conversions, enforceDealFloors)
		}

		if updateReqExt {
			requestExt.SetPrebid(prebidExt)
			err = bidRequestWrapper.RebuildRequestExt()
			if err != nil {
				rejectionsErrs = append(rejectionsErrs, err)
				return seatBids, rejectionsErrs, rejecteBids
			}
		}
	}
	return seatBids, rejectionsErrs, rejecteBids
}
