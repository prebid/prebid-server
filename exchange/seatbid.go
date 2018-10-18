package exchange

import "encoding/json"

// ExtSeatBid defines the contract for bidresponse.seatbid.ext
type ExtSeatBid struct {
	Bidder json.RawMessage `json:"bidder,omitempty"`
}
