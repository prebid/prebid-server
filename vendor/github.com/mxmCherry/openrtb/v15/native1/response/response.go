// Package response provides OpenRTB Native 1.2 response types
// (section "5 Native Ad Response Markup Details")
//
// https://iabtechlab.com/standards/openrtb-native/
// https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf
package response

import "encoding/json"

// 5.1 Object: Response
//
// The native1 object is the top level JSON object which identifies a native response.
// Note: Prior to VERSION 1.1, the native1 response’s root node was an object with a single field “native” that would contain the object above as its value.
// The Native Object specified above is now the root object.
//
// Note re: assetsurl format responses: In the case of assetsurl or dcourl (beta) bidding, since the ultimate buyer/creative engine cannot alter the assets response based on the details inside the assets request (as it does not receive said request), it is critical that all required assets are provided, and such communications will need to be handled offline for recommended/optional elements.
//
// In the normal embedded response, certain attributes of the response are assumed based on matching the ID of the asset object in the response to the ID of the asset object in the request.
// Since the asset response will not be reading the asset request directly in this implementation, that information is added as option in the below object definitions and marked for that use case.
//
// It is also recommended that where the standard specifies multiple options for an element, that all options be provided.
// IE all 4 supported thumbnail aspect ratios and all 3 supported title lengths.
//
// The ID component of the asset responses can be omitted.
//
// Note that this change to provide the metadata description of the asset in the response, rather than using the asset ID to implicitly specify that, may be reflected into the inline version of responses in a future version to align the two methods of replying.
// Making that change now would break backwards compatibility of the inline response mechanism.
type Response struct {
	// Field:
	//   ver
	// Scope:
	//   recommended
	// Type:
	//   string
	// Default:
	//   1.2
	// Description:
	//   Version of the Native Markup version in use.
	Ver string `json:"ver,omitempty"`

	// Field:
	//   assets
	// Scope:
	//   recommended
	// Type:
	//   object array
	// Description:
	//   List of native1 ad’s assets.
	//   Required if no assetsurl.
	//   Recommended as fallback even if assetsurl is provided.
	Assets []Asset `json:"assets,omitempty"`

	// Field:
	//   assetsurl
	// Scope:
	//   optional
	// Type:
	//   string
	// Description:
	//   URL of an alternate source for the assets object.
	//   The expected response is a JSON object mirroring the assets object in the bid response, subject to certain requirements as specified in the individual objects.
	//   Where present, overrides the asset object in the response.
	//   The provided “assetsurl” or “dcourl” should be expected to provide a valid response when called in any context, including importantly the brand name and example thumbnails and headlines (to use for reporting, blacklisting, etc), but it is understood the final actual call should come from the client device from which the ad will be rendered to give the buyer the benefit of the cookies and header data available in that context.
	AssetsURL string `json:"assetsurl,omitempty"`

	// Field:
	//   dcourl
	// Scope:
	//   optional
	// Type:
	//   string
	// Description:
	//   URL where a dynamic creative specification may be found for populating this ad, per the Dynamic Content Ads Specification.
	//   Note this is a beta option as the interpretation of the Dynamic Content Ads Specification and how to assign those elements into a native1 ad is outside the scope of this spec and must be agreed offline between the parties or as may be specified in a future revision of the Dynamic Content Ads spec.
	//   Where present, overrides the asset object in the response.
	DCOURL string `json:"dcourl,omitempty"`

	// Field:
	//   link
	// Scope:
	//   required
	// Type:
	//   object
	// Description:
	//   Destination Link.
	//   This is default link object for the ad.
	//   Individual assets can also have a link object which applies if the asset is activated(clicked).
	//   If the asset doesn’t have a link object, the parent link object applies.
	Link Link `json:"link"`

	// Field:
	//   imptrackers
	// Scope:
	//   optional
	// Type:
	//   string array
	// Description:
	//   Array of impression tracking URLs, expected to return a 1x1 image or 204 response - typically only passed when using 3rd party trackers.
	//   To be deprecated - replaced with eventtrackers.
	ImpTrackers []string `json:"imptrackers,omitempty"`

	// Field:
	//   jstracker
	// Scope:
	//   optional
	// Type:
	//   string
	// Description:
	//   Optional JavaScript impression tracker.
	//   This is a valid HTML, Javascript is already wrapped in <script> tags.
	//   It should be executed at impression time where it can be supported.
	//   To be deprecated - replaced with eventtrackers.
	JSTracker string `json:"jstracker,omitempty"`

	// Field:
	//   eventtrackers
	// Scope:
	//   optional
	// Type:
	//   Array of objects
	// Description:
	//   Array of tracking objects to run with the ad, in response to the declared supported methods in the request.
	//   Replaces imptrackers and jstracker, to be deprecated.
	EventTrackers []EventTracker `json:"eventtrackers,omitempty"`

	// Field:
	//   privacy
	// Scope:
	//   optional
	// Type:
	//   string
	// Description:
	//   If support was indicated in the request, URL of a page informing the user about the buyer’s targeting activity.
	Privacy string `json:"privacy,omitempty"`

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
