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
		// discards non selected bids, do we ever need them???
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

/*
 JAVA-PBS

 		bidderToMultiBids
				pubmatic -> maxBids: 2
							prefixBiddercode: pm
				appnexus -> maxBids: 2

	create(auctionParticipations) - Creates an OpenRTB {@link BidResponse} from the bids supplied by the bidder, including processing of winning bids with cache IDs.
		[]auctionParticipations.bidderResponses
		...
		bidderResponses
			[
				SeatBid{pubmatic, [bid1, bid2, bid3, bid7, bid8, bid9]}
				SeatBid{appnexus, [bid4, bid5, bid6, bid10, bid11, bid12]}
			]
		cacheBidsAndCreateResponse(bidderResponses)
			toBidderResponseWithTargetingBidInfos(bidderResponses)
				bidderResponseToReducedBidInfos - uses toSortedMultiBidInfo()(returns list with sort and maxBid applied)
						{pubmatic, [bid1, bid2, bid3, bid7, bid8, bid9]}		->	[bid1, bid2, bid7, bid8]
						{appnexus, [bid4, bid5, bid6, bid10, bid11, bid12]}		->	[bid4, bid5, bid10, bid11]
				impIdToBidderToBidInfos -> group by impId, bidder
						imp1	-> pubmatic -> [bid1, bid2]   (bid3 removed by maxBid)
								-> appnexus -> [bid4, bid5]   (bid6 removed by maxBid)
						imp2    -> pubmatic -> [bid7, bid8]   (bid9 removed by maxBid)
								-> appnexus -> [bid10, bid11] (bid12 removed by maxBid)
				forEach impIdToBidderToBidInfo
							0.
								bidderToBidInfos
									pubmatic -> [bid1, bid2]
									appnexus -> [bid4, bid5]

								winningBidsByBidder -> [bid1, bid2, bid4, bid5]
								winningBids -> [bid1] -> (bid1 is max in [bid1, bid2, bid4, bid5])

							1.
								bidderToBidInfos
									pubmatic -> [bid7, bid8]
									appnexus -> [bid10, bid11]

								winningBidsByBidder -> [bid1, bid2, bid4, bid5, bid7, bid8, bid10, bid11]
								winningBids -> [bid1, bid10] -> (bid10 is max in [bid7, bid8, bid10, bid11])
				forEach bidderResponseToReducedBidInfos
					0.
						bidderResponseInfo - bidderResponseToReducedBidInfos[0].Key - {pubmatic, [bid1, bid2, bid3, bid7, bid8, bid9]}
						bidderBidInfos - bidderResponseToReducedBidInfos[0].Value - [bid1, bid2, bid7, bid8]

						injectBidInfoWithTargeting()
							bidder = pubmatic
							toBidInfoWithTargeting()
								bidderBidInfos -> impIdToBidInfos ->	imp1	-> 	bid1, bid2
																		imp2	->	bid7, bid8
								// forEach impIdToBidInfos.values() -> [bid1, bid2] -> CHECK THIS WITH AMOL
									0.
										injectTargeting()
											bidderImpIdBidInfos: [bid1, bid2]
											forEach bidderImpIdBidInfos
												0.
													bidderCode = pubmatic
												1.
													bidderCode = pm2
									1.
										injectTargeting()
											bidderImpIdBidInfos: [bid7, bid8]
											forEach bidderImpIdBidInfos
												0.
													bidderCode = pubmatic
												1.
													bidderCode = pm2

								bidInfosWithTargeting = [bid1, bid2, bid7, bid8]

					1.
						bidderResponseInfo - bidderResponseToReducedBidInfos[1].Key - {appnexus, [bid4, bid5, bid6, bid10, bid11, bid12]}
						bidderBidInfos - bidderResponseToReducedBidInfos[1].Value - [bid4, bid5, bid10, bid11]

						injectBidInfoWithTargeting()
							bidder = appnexus
							toBidInfoWithTargeting()
								impIdToBidInfos ->	imp1	-> 	bid4, bid5
													imp2	->	bid10, bid11
								// forEach impIdToBidInfos.values()
									injectTargeting(impIdToBidInfos[i])
									0.
										injectTargeting()
											bidderImpIdBidInfos: [bid4, bid5]
											forEach bidderImpIdBidInfos
												0.
													bidderCode = appnexus
												1.
													bidderCode = appnexus // no targeting
									1.
										injectTargeting()
											bidderImpIdBidInfos: [bid10, bid11]
											forEach bidderImpIdBidInfos
												0.
													bidderCode = appnexus
												1.
													bidderCode = appnexus // no targeting

								bidInfosWithTargeting =[bid4, bid5, bid10, bid11]
		bidderResponseInfos
			[
				SeatBid{pubmatic, [bid1, bid2, bid7, bid8]}
				SeatBid{appnexus, [bid4, bid5, bid10, bid11]}
			]
*/
