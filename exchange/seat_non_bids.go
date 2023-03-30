package exchange

import (
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type seatNonBids struct {
	seatNonBidsMap map[string][]openrtb_ext.NonBid
}

func newSeatNonBids() seatNonBids {
	return seatNonBids{
		seatNonBidsMap: make(map[string][]openrtb_ext.NonBid),
	}
}

func (snb *seatNonBids) add(bid *entities.PbsOrtbBid, nonBidReason int, seat string) {
	if bid == nil || bid.Bid == nil {
		return
	}
	if snb.seatNonBidsMap == nil {
		snb.seatNonBidsMap = make(map[string][]openrtb_ext.NonBid)
	}
	nonBid := openrtb_ext.NonBid{
		ImpId:      bid.Bid.ImpID,
		StatusCode: nonBidReason, //
		Ext: openrtb_ext.NonBidExt{
			Prebid: openrtb_ext.Prebid{Bid: openrtb_ext.Bid{
				Bid: openrtb2.Bid{
					Price:   bid.Bid.Price,
					ADomain: bid.Bid.ADomain,
					CatTax:  bid.Bid.CatTax,
					Cat:     bid.Bid.Cat,
					DealID:  bid.Bid.DealID,
					W:       bid.Bid.W,
					H:       bid.Bid.H,
					Dur:     bid.Bid.Dur,
					MType:   bid.Bid.MType,
				},
			}},
		},
	}

	nonBids := snb.seatNonBidsMap[seat]

	if nonBids == nil {
		snb.seatNonBidsMap[seat] = []openrtb_ext.NonBid{nonBid}
	} else {
		snb.seatNonBidsMap[seat] = append(nonBids, nonBid)
	}
}

func (snb *seatNonBids) get() []openrtb_ext.SeatNonBid {
	var seatNonBid []openrtb_ext.SeatNonBid
	if snb.seatNonBidsMap != nil {
		for seat, nonBids := range snb.seatNonBidsMap {
			seatNonBid = append(seatNonBid, openrtb_ext.SeatNonBid{
				Seat:   seat,
				NonBid: nonBids,
			})
		}
	}
	return seatNonBid
}
