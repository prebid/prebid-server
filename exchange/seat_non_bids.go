package exchange

import (
	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type nonBids struct {
	seatNonBidsMap map[string][]openrtb_ext.NonBid
}

// addBid is not thread safe as we are initializing and writing to map
func (snb *nonBids) addBid(bid *entities.PbsOrtbBid, nonBidReason int, seat string) {
	if bid == nil || bid.Bid == nil {
		return
	}
	if snb.seatNonBidsMap == nil {
		snb.seatNonBidsMap = make(map[string][]openrtb_ext.NonBid)
	}
	nonBid := openrtb_ext.NonBid{
		ImpId:      bid.Bid.ImpID,
		StatusCode: nonBidReason,
		Ext: openrtb_ext.NonBidExt{
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

	snb.seatNonBidsMap[seat] = append(snb.seatNonBidsMap[seat], nonBid)
}

func (snb *nonBids) get() []openrtb_ext.SeatNonBid {
	if snb == nil {
		return nil
	}
	var seatNonBid []openrtb_ext.SeatNonBid
	for seat, nonBids := range snb.seatNonBidsMap {
		seatNonBid = append(seatNonBid, openrtb_ext.SeatNonBid{
			Seat:   seat,
			NonBid: nonBids,
		})
	}
	return seatNonBid
}
