package openrtb2

import "encoding/json"

// 4.2.1 Object: BidResponse
//
// This object is the top-level bid response object (i.e., the unnamed outer JSON object).
// The id attribute is a reflection of the bid request ID for logging purposes.
// Similarly, bidid is an optional response tracking ID for bidders.
// If specified, it can be included in the subsequent win notice call if the bidder wins.
// At least one seatbid object is required, which contains at least one bid for an impression.
// Other attributes are optional.
//
// To express a “no-bid”, the options are to return an empty response with HTTP 204.
// Alternately if the bidder wishes to convey to the exchange a reason for not bidding, just a BidResponse object is returned with a reason code in the nbr attribute.
type BidResponse struct {

	// Attribute:
	//   id
	// Type:
	//   string; required
	// Description:
	//   ID of the bid request to which this is a response.
	ID string `json:"id"`

	// Attribute:
	//   seatbid
	// Type:
	//   object array
	// Description:
	//   Array of seatbid objects; 1+ required if a bid is to be made.
	SeatBid []SeatBid `json:"seatbid,omitempty"`

	// Attribute:
	//   bidid
	// Type:
	//   string
	// Description:
	//   Bidder generated response ID to assist with logging/tracking.
	BidID string `json:"bidid,omitempty"`

	// Attribute:
	//   cur
	// Type:
	//   string; default “USD”
	// Description:
	//   Bid currency using ISO-4217 alpha codes.
	Cur string `json:"cur,omitempty"`

	// Attribute:
	//   customdata
	// Type:
	//   string
	// Description:
	//   Optional feature to allow a bidder to set data in the
	//   exchange’s cookie. The string must be in base85 cookie safe
	//   characters and be in any format. Proper JSON encoding must
	//   be used to include “escaped” quotation marks.
	CustomData string `json:"customdata,omitempty"`

	// Attribute:
	//   nbr
	// Type:
	//   integer
	// Description:
	//   Reason for not bidding. Refer to List 5.24.
	NBR *NoBidReasonCode `json:"nbr,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for bidder-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
