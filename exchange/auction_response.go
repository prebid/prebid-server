package exchange

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// AuctionResponse contains OpenRTB Bid Response object and its extension (un-marshalled) object
type AuctionResponse struct {
	*openrtb2.BidResponse
	ExtBidResponse *openrtb_ext.ExtBidResponse
	SeatNonBid     openrtb_ext.SeatNonBidBuilder
}
