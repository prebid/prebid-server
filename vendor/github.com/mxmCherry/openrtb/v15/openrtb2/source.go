package openrtb2

import "encoding/json"

// 3.2.2 Object: Source
//
// This object describes the nature and behavior of the entity that is the source of the bid request upstream from the exchange.
// The primary purpose of this object is to define post-auction or upstream decisioning when the exchange itself does not control the final decision.
// A common example of this is header bidding, but it can also apply to upstream server entities such as another RTB exchange, a mediation platform, or an ad server combines direct campaigns with 3rd party demand in decisioning.
type Source struct {

	// Attribute:
	//   fd
	// Type:
	//   Integer; recommended
	// Description:
	//   Entity responsible for the final impression sale decision, where
	//   0 = exchange, 1 = upstream source.
	FD int8 `json:"fd,omitempty"`

	// Attribute:
	//   tid
	// Type:
	//   string; recommended
	// Description:
	//   Transaction ID that must be common across all participants in
	//   this bid request (e.g., potentially multiple exchanges).
	TID string `json:"tid,omitempty"`

	// Attribute:
	//   pchain
	// Type:
	//   string; recommended
	// Description:
	//   Payment ID chain string containing embedded syntax
	//   described in the TAG Payment ID Protocol v1.0.
	PChain string `json:"pchain,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
