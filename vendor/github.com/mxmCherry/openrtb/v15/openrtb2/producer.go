package openrtb2

import "encoding/json"

// 3.2.17 Object: Producer
//
// This object defines the producer of the content in which the ad will be shown.
// This is particularly useful when the content is syndicated and may be distributed through different publishers and thus when the producer and publisher are not necessarily the same entity.
type Producer struct {

	// Attribute:
	//   id
	// Type:
	//   string
	// Description:
	//   Content producer or originator ID. Useful if content is
	//   syndicated and may be posted on a site using embed tags.
	ID string `json:"id,omitempty"`

	// Attribute:
	//   name
	// Type:
	//   string
	// Description:
	//   Content producer or originator name (e.g., “Warner Bros”).
	Name string `json:"name,omitempty"`

	// Attribute:
	//   cat
	// Type:
	//   string array
	// Description:
	//   Array of IAB content categories that describe the content
	//   producer. Refer to List 5.1.
	Cat []string `json:"cat,omitempty"`

	// Attribute:
	//   domain
	// Type:
	//   string
	// Description:
	//   Highest level domain of the content producer (e.g.,
	//   “producer.com”).
	Domain string `json:"domain,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
