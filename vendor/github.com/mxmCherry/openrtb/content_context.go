package openrtb

// 5.18 Content Context
//
// Various options for indicating the type of content being used or consumed by the user in which the impression will appear.
// This OpenRTB list has values derived from the Inventory Quality Guidelines (IQG).
// Practitioners should keep in sync with updates to the IQG values.
type ContentContext int8

const (
	ContentContextVideo       ContentContext = 1 // Video (i.e., video file or stream such as Internet TV broadcasts)
	ContentContextGame        ContentContext = 2 // Game (i.e., an interactive software game)
	ContentContextMusic       ContentContext = 3 // Music (i.e., audio file or stream such as Internet radio broadcasts)
	ContentContextApplication ContentContext = 4 // Application (i.e., an interactive software application)
	ContentContextText        ContentContext = 5 // Text (i.e., primarily textual document such as a web page, eBook, or news article)
	ContentContextOther       ContentContext = 6 // Other (i.e., none of the other categories applies)
	ContentContextUnknown     ContentContext = 7 // Unknown
)
