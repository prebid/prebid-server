package native1

// 7.3 Placement Type IDs
//
// The FORMAT of the ad you are purchasing, separate from the surrounding context
type PlacementType int64

const (
	PlacementTypeFeed                 = 1 // In the feed of content - for example as an item inside the organic feed/grid/listing/carousel.
	PlacementTypeAtomicContentUnit    = 2 // In the atomic unit of the content - IE in the article page or single image page
	PlacementTypeOutsideCoreContent   = 3 // Outside the core content - for example in the ads section on the right rail, as a banner-style placement near the content, etc.
	PlacementTypeRecommendationWidget = 4 // Recommendation widget, most commonly presented below the article content.

	// 500+ To be defined by the exchange
)
