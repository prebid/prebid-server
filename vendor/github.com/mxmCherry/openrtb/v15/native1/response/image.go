package response

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/native1"
)

// 5.4 Object: Image
//
// Corresponds to the Image Object in the request.
// The Image object to be used for all image elements of the Native ad such as Icons, Main Image, etc.
//
// It is recommended that if assetsurl/dcourl is being used rather than embedded assets, that an image of each recommended aspect ratio (per the Image Types table) be provided forimage type 3.
type Image struct {
	// Field:
	//   type
	// Scope:
	//   optional
	// Type:
	//   integer
	// Description:
	//    Required for assetsurl or dcourl responses, not required for embedded asset responses.
	//   The type of image element being submitted from the Image Asset Types table.
	Type native1.ImageAssetType `json:"type,omitempty"`

	// Field:
	//   url
	// Scope:
	//   required
	// Type:
	//   string
	// Description:
	//   URL of the image asset
	URL string `json:"url"`

	// Field:
	//   w
	// Scope:
	//   recommended
	// Type:
	//   int
	// Description:
	//   Width of the image in pixels.
	//   Recommended for embedded asset responses.
	//   Required for assetsurl/dcourlresponses if multiple assets of same type submitted.
	W int64 `json:"w,omitempty"`

	// Field:
	//   h
	// Scope:
	//   recommended
	// Type:
	//   int
	// Description:
	//   Height of the image in pixels.
	//   Recommended for embedded asset responses.
	//   Required for assetsurl/dcourl responses if multiple assets of same type submitted.
	H int64 `json:"h,omitempty"`

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
