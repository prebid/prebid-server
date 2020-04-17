package openrtb

// 5.9 Video Placement Types
//
// Various types of video placements derived largely from the IAB Digital Video Guidelines.
type VideoPlacementType int8

const (
	VideoPlacementTypeInStream                   VideoPlacementType = 1 // In-Stream. Played before, during or after the streaming video content that the consumer has requested (e.g., Pre-roll, Mid-roll, Post-roll).
	VideoPlacementTypeInBanner                   VideoPlacementType = 2 // In-Banner. Exists within a web banner that leverages the banner space to deliver a video experience asopposed to another static or rich media format. The format relies on the existence of displayad inventory on the page for its delivery.
	VideoPlacementTypeInArticle                  VideoPlacementType = 3 // In-Article. Loads and plays dynamically between paragraphs of editorial content; existing as a standalonebranded message.
	VideoPlacementTypeInFeed                     VideoPlacementType = 4 // In-Feed. Found in content, social, or product feeds.
	VideoPlacementTypeInterstitialSliderFloating VideoPlacementType = 5 // Interstitial/Slider/Floating. Covers the entire or a portion of screen area, but is always on screen while displayed (i.e.cannot be scrolled out of view). Note that a full-screen interstitial (e.g., in mobile) can bedistinguished from a floating/slider unit by the imp.instl field.
)
