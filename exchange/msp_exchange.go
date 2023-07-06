package exchange

import (
	"encoding/json"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	MSP_SEAT_IN_HOUSE = "msp-in-house"
)

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
