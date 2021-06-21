package native1

// 7.5 Image Asset Types
//
// Below is a list of common image asset element types of native advertising at the time of writing this spec.
// This list is non-exhaustive and intended to be extended by the buyers and sellers as the format evolves.
//
// An implementing exchange may not support all asset variants or may introduce new ones unique to that system.
//
// In order to facilitate adoption, recommendations are made for both minimum sizes and aspect ratios.
// We speak here of 'minimum maximum height' or ‘max height of at least’, which means the SSP should support a max height of at least this value.
// They are free to support larger, but the DSP knows that if they have an image of this size it will be accepted.
// Note that SSPs will be responsible for sizing image to exact size if min-maxheight framework is used; exact size may not be available at bid request time.
// Width is calculated from the 3 supported aspect ratios.
// Note we merged the prior overlapping type 1 and type 2 as just type 1 - to be used for app icon, brand logo, or similar.
type ImageAssetType int64

const (
	ImageAssetTypeIcon ImageAssetType = 1 // Icon; Icon image; Optional. Max height: at least 50; aspect ratio: 1:1
	ImageAssetTypeLogo ImageAssetType = 2 // Logo; Logo image for the brand/app. Deprecated in version 1.2 - use type 1 Icon.

	// Main; Large image preview for the ad. At least one of 2 size variants required:
	//   Small Variant:
	//     max height: at least 200
	//     max width: at least 200, 267, or 382
	//     aspect ratio: 1:1, 4:3, or 1.91:1
	//   Large Variant:
	//     max height: at least 627
	//     max width: at least 627, 836, or 1198
	//     aspect ratio: 1:1, 4:3, or 1.91:1
	ImageAssetTypeMain ImageAssetType = 3

	// 500+ XXX; Reserved for Exchange specific usage numbered above 500. No recommendations
)
