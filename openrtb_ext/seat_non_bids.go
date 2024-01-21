package openrtb_ext

import "github.com/prebid/openrtb/v19/openrtb2"

// NonBidCollection contains the map of seat with list of nonBids
type NonBidCollection struct {
	seatNonBidsMap map[string][]NonBid
}

// NonBidParams contains the fields that are required to form the nonBid object
type NonBidParams struct {
	Bid            *openrtb2.Bid
	NonBidReason   int
	Seat           string
	OriginalBidCPM float64
	OriginalBidCur string
}

// NewNonBid creates the NonBid object from NonBidParams object and returns it
func NewNonBid(bidParams NonBidParams) NonBid {
	return NonBid{
		ImpId:      bidParams.Bid.ImpID,
		StatusCode: bidParams.NonBidReason,
		Ext: ExtNonBid{
			Prebid: ExtNonBidPrebid{Bid: ExtNonBidPrebidBid{
				Price:          bidParams.Bid.Price,
				ADomain:        bidParams.Bid.ADomain,
				CatTax:         bidParams.Bid.CatTax,
				Cat:            bidParams.Bid.Cat,
				DealID:         bidParams.Bid.DealID,
				W:              bidParams.Bid.W,
				H:              bidParams.Bid.H,
				Dur:            bidParams.Bid.Dur,
				MType:          bidParams.Bid.MType,
				OriginalBidCPM: bidParams.OriginalBidCPM,
				OriginalBidCur: bidParams.OriginalBidCur,
			}},
		},
	}
}

// AddBid adds the nonBid into the map against the respective seat.
// Note: This function is not a thread safe.
func (snb *NonBidCollection) AddBid(bidParams NonBidParams) {
	if bidParams.Bid == nil {
		return
	}
	if snb.seatNonBidsMap == nil {
		snb.seatNonBidsMap = make(map[string][]NonBid)
	}
	nonBid := NonBid{
		ImpId:      bidParams.Bid.ImpID,
		StatusCode: bidParams.NonBidReason,
		Ext: ExtNonBid{
			Prebid: ExtNonBidPrebid{Bid: ExtNonBidPrebidBid{
				Price:          bidParams.Bid.Price,
				ADomain:        bidParams.Bid.ADomain,
				CatTax:         bidParams.Bid.CatTax,
				Cat:            bidParams.Bid.Cat,
				DealID:         bidParams.Bid.DealID,
				W:              bidParams.Bid.W,
				H:              bidParams.Bid.H,
				Dur:            bidParams.Bid.Dur,
				MType:          bidParams.Bid.MType,
				OriginalBidCPM: bidParams.OriginalBidCPM,
				OriginalBidCur: bidParams.OriginalBidCur,
			}},
		},
	}

	snb.seatNonBidsMap[bidParams.Seat] = append(snb.seatNonBidsMap[bidParams.Seat], nonBid)
}

// Append functions appends the NonBids from the input instance into the current instance's seatNonBidsMap, creating the map if needed.
// Note: This function is not a thread safe.
func (snb *NonBidCollection) Append(nonbid NonBidCollection) {
	if snb == nil || len(nonbid.seatNonBidsMap) == 0 {
		return
	}
	if snb.seatNonBidsMap == nil {
		snb.seatNonBidsMap = make(map[string][]NonBid, len(nonbid.seatNonBidsMap))
	}
	for seat, nonBids := range nonbid.seatNonBidsMap {
		snb.seatNonBidsMap[seat] = append(snb.seatNonBidsMap[seat], nonBids...)
	}
}

// Get function converts the internal seatNonBidsMap to standard openrtb seatNonBid structure and returns it
func (snb *NonBidCollection) Get() []SeatNonBid {
	if snb == nil {
		return nil
	}
	var seatNonBid []SeatNonBid
	for seat, nonBids := range snb.seatNonBidsMap {
		seatNonBid = append(seatNonBid, SeatNonBid{
			Seat:   seat,
			NonBid: nonBids,
		})
	}
	return seatNonBid
}
