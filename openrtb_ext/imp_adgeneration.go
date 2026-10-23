package openrtb_ext

type ExtImpAdgeneration struct {
	Id string `json:"id"`
	// MarginTop is the value passed to ADGBrowserM.init({marginTop}) for
	// upper_billboard placements. It is an optional field that keeps behavior
	// aligned with Prebid.js (bidder params.marginTop).
	MarginTop string `json:"marginTop,omitempty"`
}
