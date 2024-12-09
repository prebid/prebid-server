package exchange

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// AuctionResponse contains OpenRTB Bid Response object and its extension (un-marshalled) object
type AuctionResponse struct {
	*openrtb2.BidResponse
	ExtBidResponse *openrtb_ext.ExtBidResponse
}

// GetSeatNonBid returns array of seat non-bid if present. nil otherwise
func (ar *AuctionResponse) GetSeatNonBid() []openrtb_ext.SeatNonBid {
	if ar != nil && ar.ExtBidResponse != nil && ar.ExtBidResponse.Prebid != nil {
		return ar.ExtBidResponse.Prebid.SeatNonBid
	}
	return nil
}
