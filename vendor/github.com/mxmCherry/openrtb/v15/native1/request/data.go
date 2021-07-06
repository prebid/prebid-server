package request

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 4.6 Data Object
//
// The Data Object is to be used for all non-core elements of the native1 unit such as Brand Name, Ratings, Review Count, Stars, Download count, descriptions etc.
// It is also generic for future native1 elements not contemplated at the time of the writing of this document.
// In some cases, additional recommendations are also included in the Data Asset Types table.
type Data struct {

	// Field:
	//   type
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Type ID of the element supported by the publisher.
	//   The publisher can display this information in an appropriate format.
	//   See Data Asset Types table for commonly used examples.
	Type native1.DataAssetType `json:"type"`

	// Field:
	//   len
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   Maximum length of the text in the element’s response.
	Len int64 `json:"len,omitempty"`

	// Field:
	//   ext
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	// This object is a placeholder that may contain custom JSON agreed to by the parties to support flexibility beyond the standard defined in this specification
	Ext json.RawMessage `json:"ext,omitempty"`
}
