package request

import "encoding/json"

// 4.3 Title Object
//
// The Title object is to be used for title element of the Native ad.
type Title struct {
	// Field:
	//   len
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Maximum length of the text in the title element.
	//   Recommended to be 25, 90, or 140.
	Len int64 `json:"len"`

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
