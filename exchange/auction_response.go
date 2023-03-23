package exchange

import (
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// AuctionResponse contains OpenRTB Bid Response object and its extension (un-marshalled) object
type AuctionResponse struct {
	*openrtb2.BidResponse                             // BidResponse defines the contract for openrtb bidresponse
	ExtBidResponse        *openrtb_ext.ExtBidResponse // ExtBidResponse defines the contract for bidresponse.ext
}

// GetSeatNonBid returns array of seat non-bid if present. nil otherwise
func (ar *AuctionResponse) GetSeatNonBid() []openrtb_ext.SeatNonBid {
	if ar.ExtBidResponse != nil && ar.ExtBidResponse.Prebid != nil {
		return ar.ExtBidResponse.Prebid.SeatNonBid
	}
	return nil
}
