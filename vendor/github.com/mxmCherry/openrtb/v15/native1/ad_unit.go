package native1

// 7.2 Native Ad Unit IDs in version 1.1.
// Deprecated/removed in version 1.2.
//
// Below is a list of the core ad unit ids described by IAB here http://www.iab.net/media/file/IABNativeAdvertisingPlaybook120413.pdf
//
// In feed unit is essentially a layout, it has been removed from the list.
// The in feed units can be identified via the layout parameter on the request.
//
// An implementing exchange may not support all asset variants or introduce new ones unique to that system.
type AdUnit int64

const (
	AdUnitPaidSearch           AdUnit = 1 // Paid Search Units
	AdUnitRecommendationWidget AdUnit = 2 // Recommendation Widgets
	AdUnitPromotedListing      AdUnit = 3 // Promoted Listings
	AdUnitInAd                 AdUnit = 4 // In-Ad (IAB Standard) with Native Element Units
	AdUnitCustom               AdUnit = 5 // Custom /”Can’t Be Contained”

	// 500+ Reserved for Exchange specific formats.
)
