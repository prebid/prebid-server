package openrtb2

import "encoding/json"

// 3.2.22 Object: Segment
//
// Segment objects are essentially key-value pairs that convey specific units of data.
// The parent Data object is a collection of such values from a given data provider.
// The specific segment names and value options must be published by the exchange a priori to its bidders.
type Segment struct {

	// Attribute:
	//   id
	// Type:
	//   string
	// Description:
	//   ID of the data segment specific to the data provider.
	ID string `json:"id,omitempty"`

	// Attribute:
	//   name
	// Type:
	//   string
	// Description:
	//   Name of the data segment specific to the data provider.
	Name string `json:"name,omitempty"`

	// Attribute:
	//   value
	// Type:
	//   string
	// Description:
	//   String representation of the data segment value.
	Value string `json:"value,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
