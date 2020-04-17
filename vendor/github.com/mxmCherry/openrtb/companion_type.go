package openrtb

// 5.14 Companion Types
//
// Options to indicate markup types allowed for companion ads that apply to video and audio ads.
// This table is derived from VAST 2.0+ and DAAST 1.0 specifications.
// Refer to www.iab.com/guidelines/digital-video-suite for more information.
type CompanionType int8

const (
	CompanionTypeStatic CompanionType = 1 // Static Resource
	CompanionTypeHTML   CompanionType = 2 // HTML Resource
	CompanionTypeIframe CompanionType = 3 // iframe Resource
)
