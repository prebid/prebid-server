package openrtb2

import "encoding/json"

// 3.2.6 Object: Banner
//
// This object represents the most general type of impression.
// Although the term “banner” may have very specific meaning in other contexts, here it can be many things including a simple static image, an expandable ad unit, or even in-banner video (refer to the Video object in Section 3.2.7 for the more generalized and full featured video ad units).
// An array of Banner objects can also appear within the Video to describe optional companion ads defined in the VAST specification.
//
// The presence of a Banner as a subordinate of the Imp object indicates that this impression is offered as a banner type impression.
// At the publisher’s discretion, that same impression may also be offered as video, audio, and/or native by also including as Imp subordinates objects of those types.
// However, any given bid for the impression must conform to one of the offered types.
type Banner struct {

	// Attribute:
	//   format
	// Type:
	//   object array; recommended
	// Description:
	//   Array of format objects (Section 3.2.10) representing the
	//   banner sizes permitted. If none are specified, then use of the
	//   h and w attributes is highly recommended.
	Format []Format `json:"format,omitempty"`

	// Attribute:
	//   w
	// Type:
	//   integer; recommended
	// Description:
	//   Exact width in device independent pixels (DIPS);
	//   recommended if no format objects are specified.
	W *int64 `json:"w,omitempty"`

	// Attribute:
	//   h
	// Type:
	//   integer; recommended
	// Description:
	//   Exact height in device independent pixels (DIPS);
	//   recommended if no format objects are specified.
	H *int64 `json:"h,omitempty"`

	// Attribute:
	//   wmax
	// Type:
	//   integer; DEPRECATED
	// Description:
	//   NOTE: Deprecated in favor of the format array.
	//   Maximum width in device independent pixels (DIPS).
	WMax int64 `json:"wmax,omitempty"`

	// Attribute:
	//   hmax
	// Type:
	//   integer; DEPRECATED
	// Description:
	//   NOTE: Deprecated in favor of the format array.
	//   Maximum height in device independent pixels (DIPS).
	HMax int64 `json:"hmax,omitempty"`

	// Attribute:
	//   wmin
	// Type:
	//   integer; DEPRECATED
	// Description:
	//   NOTE: Deprecated in favor of the format array.
	//   Minimum width in device independent pixels (DIPS).
	WMin int64 `json:"wmin,omitempty"`

	// Attribute:
	//   hmin
	// Type:
	//   integer; DEPRECATED
	// Description:
	//   NOTE: Deprecated in favor of the format array.
	//   Minimum height in device independent pixels (DIPS).
	HMin int64 `json:"hmin,omitempty"`

	// Attribute:
	//   btype
	// Type:
	//   integer array
	// Description:
	//   Blocked banner ad types. Refer to List 5.2.
	BType []BannerAdType `json:"btype,omitempty"`

	// Attribute:
	//   battr
	// Type:
	//   integer array
	// Description:
	//   Blocked creative attributes. Refer to List 5.3.
	BAttr []CreativeAttribute `json:"battr,omitempty"`

	// Attribute:
	//   pos
	// Type:
	//   integer
	// Description:
	//   Ad position on screen. Refer to List 5.4.
	Pos *AdPosition `json:"pos,omitempty"`

	// Attribute:
	//   mimes
	// Type:
	//   string array
	// Description:
	//   Content MIME types supported. Popular MIME types may
	//   include “application/x-shockwave-flash”,
	//   “image/jpg”, and “image/gif”.
	MIMEs []string `json:"mimes,omitempty"`

	// Attribute:
	//   topframe
	// Type:
	//   integer
	// Description:
	//   Indicates if the banner is in the top frame as opposed to an
	//   iframe, where 0 = no, 1 = yes.
	TopFrame int8 `json:"topframe,omitempty"`

	// Attribute:
	//   expdir
	// Type:
	//   integer array
	// Description:
	//   Directions in which the banner may expand. Refer to List 5.5.
	ExpDir []ExpandableDirection `json:"expdir,omitempty"`

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
	//   id
	// Type:
	//   string
	// Description:
	//   Unique identifier for this banner object. Recommended when
	//   Banner objects are used with a Video object (Section 3.2.7) to
	//   represent an array of companion ads. Values usually start at 1
	//   and increase with each object; should be unique within an
	//   impression.
	ID string `json:"id,omitempty"`

	// Attribute:
	//   vcm
	// Type:
	//   integer
	// Description:
	//   Relevant only for Banner objects used with a Video object
	//   (Section 3.2.7) in an array of companion ads. Indicates the
	//   companion banner rendering mode relative to the associated
	//   video, where 0 = concurrent, 1 = end-card.
	VCm int8 `json:"vcm,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
