package exchange

import (
	"sort"

	"github.com/prebid/prebid-server/openrtb_ext"
)

const DefaultBidLimit = 1

type ExtMultiBidMap map[string]*openrtb_ext.ExtMultiBid

// Validate and add multiBid value
func (mb *ExtMultiBidMap) Add(multiBid *openrtb_ext.ExtMultiBid) {
	// Min and default is 1
	if multiBid.MaxBids < 1 {
		multiBid.MaxBids = 1
	}

	// Max 9
	if multiBid.MaxBids > 9 {
		multiBid.MaxBids = 9
	}

	// Prefer Bidder over []Bidders
	if multiBid.Bidder != "" {
		if _, ok := (*mb)[multiBid.Bidder]; ok || multiBid.MaxBids == 0 {
			//specified multiple times, use the first bidder. TODO add warning when in debug mode
			//ignore whole block if maxbid not specified. TODO add debug warning
			return
		}

		multiBid.Bidders = nil //ignore 'bidders' and add warning when in debug mode
		(*mb)[multiBid.Bidder] = multiBid
	} else if len(multiBid.Bidders) > 0 {
		for _, bidder := range multiBid.Bidders {
			if _, ok := (*mb)[multiBid.Bidder]; ok {
				return
			}
			multiBid.TargetBidderCodePrefix = "" //ignore targetbiddercodeprefix and add warning when in debug mode
			(*mb)[bidder] = multiBid
		}
	}
}

// Get multi-bid limit for this bidder
func (mb *ExtMultiBidMap) GetMaxBids(bidder string) int {
	if maxBid, ok := (*mb)[bidder]; ok {
		return maxBid.MaxBids
	}
	return DefaultBidLimit
}

func sortLimitMultiBid(seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid,
	multiBid ExtMultiBidMap, preferDeals bool,
) map[openrtb_ext.BidderName]*pbsOrtbSeatBid {
	for bidderName, seatBid := range seatBids {
		if seatBid == nil {
			continue
		}

		impIdToBidMap := getBidsByImpId(seatBid.bids)
		bidLimit := multiBid.GetMaxBids(bidderName.String())

		var finalBids []*pbsOrtbBid
		for _, bids := range impIdToBidMap {
			// sort bids for this impId (same logic as auction)
			sort.Slice(bids, func(i, j int) bool {
				return isNewWinningBid(bids[i].bid, bids[j].bid, preferDeals)
			})

			// assert maxBids for this impId
			if len(bids) > bidLimit {
				bids = bids[:bidLimit]
			}

			finalBids = append(finalBids, bids...)
		}

		// Update the final bid list of this bidder
		// i.e. All the bids sorted and capped by impId
		seatBid.bids = finalBids

	}
	return seatBids
}

// groupby bids by impId
func getBidsByImpId(pbsBids []*pbsOrtbBid) (impIdToBidMap map[string][]*pbsOrtbBid) {
	impIdToBidMap = make(map[string][]*pbsOrtbBid)
	for _, pbsBid := range pbsBids {
		impIdToBidMap[pbsBid.bid.ImpID] = append(impIdToBidMap[pbsBid.bid.ImpID], pbsBid)
	}
	return
}
