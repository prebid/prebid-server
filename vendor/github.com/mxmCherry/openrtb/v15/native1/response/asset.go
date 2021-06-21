package response

import "encoding/json"

// 5.2 Object: Asset
//
// Corresponds to the Asset Object in the request. The main container object for each asset
// requested or supported by Exchange on behalf of the rendering client. Any object that is
// required is to be flagged as such. Only one of the {title,img,video,data} objects should be
// present in each object. All others should be null/absent. The id is to be unique within the
// AssetObject array so that the response can be aligned.
type Asset struct {
	// Field:
	//   id
	// Scope:
	//   optional
	// Type:
	//   int
	// Description:
	//   Optional if assetsurl/dcourl is being used; required if embedded asset is being used.
	ID *int64 `json:"id,omitempty"`

	// Field:
	//   required
	// Scope:
	//   optional
	// Type:
	//   int
	// Default:
	//   0
	// Description:
	//   Set to 1 if asset is required. (bidder requires it to be displayed).
	Required int8 `json:"required,omitempty"`

	// Field:
	//   title
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   Title object for title assets.
	//   Asset object may contain only one of title, img, data or video.
	Title *Title `json:"title,omitempty"`

	// Field:
	//   img
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   Image object for image assets.
	//   Asset object may contain only one of title, img, data or video.
	Img *Image `json:"img,omitempty"`

	// Field:
	//   video
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   Video object for video assets. See Video response object definition.
	//   Note that in-stream video ads are not part of Native.
	//   Native ads may contain a video as the ad creative itself.
	//   Asset object may contain only one of title, img, data or video.
	Video *Video `json:"video,omitempty"`

	// Field:
	//   data
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   Data object for ratings, prices etc.
	//   Asset object may contain only one of title, img, data or video.
	Data *Data `json:"data,omitempty"`

	// Field:
	//   link
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   Link object for call to actions. The link object applies if the asset item is activated (clicked).
	//   If there is no link object on the asset, the parent link object on the bid response applies.
	Link *Link `json:"link,omitempty"`

	// Field:
	//   ext
	// Scope:
	//   optional
	// Type:
	//   object
	// Description:
	//   This object is a placeholder that may contain custom JSON agreed to by the parties to support flexibility beyond the standard defined in this specification
	//   Bidders are encouraged not to use asset.ext for exchanging text assets.
	//   Use data.ext with custom type instead.
	Ext json.RawMessage `json:"ext,omitempty"`
}
