package response

import "encoding/json"

// 5.7 Object: Link
//
// Used for ‘call to action’ assets, or other links from the Native ad.
// This Object should be associated to its peer object in the parent Asset Object or as the master link in the top level Native Ad response object.
// When that peer object is activated (clicked) the action should take the user to the location of the link.
type Link struct {
	// Field:
	//   url
	// Scope:
	//   required
	// Type:
	//   string
	// Description:
	//   Landing URL of the clickable link.
	URL string `json:"url"`

	// Field:
	//   clicktrackers
	// Scope:
	//   optional
	// Type:
	//   string array
	// Description:
	//   List of third-party tracker URLs to be fired on click of the URL.
	ClickTrackers []string `json:"clicktrackers,omitempty"`

	// Field:
	//   fallback
	// Scope:
	//   optional
	// Type:
	//   string
	// Description:
	//   Fallback URL for deeplink.
	//   To be used if the URL given in url is not supported by the device.
	Fallback string `json:"fallback,omitempty"`

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
