package response

import "encoding/json"

// 5.3 Object: Title
//
// Corresponds to the Title Object in the request, with the value filled in.
//
// If using assetsurl or dcourl response rather than embedded asset response, it is recommended that three title objects be provided, the length of each of which is less than or equal to the three recommended maximum title lengths (25,90,140).
type Title struct {
	// Field:
	//   text
	// Scope:
	//   required
	// Type:
	//   string
	// Description:
	//   The text associated with the text element.
	Text string `json:"text"`

	// Field:
	//   len
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   The length of the title being provided.
	//   Required if using assetsurl/dcourl representation, optional if using embedded asset representation.
	Len int64 `json:"len,omitempty"`

	// Field:
	//   ext
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   This object is a placeholder that may contain custom JSON agreed to by the parties to support flexibility beyond the standard defined in this specification
	Ext json.RawMessage `json:"ext,omitempty"`
}
