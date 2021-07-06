package response

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 5.8 Event Tracker Response Object
//
// The event trackers response is an array of objects and specifies the types of events the bidder wishes to track and the URLs/information to track them.
// Bidder must only respond with methods indicated as available in the request.
// Note that most javascript trackers expect to be loaded at impression time, so it’s not generally recommended for the buyer to respond with javascript trackers on other events, but the appropriateness of this is up to each buyer.
type EventTracker struct {
	// Field:
	//   event
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Type of event to track.
	//   See Event Types table.
	Event native1.EventType `json:"event"`

	// Field:
	//   method
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Type of tracking requested
	//   See Event Tracking Methods table.
	Method native1.EventTrackingMethod `json:"method"`

	// Field:
	//   url
	// Scope:
	//   optional
	// Type:
	//   text
	// Description:
	//   The URL of the image or js.
	//   Required for image or js, optional for custom.
	URL string `json:"url,omitempty"`

	// Field:
	//   customdata
	// Scope:
	//   optional
	// Type:
	//   object containing key:value pairs
	// Description:
	//   To be agreed individually with the exchange, an array of key:value objects for custom tracking, for example the account number of the DSP with a tracking company.
	//   IE {“accountnumber”:”123”}
	// Dev note:
	//   - type is a key-value object (not specified if value is string-only)
	//   - descriptions says "an array of key:value objects" (???)
	CustomData json.RawMessage `json:"customdata,omitempty"`

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
