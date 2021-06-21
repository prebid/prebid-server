package request

import "encoding/json"

// 4.2 Asset Object
//
// The main container object for each asset requested or supported by Exchange on behalf of the rendering client.
// Any object that is required is to be flagged as such.
// Only one of the {title,img,video,data} objects should be present in each object.
// All others should be null/absent.
// The id is to be unique within the AssetObject array so that the response can be aligned.
//
// To be more explicit, it is the ID of each asset object that maps the response to the request.
// So if a request for a title object is sent with id 1, then the response containing the title should have an id of 1.
//
// Since version 1.1 of the spec, there are recommended sizes/lengths/etc with some of the asset types.
// The goal for asset requirements standardization is to facilitate adoption of native1 by DSPs by limiting the diverse types/sizes/requirements of assets they must have available to purchase a native ad impression.
// While great diversity may exist in publishers, advertisers/DSPs can not be expected to provide infinite headline lengths, thumbnail aspect ratios, etc.
// While we have not gone as far as creating a single standard, we've honed in on a few options that cover the most common cases.
// SSPs can deviate from these standards, but should understand they may limit applicable DSP demand by doing so.
// DSPs should feel confident that if they support these standards they'll be able to access most native1 inventory.
type Asset struct {
	// Field:
	//   id
	// Scope:
	//   required
	// Type:
	//   int
	// Description:
	//   Unique asset ID, assigned by exchange.
	//   Typically a counter for the array.
	ID int64 `json:"id"`

	// Field:
	//   required
	// Scope:
	//   optional
	// Type:
	//   int
	// Default:
	//   0
	// Description:
	//   Set to 1 if asset is required (exchange will not accept a bid without it)
	Required int8 `json:"required,omitempty"`

	// Field:
	//   title
	// Scope:
	//   recommended (each asset object may contain only one of title, img, data or video)
	// Type:
	//   object
	// Description:
	//   Title object for title assets.
	//   See TitleObject definition.
	//   Each asset object may contain only one of title, img, data or video.
	Title *Title `json:"title,omitempty"`

	// Field:
	//   img
	// Scope:
	//   recommended (each asset object may contain only one of title, img, data or video)
	// Type:
	//   object
	// Description:
	//   Image object for image assets.
	//   See ImageObject definition.
	//   Each asset object may contain only one of title, img, data or video.
	Img *Image `json:"img,omitempty"`

	// Field:
	//   video
	// Scope:
	//   optional (each asset object may contain only one of title, img, data or video)
	// Type:
	//   object
	// Description:
	//   Video object for video assets.
	//   See the Video request object definition.
	//   Note that in-stream (ie preroll, etc) video ads are not part of Native.
	//   Native ads may contain a video as the ad creative itself.
	//   Each asset object may contain only one of title, img, data or video.
	Video *Video `json:"video,omitempty"`

	// Field:
	//   data
	// Scope:
	//   recommended (each asset object may contain only one of title, img, data or video)
	// Type:
	//   object
	// Description:
	//   Data object for brand name, description, ratings, prices etc.
	//   See DataObject definition.
	//   Each asset object may contain only one of title, img, data or video.
	Data *Data `json:"data,omitempty"`

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
