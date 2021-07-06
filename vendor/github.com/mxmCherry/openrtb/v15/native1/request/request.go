// Package request provides OpenRTB Native 1.2 request types
// (section "4 Native Ad Request Markup Details")
//
// https://iabtechlab.com/standards/openrtb-native/
// https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf
package request

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 4.1 Native Markup Request Object
//
// The Native Object defines the native1 advertising opportunity available for bid via this bid request.
// It will be included as a JSON-encoded string in the bid request’s imp.native1 field or as a direct JSON object, depending on the choice of the exchange.
// While OpenRTB 2.x officially supports only JSON-encoded strings, many exchanges have implemented a formal object.
// Check with your integration docs.
//
// Note: Prior to VERSION 1.1, the specification could be interpreted as requiring the native1 request to have a root node with a single field “native” that would contain the object above as its value.
// The Native Markup Request Object specified above is now the root object.
type Request struct {

	// Field:
	//   ver
	// Scope:
	//   optional
	// Type:
	//   string
	// Default:
	//   1.2
	// Description:
	//   Version of the Native Markup version in use.
	Ver string `json:"ver,omitempty"`

	// Field:
	//   layout
	// Scope:
	//   recommended in 1.0, deprecated/removed in 1.2
	// Type:
	//   integer
	// Description:
	//   The Layout ID of the native1 ad unit.
	//   See the Table of Layout IDs below.
	Layout native1.Layout `json:"layout,omitempty"`

	// Field:
	//   adunit
	// Scope:
	//   recommended in 1.0, deprecated/removed in 1.2
	// Type:
	//   integer
	// Description:
	//   The Ad unit ID of the native1 ad unit.
	//   See Table of Ad Unit IDs below for a list of supported core ad units.
	AdUnit native1.AdUnit `json:"adunit,omitempty"`

	// Field:
	//   context
	// Scope:
	//   recommended
	// Type:
	//   integer
	// Description:
	//   The context in which the ad appears.
	//   See Table of Context IDs below for a list of supported context types.
	Context native1.ContextType `json:"context,omitempty"`

	// Field:
	//   contextsubtype
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   A more detailed context in which the ad appears.
	//   See Table of Context SubType IDs below for a list of supported context subtypes.
	ContextSubType native1.ContextSubType `json:"contextsubtype,omitempty"`

	// Field:
	//   plcmttype
	// Scope:
	//   recommended
	// Type:
	//   integer
	// Description:
	//   The design/format/layout of the ad unit being offered.
	//   See Table of Placement Type IDs below for a list of supported placement types.
	PlcmtType native1.PlacementType `json:"plcmttype,omitempty"`

	// Field:
	//   plcmtcnt
	// Scope:
	//   optional
	// Type:
	//   integer
	// Default:
	//   1
	// Description:
	//   The number of identical placements in this Layout.
	//   Refer Section 8.1 Multiplacement Bid Requests for further detail.
	PlcmtCnt int64 `json:"plcmtcnt,omitempty"`

	// Field:
	//   seq
	// Scope:
	//   optional
	// Type:
	//   integer
	// Default:
	//   0
	// Description:
	//   0 for the first ad, 1 for the second ad, and so on.
	//   Note this would generally NOT be used in combination with plcmtcnt - either you are auctioning multiple identical placements (in which case plcmtcnt>1, seq=0) or you are holding separate auctions for distinct items in the feed (in which case plcmtcnt=1, seq=>=1)
	Seq int64 `json:"seq,omitempty"`

	// Field:
	//   assets
	// Scope:
	//   required
	// Type:
	//   array of objects
	// Description:
	//   An array of Asset Objects.
	//   Any bid response must comply with the array of elements expressed in the bid request.
	Assets []Asset `json:"assets"`

	// Field:
	//   aurlsupport
	// Scope:
	//   optional
	// Type:
	//   int
	// Default:
	//   0
	// Description:
	//   Whether the supply source / impression supports returning an assetsurl instead of an asset object.
	//   0 or the absence of the field indicates no such support.
	AURLSupport int8 `json:"aurlsupport,omitempty"`

	// Field:
	//   durlsupport
	// Scope:
	//   optional
	// Type:
	//   int
	// Default:
	//   0
	// Description:
	//   Whether the supply source / impression supports returning a dco url instead of an asset object.
	//   0 or the absence of the field indicates no such support.
	//   Beta feature.
	DURLSupport int8 `json:"durlsupport,omitempty"`

	// Field:
	//   eventtrackers
	// Scope:
	//   optional
	// Type:
	//   array of objects
	// Description:
	//   Specifies what type of event tracking is supported - see Event Trackers Request Object
	EventTrackers []EventTracker `json:"eventtrackers,omitempty"`

	// Field:
	//   privacy
	// Scope:
	//   recommended
	// Type:
	//   integer
	// Default:
	//   0
	// Description:
	//   Set to 1 when the native1 ad supports buyer-specific privacy notice.
	//   Set to 0 (or field absent) when the native1 ad doesn’t support custom privacy links or if support is unknown.
	Privacy int8 `json:"privacy,omitempty"`

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
