package openrtb2

import "encoding/json"

// 4.2.2 Object: SeatBid
//
// A bid response can contain multiple SeatBid objects, each on behalf of a different bidder seat and each containing one or more individual bids.
// If multiple impressions are presented in the request, the group attribute can be used to specify if a seat is willing to accept any impressions that it can win (default) or if it is only interested in winning any if it can win them all as a group.
type SeatBid struct {

	// Attribute:
	//   bid
	// Type:
	//   object array; required
	// Description:
	//   Array of 1+ Bid objects (Section 4.2.3) each related to an
	//   impression. Multiple bids can relate to the same impression.
	Bid []Bid `json:"bid"`

	// Attribute:
	//   seat
	// Type:
	//   string
	// Description:
	//   ID of the buyer seat (e.g., advertiser, agency) on whose behalf
	//   this bid is made.
	Seat string `json:"seat,omitempty"`

	// Attribute:
	//   group
	// Type:
	//   integer; default 0
	// Description:
	//   0 = impressions can be won individually; 1 = impressions must
	//   be won or lost as a group.
	Group int8 `json:"group,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for bidder-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
