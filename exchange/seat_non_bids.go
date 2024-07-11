package exchange

import (
	"github.com/prebid/openrtb/v20/openrtb3"
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

	snb.seatNonBidsMap[seat] = append(snb.seatNonBidsMap[seat], nonBid)
}

// get returns a slice of SeatNonBid objects representing the non-bids for each seat.
// If snb is nil, it returns nil.
// It iterates over the seatNonBidsMap and appends each seat and its corresponding non-bids to the seatNonBid slice.
// Finally, it returns the seatNonBid slice.
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

// newProxyNonBid creates a new proxy non-bid object with the given impression ID and non-bid reason.
func newProxyNonBid(impId string, nonBidReason int) openrtb_ext.NonBid {
	return openrtb_ext.NonBid{
		ImpId:      impId,
		StatusCode: nonBidReason,
	}
}

// buildProxyNonBids creates a list of proxy non-bids with the given impression IDs and non-bid reason.
// this method is not thread safe as we are initializing and writing to map
func buildProxyNonBids(impIds []string, nonBidReason openrtb3.NoBidReason) []openrtb_ext.NonBid {
	if len(impIds) == 0 {
		return nil
	}
	proxyNonBids := []openrtb_ext.NonBid{}
	for _, impId := range impIds {
		nonBid := newProxyNonBid(impId, int(nonBidReason))
		proxyNonBids = append(proxyNonBids, nonBid)
	}
	return proxyNonBids
}

// addProxyNonBids adds the proxy non-bids to the seatNonBidsMap.
// It takes a list of impression IDs, a non-bid reason, and a seat as input parameters.
// It builds the proxy non-bids using the buildProxyNonBids function and appends them to the seatNonBidsMap.
// If the seatNonBidsMap is nil, it initializes it with an empty map.
// The proxy non-bids are added to the seatNonBidsMap under the specified seat.
// This method is not thread safe as we are initializing and writing to map
func (snb *nonBids) addProxyNonBids(impIds []string, nonBidReason openrtb3.NoBidReason, seat string) {
	if len(impIds) == 0 {
		return
	}
	proxyNonBids := buildProxyNonBids(impIds, nonBidReason)
	if snb.seatNonBidsMap == nil {
		snb.seatNonBidsMap = make(map[string][]openrtb_ext.NonBid)
	}
	snb.seatNonBidsMap[seat] = append(snb.seatNonBidsMap[seat], proxyNonBids...)
}

// append adds the nonBids from the input nonBids to the current nonBids.
func (snb *nonBids) append(nonBids ...nonBids) {
	if snb.seatNonBidsMap == nil {
		snb.seatNonBidsMap = make(map[string][]openrtb_ext.NonBid)
	}
	for _, nonBid := range nonBids {
		for seat, nonBids := range nonBid.seatNonBidsMap {
			snb.seatNonBidsMap[seat] = append(snb.seatNonBidsMap[seat], nonBids...)
		}
	}
}
