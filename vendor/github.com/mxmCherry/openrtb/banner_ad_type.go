package openrtb

// 5.2 Banner Ad Types
//
// Types of ads that can be accepted by the exchange unless restricted by publisher site settings.
type BannerAdType int8

const (
	BannerAdTypeXHTMLTextAd   BannerAdType = 1 // XHTML Text Ad (usually mobile)
	BannerAdTypeXHTMLBannerAd BannerAdType = 2 // XHTML Banner Ad. (usually mobile)
	BannerAdTypeJavaScriptAd  BannerAdType = 3 // JavaScript Ad; must be valid XHTML (i.e., Script Tags Included)
	BannerAdTypeIframe        BannerAdType = 4 // iframe
)
