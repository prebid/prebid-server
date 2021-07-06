package request

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 4.4 Image Object
//
// The Image object to be used for all image elements of the Native ad such as Icons, Main Image, etc.
// Recommended sizes and aspect ratios are included in the Image Asset Types section.
type Image struct {

	// Field:
	//   type
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   Type ID of the image element supported by the publisher.
	//   The publisher can display this information in an appropriate format.
	//   See Table Image Asset Types.
	Type native1.ImageAssetType `json:"type,omitempty"`

	// Field:
	//   w
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   Width of the image in pixels.
	W int64 `json:"w,omitempty"`

	// Field:
	//   wmin
	// Scope:
	//   recommended
	// Type:
	//   integer
	// Description:
	//   The minimum requested width of the image in pixels.
	//   This option should be used for any rescaling of images by the client.
	//   Either w or wmin should be transmitted.
	//   If only w is included, it should be considered an exact requirement.
	WMin int64 `json:"wmin,omitempty"`

	// Field:
	//   h
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//   Height of the image in pixels.
	H int64 `json:"h,omitempty"`

	// Field:
	//   hmin
	// Scope:
	//   recommended
	// Type:
	//   integer
	// Description:
	// The minimum requested height of the image in pixels.
	// This option should be used for any rescaling of images by the client.
	// Either h or hmin should be transmitted.
	// If only h is included, it should be considered an exact requirement.
	HMin int64 `json:"hmin,omitempty"`

	// Field:
	//   mimes
	// Scope:
	//   optional
	// Type:
	//   array of strings
	// Default:
	//   All types allowed
	// Description:
	//   Whitelist of content MIME types supported.
	//   Popular MIME types include, but are not limited to “image/jpg” “image/gif”.
	//   Each implementing Exchange should have their own list of supported types in the integration docs.
	//   See Wikipedia's MIME page for more information and links to all IETF RFCs.
	//   If blank, assume all types are allowed.
	MIMEs []string `json:"mimes,omitempty"`

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
