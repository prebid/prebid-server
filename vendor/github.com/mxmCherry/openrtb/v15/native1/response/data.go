package response

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 5.5 Object: Data
//
// Corresponds to the Data Object in the request, with the value filled in.
// The Data Object is to be used for all miscellaneous elements of the native1 unit such as Brand Name, Ratings, Review Count, Stars, Downloads, Price count etc.
// It is also generic for future native1 elements not contemplated at the time of the writing of this document.
type Data struct {
	// Field:
	//   type
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   Required for assetsurl/dcourl responses, not required for embedded asset responses.
	//   The type of data element being submitted from the Data Asset Types table.
	Type native1.DataAssetType `json:"type,omitempty"`

	// Field:
	//   len
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   Required for assetsurl/dcourl responses, not required for embedded asset responses.
	//   The length of the data element being submitted.
	//   Where applicable, must comply with the recommended maximum lengths in the Data Asset Types table.
	Len int64 `json:"len,omitempty"`

	// Field:
	//   label
	// Scope:
	//   optional in 1.1, deprecated/removed in 1.2
	// Type:
	//   string
	// Description:
	//   The optional formatted string name of the data type to be displayed.
	Label string `json:"label,omitempty"`

	// Field:
	//   value
	// Scope:
	//   required
	// Type:
	//   string
	// Description:
	//   The formatted string of data to be displayed.
	//   Can contain a formatted value such as "5 stars" or "$10" or "3.4 stars out of 5".
	Value string `json:"value"`

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
