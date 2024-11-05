package exchange

import (
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type SeatNonBidBuilder map[string][]openrtb_ext.NonBid

// rejectBid appends a non bid object to the builder based on a bid
func (b SeatNonBidBuilder) rejectBid(bid *entities.PbsOrtbBid, nonBidReason int, seat string) {
	if b == nil || bid == nil || bid.Bid == nil {
		return
	}

	nonBid := openrtb_ext.NonBid{
		ImpId:      bid.Bid.ImpID,
		StatusCode: nonBidReason,
		Ext: &openrtb_ext.NonBidExt{
			Prebid: openrtb_ext.ExtResponseNonBidPrebid{Bid: openrtb_ext.NonBidObject{
				Price:          bid.Bid.Price,
				ADomain:        bid.Bid.ADomain,
				CatTax:         bid.Bid.CatTax,
				Cat:            bid.Bid.Cat,
				DealID:         bid.Bid.DealID,
				W:              bid.Bid.W,
				H:              bid.Bid.H,
				Dur:            bid.Bid.Dur,
				MType:          bid.Bid.MType,
				OriginalBidCPM: bid.OriginalBidCPM,
				OriginalBidCur: bid.OriginalBidCur,
			}},
		},
	}
	b[seat] = append(b[seat], nonBid)
}

// rejectImps appends a non bid object to the builder for every specified imp
func (b SeatNonBidBuilder) rejectImps(impIds []string, nonBidReason NonBidReason, seat string) {
	nonBids := []openrtb_ext.NonBid{}
	for _, impId := range impIds {
		nonBid := openrtb_ext.NonBid{
			ImpId:      impId,
			StatusCode: int(nonBidReason),
		}
		nonBids = append(nonBids, nonBid)
	}

	if len(nonBids) > 0 {
		b[seat] = append(b[seat], nonBids...)
	}
}

// slice transforms the seat non bid map into a slice of SeatNonBid objects representing the non-bids for each seat
func (b SeatNonBidBuilder) Slice() []openrtb_ext.SeatNonBid {
	seatNonBid := make([]openrtb_ext.SeatNonBid, 0)
	for seat, nonBids := range b {
		seatNonBid = append(seatNonBid, openrtb_ext.SeatNonBid{
			Seat:   seat,
			NonBid: nonBids,
		})
	}
	return seatNonBid
}

// append adds the nonBids from the input nonBids to the current nonBids.
// This method is not thread safe as we are initializing and writing to map
func (b SeatNonBidBuilder) append(nonBids ...SeatNonBidBuilder) {
	if b == nil {
		return
	}
	for _, nonBid := range nonBids {
		for seat, nonBids := range nonBid {
			b[seat] = append(b[seat], nonBids...)
		}
	}
}
