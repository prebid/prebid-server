package exchange

import "github.com/mxmCherry/openrtb"

// ExtSeatBid defines the contract for bidresponse.seatbid.ext
type ExtSeatBid struct {
	Bidder openrtb.RawJSON `json:"bidder,omitempty"`
}
