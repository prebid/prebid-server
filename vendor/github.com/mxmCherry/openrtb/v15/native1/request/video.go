package request

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 4.5 Video Object
//
// The video object to be used for all video elements supported in the Native Ad.
// This corresponds to the Video object of OpenRTB.
// Exchange implementers can impose their own specific restrictions.
// Here are the required attributes of the Video Object.
// For optional attributes please refer to OpenRTB.
type Video struct {
	// Field:
	//   mimes
	// Scope:
	//   required
	// Type:
	//   array of string
	// Description:
	//   Content MIME types supported.
	//   Popular MIME types include,but are not limited to “video/x-mswmv” for Windows Media, and “video/x-flv” for Flash Video, or “video/mp4”.
	//   Note that native1 frequently does not support flash.
	MIMEs []string `json:"mimes"`

	// Field:
	//   minduration
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Minimum video ad duration in seconds.
	MinDuration int64 `json:"minduration"`

	// Field:
	//   maxduration
	// Scope:
	//   required
	// Type:
	//   integer
	// Description:
	//   Maximum video ad duration in seconds.
	MaxDuration int64 `json:"maxduration"`

	// Field:
	//   protocols
	// Scope:
	//   required
	// Type:
	//   array of integers
	// Description:
	//   An array of video protocols the publisher can accept in the bid response.
	//   See OpenRTB Table ‘Video Bid Response Protocols’ for a list of possible values.
	Protocols []native1.Protocol `json:"protocols"`

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
