package openrtb_ext

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// SeatNonBid list the reasons why bid was not resulted in positive bid
// reason could be either No bid, Error, Request rejection or Response rejection
// Reference:  https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/seat-non-bid.md#list-non-bid-status-codes
type NonBidReason int64

const (
	ErrorGeneral                           NonBidReason = 100 // Error - General
	ErrorTimeout                           NonBidReason = 101 // Error - Timeout
	ErrorBidderUnreachable                 NonBidReason = 103 // Error - Bidder Unreachable
	ResponseRejectedGeneral                NonBidReason = 300
	ResponseRejectedBelowFloor             NonBidReason = 301 // Response Rejected - Below Floor
	ResponseRejectedCategoryMappingInvalid NonBidReason = 303 // Response Rejected - Category Mapping Invalid
	ResponseRejectedBelowDealFloor         NonBidReason = 304 // Response Rejected - Bid was Below Deal Floor
	ResponseRejectedCreativeSizeNotAllowed NonBidReason = 351 // Response Rejected - Invalid Creative (Size Not Allowed)
	ResponseRejectedCreativeNotSecure      NonBidReason = 352 // Response Rejected - Invalid Creative (Not Secure)
)

// NonBidCollection contains the map of seat with list of nonBids
type SeatNonBidBuilder map[string][]NonBid

// NonBidParams contains the fields that are required to form the nonBid object
type NonBidParams struct {
	Bid            *openrtb2.Bid
	NonBidReason   int
	OriginalBidCPM float64
	OriginalBidCur string
}

// NewNonBid creates the NonBid object from NonBidParams and return it
func NewNonBid(bidParams NonBidParams) NonBid {
	if bidParams.Bid == nil {
		bidParams.Bid = &openrtb2.Bid{}
	}
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
func (snb *SeatNonBidBuilder) AddBid(nonBid NonBid, seat string) {
	if *snb == nil {
		*snb = make(map[string][]NonBid)
	}
	(*snb)[seat] = append((*snb)[seat], nonBid)
}

// append adds the nonBids from the input nonBids to the current nonBids.
// This method is not thread safe as we are initializing and writing to map
func (snb *SeatNonBidBuilder) Append(nonBids ...SeatNonBidBuilder) {
	if *snb == nil {
		return
	}
	for _, nonBid := range nonBids {
		for seat, nonBids := range nonBid {
			(*snb)[seat] = append((*snb)[seat], nonBids...)
		}
	}
}

// Get function converts the internal seatNonBidsMap to standard openrtb seatNonBid structure and returns it
func (snb *SeatNonBidBuilder) Get() []SeatNonBid {
	if *snb == nil {
		return nil
	}
	var seatNonBid []SeatNonBid
	for seat, nonBids := range *snb {
		seatNonBid = append(seatNonBid, SeatNonBid{
			Seat:   seat,
			NonBid: nonBids,
		})
	}
	return seatNonBid
}

// rejectImps appends a non bid object to the builder for every specified imp
func (b *SeatNonBidBuilder) RejectImps(impIds []string, nonBidReason NonBidReason, seat string) {
	nonBids := []NonBid{}
	for _, impId := range impIds {
		nonBid := NonBid{
			ImpId:      impId,
			StatusCode: int(nonBidReason),
		}
		nonBids = append(nonBids, nonBid)
	}

	if len(nonBids) > 0 {
		(*b)[seat] = append((*b)[seat], nonBids...)
	}
}
