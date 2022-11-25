package exchange

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

const DefaultBidLimit = 1
const MaxBidLimit = 9

type ExtMultiBidMap map[string]*openrtb_ext.ExtMultiBid

// Validate and add multiBid value
func (mb *ExtMultiBidMap) Add(multiBid *openrtb_ext.ExtMultiBid) {
	// If maxbids is not specified, ignore whole block and add warning when in debug mode
	if multiBid.MaxBids == nil {
		return
	}

	// Min and default is 1
	if *multiBid.MaxBids < DefaultBidLimit {
		*multiBid.MaxBids = DefaultBidLimit
	}

	// Max 9
	if *multiBid.MaxBids > MaxBidLimit {
		*multiBid.MaxBids = MaxBidLimit
	}

	// Prefer Bidder over []Bidders
	if multiBid.Bidder != "" {
		if _, ok := (*mb)[multiBid.Bidder]; ok {
			// specified multiple times, use the first instance, ignore all the following mentions.
			// TODO add warning when in debug mode
			// ignore whole block if maxbid not specified. TODO add debug warning
			return
		}

		multiBid.Bidders = nil // ignore 'bidders' and add warning when in debug mode
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
		return *maxBid.MaxBids
	}
	return DefaultBidLimit
}

// groupby bids by impId
func getBidsByImpId(pbsBids []*pbsOrtbBid) (impIdToBidMap map[string][]*pbsOrtbBid) {
	impIdToBidMap = make(map[string][]*pbsOrtbBid)
	for _, pbsBid := range pbsBids {
		impIdToBidMap[pbsBid.bid.ImpID] = append(impIdToBidMap[pbsBid.bid.ImpID], pbsBid)
	}
	return
}
