package openrtb2

import "encoding/json"

// 3.2.11 Object: Pmp
//
// This object is the private marketplace container for direct deals between buyers and sellers that may pertain to this impression.
// The actual deals are represented as a collection of Deal objects.
// Refer to Section 7.3 for more details.
type PMP struct {

	// Attribute:
	//   private_auction
	// Type:
	//   integer; default 0
	// Description:
	//   Indicator of auction eligibility to seats named in the Direct
	//   Deals object, where 0 = all bids are accepted, 1 = bids are
	//   restricted to the deals specified and the terms thereof.
	PrivateAuction int8 `json:"private_auction,omitempty"`

	// Attribute:
	//   deals
	// Type:
	//   object array
	// Description:
	//   Array of Deal (Section 3.2.12) objects that convey the specific
	//   deals applicable to this impression.
	Deals []Deal `json:"deals,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
