package request

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 4.7 Event Trackers Request Object
//
// The event trackers object specifies the types of events the bidder can request to be tracked in the bid response, and which types of tracking are available for each event type, and is included as an array in the request.
type EventTracker struct {

	// Field:
	//   event
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Type of event available for tracking.
	//   See Event Types table.
	Event native1.EventType `json:"event"`

	// Field:
	//   methods
	// Scope:
	//   required
	// Type:
	//   array of integers
	// Description:
	//   Array of the types of tracking available for the given event.
	//   See Event Tracking Methods table
	Methods []native1.EventTrackingMethod `json:"methods"`

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
