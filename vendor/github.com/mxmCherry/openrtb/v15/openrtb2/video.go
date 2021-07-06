package openrtb2

import "encoding/json"

// 3.2.7 Object: Video
//
// This object represents an in-stream video impression.
// Many of the fields are non-essential for minimally viable transactions, but are included to offer fine control when needed.
// Video in OpenRTB generally assumes compliance with the VAST standard.
// As such, the notion of companion ads is supported by optionally including an array of Banner objects (refer to the Banner object in Section 3.2.6) that define these companion ads.
//
// The presence of a Video as a subordinate of the Imp object indicates that this impression is offered as a video type impression.
// At the publisher’s discretion, that same impression may also be offered as banner, audio, and/or native by also including as Imp subordinates objects of those types.
// However, any given bid for the impression must conform to one of the offered types.
type Video struct {

	// Attribute:
	//   mimes
	// Type:
	//   string array; required
	// Description:
	//   Content MIME types supported (e.g., “video/x-ms-wmv”,
	//   “video/mp4”).
	MIMEs []string `json:"mimes"`

	// Attribute:
	//   minduration
	// Type:
	//   integer; recommended
	// Description:
	//   Minimum video ad duration in seconds.
	MinDuration int64 `json:"minduration,omitempty"`

	// Attribute:
	//   maxduration
	// Type:
	//   integer; recommended
	// Description:
	//   Maximum video ad duration in seconds.
	MaxDuration int64 `json:"maxduration,omitempty"`

	// Attribute:
	//   protocols
	// Type:
	//   integer array; recommended
	// Description:
	//   Array of supported video protocols. Refer to List 5.8. At least
	//   one supported protocol must be specified in either the
	//   protocol or protocols attribute.
	Protocols []Protocol `json:"protocols,omitempty"`

	// Attribute:
	//   protocol
	// Type:
	//   integer; DEPRECATED
	// Description:
	//   NOTE: Deprecated in favor of protocols.
	//   Supported video protocol. Refer to List 5.8. At least one
	//   supported protocol must be specified in either the protocol
	//   or protocols attribute.
	Protocol Protocol `json:"protocol,omitempty"`

	// Attribute:
	//   w
	// Type:
	//   integer; recommended
	// Description:
	//   Width of the video player in device independent pixels (DIPS).
	W int64 `json:"w,omitempty"`

	// Attribute:
	//   h
	// Type:
	//   integer; recommended
	// Description:
	//   Height of the video player in device independent pixels (DIPS).
	H int64 `json:"h,omitempty"`

	// Attribute:
	//   startdelay
	// Type:
	//   integer; recommended
	// Description:
	//   Indicates the start delay in seconds for pre-roll, mid-roll, or
	//   post-roll ad placements. Refer to List 5.12 for additional
	//   generic values.
	StartDelay *StartDelay `json:"startdelay,omitempty"`

	// Attribute:
	//   placement
	// Type:
	//   integer
	// Description:
	//   Placement type for the impression. Refer to List 5.9.
	Placement VideoPlacementType `json:"placement,omitempty"`

	// Attribute:
	//   linearity
	// Type:
	//   integer
	// Description:
	//   Indicates if the impression must be linear, nonlinear, etc. If
	//   none specified, assume all are allowed. Refer to List 5.7.
	Linearity VideoLinearity `json:"linearity,omitempty"`

	// Attribute:
	//   skip
	// Type:
	//   integer
	// Description:
	//   Indicates if the player will allow the video to be skipped,
	//   where 0 = no, 1 = yes.
	//   If a bidder sends markup/creative that is itself skippable, the
	//   Bid object should include the attr array with an element of
	//   16 indicating skippable video. Refer to List 5.3.
	Skip *int8 `json:"skip,omitempty"`

	// Attribute:
	//   skipmin
	// Type:
	//   integer; default 0
	// Description:
	//   Videos of total duration greater than this number of seconds
	//   can be skippable; only applicable if the ad is skippable.
	SkipMin int64 `json:"skipmin,omitempty"`

	// Attribute:
	//   skipafter
	// Type:
	//   integer; default 0
	// Description:
	//   Number of seconds a video must play before skipping is
	//   enabled; only applicable if the ad is skippable
	SkipAfter int64 `json:"skipafter,omitempty"`

	// Attribute:
	//   sequence
	// Type:
	//   integer
	// Description:
	//   If multiple ad impressions are offered in the same bid request,
	//   the sequence number will allow for the coordinated delivery
	//   of multiple creatives.
	Sequence int8 `json:"sequence,omitempty"`

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
	MinBitRate int64 `json:"minbitrate,omitempty"`

	// Attribute:
	//   maxbitrate
	// Type:
	//   integer
	// Description:
	//   Maximum bit rate in Kbps.
	MaxBitRate int64 `json:"maxbitrate,omitempty"`

	// Attribute:
	//   boxingallowed
	// Type:
	//   integer; default 1
	// Description:
	//   Indicates if letter-boxing of 4:3 content into a 16:9 window is
	//   allowed, where 0 = no, 1 = yes.
	BoxingAllowed int8 `json:"boxingallowed,omitempty"`

	// Attribute:
	//   playbackmethod
	// Type:
	//   integer array
	// Description:
	//   Playback methods that may be in use. If none are specified,
	//   any method may be used. Refer to List 5.10. Only one
	//   method is typically used in practice. As a result, this array may
	//   be converted to an integer in a future version of the
	//   specification. It is strongly advised to use only the first
	//   element of this array in preparation for this change.
	PlaybackMethod []PlaybackMethod `json:"playbackmethod,omitempty"`

	// Attribute:
	//   playbackend
	// Type:
	//   integer
	// Description:
	//   The event that causes playback to end. Refer to List 5.11.
	PlaybackEnd PlaybackCessationMode `json:"playbackend,omitempty"`

	// Attribute:
	//   delivery
	// Type:
	//   integer array
	// Description:
	//   Supported delivery methods (e.g., streaming, progressive). If
	//   none specified, assume all are supported. Refer to List 5.15.
	Delivery []ContentDeliveryMethod `json:"delivery,omitempty"`

	// Attribute:
	//   pos
	// Type:
	//   integer
	// Description:
	//   Ad position on screen. Refer to List 5.4.
	Pos *AdPosition `json:"pos,omitempty"`

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
	//   Supported VAST companion ad types. Refer to List 5.14.
	//   Recommended if companion Banner objects are included via
	//   the companionad array. If one of these banners will be
	//   rendered as an end-card, this can be specified using the vcm
	//   attribute with the particular banner (Section 3.2.6).
	CompanionType []CompanionType `json:"companiontype,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
