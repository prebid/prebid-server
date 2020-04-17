package openrtb

// 5.7 Video Linearity
//
// Options for video linearity.
// “In-stream” or “linear” video refers to preroll, post-roll, or mid-roll video ads where the user is forced to watch ad in order to see the video content.
// “Overlay” or “non-linear” refer to ads that are shown on top of the video content.
//
// This OpenRTB list has values derived from the Inventory Quality Guidelines (IQG).
// Practitioners should keep in sync with updates to the IQG values.
type VideoLinearity int8

const (
	VideoLinearityLinearInStream   VideoLinearity = 1 // Linear / In-Stream
	VideoLinearityNonLinearOverlay VideoLinearity = 2 // Non-Linear / Overlay
)
