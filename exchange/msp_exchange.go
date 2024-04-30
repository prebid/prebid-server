package exchange

import (
	"encoding/json"
	"math"
	"time"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
)

const (
	MSP_SEAT_IN_HOUSE = "msp-in-house"
)

type MSBExt struct {
	MSB MSBConfig `json:"msb"`
}

type MSBConfig struct {
	LastPeek MSBLastPeekConfig `json:"last_peek"`
}

type MSBLastPeekConfig struct {
	// start peek available bids after this
	PeekStartTimeMilliSeconds int64 `json:"peek_start_time_miliseconds"`
	// bidder -> FloorMult
	// set floor for last peek bidder = current_available_max_bid_price * FloorMult to:
	// 1. break tie
	// 2. gain more revenue
	PeekBidderFloorMultMap map[string]float64 `json:"peek_bidder_floor_mult_map"`
}

/*
MSB feature controller(req.ext) example:
{
    "ext": {
        "msb": {
            "last_peek": {
                "peek_start_time_miliseconds": 900,
                "peek_bidder_floor_mult_map": {
                    "msp_google": 1.01,
                    "msp_nova": 1.01
                }
            }
        }
    }
}
MSP server is response for adding above MSB info to requests for certain traffic/placement/exps. peek_bidder_floor_mult_map controls which bidders
are there for different peek tiers

In the bove example
	there are two peek tiers for bidders:
	1. last peek tier
	2. the rest(normal tier)

	after all reponses are ready for normal tier or timeout=peek_start_time_miliseconds(900ms), for bidders in last peek tier(msp_google and msp_nova) peek available responses from normal tier,
	get max_available_bid_prices_for_normal_tier and set floor =  max_available_bid_prices_for_normal_tier * peek_bidder_floor_mult_map[bidder]
*/

type MSPFloor struct {
	Floor float64 `json:"floor"`
}

var mspBidders = map[openrtb_ext.BidderName]int{
	openrtb_ext.BidderMspGoogle:  1,
	openrtb_ext.BidderMspFbAlpha: 1,
	openrtb_ext.BidderMspFbBeta:  1,
	openrtb_ext.BidderMspFbGamma: 1,
	openrtb_ext.BidderMspNova:    1,
}

func mspUpdateStoredAuctionResponse(r *AuctionRequest) bool {
	if len(r.StoredAuctionResponses) > 0 {
		if rawSeatBid, ok := r.StoredAuctionResponses[r.BidRequestWrapper.Imp[0].ID]; ok {
			var seatBids []openrtb2.SeatBid

			err := json.Unmarshal(rawSeatBid, &seatBids)
			if err == nil && len(seatBids) > 0 && seatBids[0].Seat == MSP_SEAT_IN_HOUSE {
				// when price is the same, randomly choose one
				swapIdx := time.Now().Second() % len(seatBids[0].Bid)
				seatBids[0].Bid[0], seatBids[0].Bid[swapIdx] = seatBids[0].Bid[swapIdx], seatBids[0].Bid[0]
				updatedJson, _ := json.Marshal(seatBids)
				r.StoredAuctionResponses[r.BidRequestWrapper.Imp[0].ID] = updatedJson
				return true
			}
		}
	}

	return false
}

func mspApplyStoredAuctionResponse(r *AuctionRequest) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
	*openrtb_ext.Fledge,
	[]openrtb_ext.BidderName,
	error, bool) {
	adapterBids, fledge, liveAdapters, err := buildStoredAuctionResponse(r.StoredAuctionResponses)
	return adapterBids, fledge, liveAdapters, err, err == nil
}

func mspPostProcessAuction(
	r *AuctionRequest,
	liveAdapters []openrtb_ext.BidderName,
	adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
	adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	fledge *openrtb_ext.Fledge,
	anyBidsReturned bool,
	shouldMspBackfillBids bool) (map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
	*openrtb_ext.Fledge,
	[]openrtb_ext.BidderName,
	error, bool) {
	// TODO: freqcap, blocking

	if shouldMspBackfillBids && !anyBidsReturned {
		return mspApplyStoredAuctionResponse(r)
	}

	return adapterBids, fledge, liveAdapters, nil, anyBidsReturned
}

// peek max available(ready) bid price from channel without comsuming
func peekChannelAvailableMaxBidPriceWithinTimeout(peekTier string, chBids chan *bidResponseWrapper, lastPeekConfig MSBLastPeekConfig, totalNormalBidders int) float64 {
	timeout := time.After(time.Duration(lastPeekConfig.PeekStartTimeMilliSeconds) * time.Millisecond)
	maxPrice := 0.0
	hasData := true
	peekedRespList := []*bidResponseWrapper{}
	availableBidders := []string{}
	// keep consuming message from channel until all normal bidder requests are collected or timeout reaches
	for hasData {
		select {
		case resp, ok := <-chBids:
			if !ok {
				hasData = false
			} else {
				for _, bids := range resp.adapterSeatBids {
					for _, bid := range bids.Bids {
						if bid.Bid != nil {
							maxPrice = math.Max(maxPrice, bid.Bid.Price)
						}
					}
				}
				peekedRespList = append(peekedRespList, resp)
				availableBidders = append(availableBidders, resp.bidder.String())
				if len(peekedRespList) == totalNormalBidders {
					hasData = false
				}
			}

		case <-timeout:
			hasData = false
		}

	}
	// push message back
	for _, resp := range peekedRespList {
		chBids <- resp
	}
	glog.Infof("MSB tier %s, peeked from available bidders %v, current max bid price: %f", peekTier, availableBidders, maxPrice)
	return maxPrice
}

func mspUpdateLastPeekBiddersRequest(
	chBids chan *bidResponseWrapper,
	lastPeekBidderRequests []BidderRequest,
	lastPeekConfig MSBLastPeekConfig,
	totalNormalBidders int,
) []BidderRequest {
	maxPrice := peekChannelAvailableMaxBidPriceWithinTimeout("lastPeek", chBids, lastPeekConfig, totalNormalBidders)
	for reqIdx := range lastPeekBidderRequests {
		mult := lastPeekConfig.PeekBidderFloorMultMap[lastPeekBidderRequests[reqIdx].BidderName.String()]
		updatedFloor := mult * maxPrice
		for idx := range lastPeekBidderRequests[reqIdx].BidRequest.Imp {
			bidder := &lastPeekBidderRequests[reqIdx]
			// update req.imp.bidfloor
			bidder.BidRequest.Imp[idx].BidFloor = math.Max(bidder.BidRequest.Imp[idx].BidFloor, updatedFloor)

			// for msp bidders, update req.imp.ext.bidder.floor which is the source of truth for msp bidder's floor and
			// will be updated/overwritten later by msp module stage: https://github.com/ParticleMedia/msp/blob/master/pkg/modules/dam_buckets/module/hook_bidder_request.go#L69
			if _, found := mspBidders[bidder.BidderName]; found {
				extBytes, err := jsonObject(bidder.BidRequest.Imp[idx].Ext, "bidder")
				if err == nil {
					var impExt MSPFloor
					err = json.Unmarshal(extBytes, &impExt)
					if err == nil {
						impExt.Floor = math.Max(impExt.Floor, updatedFloor)
						updatedBytes, _ := json.Marshal(impExt)
						updatedBidderBytes, _ := jsonpatch.MergePatch(extBytes, updatedBytes)
						updatedExtBytes, _ := jsonparser.Set(bidder.BidRequest.Imp[idx].Ext, updatedBidderBytes, "bidder")
						bidder.BidRequest.Imp[idx].Ext = updatedExtBytes
					}
				}
			}
		}
	}
	return lastPeekBidderRequests
}

func jsonObject(data []byte, keys ...string) ([]byte, error) {
	if result, dataType, _, err := jsonparser.Get(data, keys...); err == nil && dataType == jsonparser.Object {
		return result, nil
	} else {
		return nil, err
	}
}

func extractMSBInfoBidders(reqList []BidderRequest) MSBConfig {
	if len(reqList) > 0 {
		return ExtractMSBInfoReq(reqList[0].BidRequest)
	}

	return MSBConfig{}
}

func ExtractMSBInfoReq(req *openrtb2.BidRequest) MSBConfig {
	var config MSBExt
	if req != nil {
		err := json.Unmarshal(req.Ext, &config)
		if err != nil {
			glog.Error("MSB extract bidder config:", err)
		}
	}
	return config.MSB
}
