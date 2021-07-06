package openrtb2

import "encoding/json"

// 3.2.8 Object: Audio
//
// This object represents an audio type impression.
// Many of the fields are non-essential for minimally viable transactions, but are included to offer fine control when needed.
// Audio in OpenRTB generally assumes compliance with the DAAST standard.
// As such, the notion of companion ads is supported by optionally including an array of Banner objects (refer to the Banner object in Section 3.2.6) that define these companion ads.
//
// The presence of a Audio as a subordinate of the Imp object indicates that this impression is offered as an audio type impression.
// At the publisher’s discretion, that same impression may also be offered as banner, video, and/or native by also including as Imp subordinates objects of those types.
// However, any given bid for the impression must conform to one of the offered types.
type Audio struct {

	// Attribute:
	//   mimes
	// Type:
	// string array; required
	// Description:
	//   Content MIME types supported (e.g., “audio/mp4”).
	MIMEs []string `json:"mimes"`

	// Attribute:
	//   minduration
	// Type:
	//   integer; recommended
	// Description:
	//   Minimum audio ad duration in seconds.
	MinDuration int64 `json:"minduration,omitempty"`

	// Attribute:
	//   maxduration
	// Type:
	//   integer; recommended
	// Description:
	//   Maximum audio ad duration in seconds.
	MaxDuration int64 `json:"maxduration,omitempty"`

	// Attribute:
	//   protocols
	// Type:
	//   integer array; recommended
	// Description:
	//   Array of supported audio protocols. Refer to List 5.8.
	Protocols []Protocol `json:"protocols,omitempty"`

	// Attribute:
	//   startdelay
	// Type:
	//   integer; recommended
	// Description:
	//   Indicates the start delay in seconds for pre-roll, mid-roll, or
	//   post-roll ad placements. Refer to List 5.12.
	StartDelay *StartDelay `json:"startdelay,omitempty"`

	// Attribute:
	//   sequence
	// Type:
	//   integer
	// Description:
	//   If multiple ad impressions are offered in the same bid request,
	//   the sequence number will allow for the coordinated delivery
	//   of multiple creatives.
	Sequence int64 `json:"sequence,omitempty"`

	// Attribute:
	//   battr
	// Type:
	//   integer array
	// Description:
	//   Blocked creative attributes. Refer to List 5.3.
	BAttr []CreativeAttribute `json:"battr,omitempty"`

	// Attribute:
	//   maxextended
	// Type:
	//   integer
	// Description:
	//   Maximum extended ad duration if extension is allowed. If
	//   blank or 0, extension is not allowed. If -1, extension is
	//   allowed, and there is no time limit imposed. If greater than 0,
	//   then the value represents the number of seconds of extended
	//   play supported beyond the maxduration value.
	MaxExtended int64 `json:"maxextended,omitempty"`

	// Attribute:
	//   minbitrate
	// Type:
	//   integer
	// Description:
	//   Minimum bit rate in Kbps.
	MinBitrate int64 `json:"minbitrate,omitempty"`

	// Attribute:
	//   maxbitrate
	// Type:
	//   integer
	// Description:
	//   Maximum bit rate in Kbps.
	MaxBitrate int64 `json:"maxbitrate,omitempty"`

	// Attribute:
	//   delivery
	// Type:
	//   integer array
	// Description:
	//   Supported delivery methods (e.g., streaming, progressive). If
	//   none specified, assume all are supported. Refer to List 5.15.
	Delivery []ContentDeliveryMethod `json:"delivery,omitempty"`

	// Attribute:
	//   companionad
	// Type:
	//   object array
	// Description:
	//   Array of Banner objects (Section 3.2.6) if companion ads are
	//   available.
	CompanionAd []Banner `json:"companionad,omitempty"`

	// Attribute:
	//   api
	// Type:
	//   integer array
	// Description:
	//   List of supported API frameworks for this impression. Refer to
	//   List 5.6. If an API is not explicitly listed, it is assumed not to be
	//   supported.
	API []APIFramework `json:"api,omitempty"`

	// Attribute:
	//   companiontype
	// Type:
	//   integer array
	// Description:
	//   Supported DAAST companion ad types. Refer to List 5.14.
	//   Recommended if companion Banner objects are included via
	//   the companionad array.
	CompanionType []CompanionType `json:"companiontype,omitempty"`

	// Attribute:
	//   maxseq
	// Type:
	//   integer
	// Description:
	//   The maximum number of ads that can be played in an ad pod.
	//   OpenRTB API Specification Version 2.5 IAB Technology Lab
	//   www.iab.com/openrtb Page 18
	MaxSeq int64 `json:"maxseq,omitempty"`

	// Attribute:
	//   feed
	// Type:
	//   integer
	// Description:
	//   Type of audio feed. Refer to List 5.16.
	Feed FeedType `json:"feed,omitempty"`

	// Attribute:
	//   stitched
	// Type:
	//   integer
	// Description:
	//   Indicates if the ad is stitched with audio content or delivered
	//   independently, where 0 = no, 1 = yes.
	Stitched int8 `json:"stitched,omitempty"`

	// Attribute:
	//   nvol
	// Type:
	//   integer
	// Description:
	//   Volume normalization mode. Refer to List 5.17.
	NVol *VolumeNormalizationMode `json:"nvol,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
