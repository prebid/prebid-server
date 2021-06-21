package native1

// 7.2 Context Sub Type IDs
//
// Next-level context in which the ad appears.
// Again this reflects the primary context, and does not imply no presence of other elements.
// For example, an article is likely to contain images but is still first and foremost an article.
// SubType should only be combined with the primary context type as indicated (ie for a context type of 1, only context subtypes that start with 1 are valid).
type ContextSubType int64

const (
	ContextSubTypeGeneral       ContextSubType = 10 // General or mixed content.
	ContextSubTypeArticle       ContextSubType = 11 // Primarily article content (which of course could include images, etc as part of the article)
	ContextSubTypeVideo         ContextSubType = 12 // Primarily video content
	ContextSubTypeAudio         ContextSubType = 13 // Primarily audio content
	ContextSubTypeImage         ContextSubType = 14 // Primarily image content
	ContextSubTypeUserGenerated ContextSubType = 15 // User-generated content - forums, comments, etc
	ContextSubTypeSocial        ContextSubType = 20 // General social content such as a general social network
	ContextSubTypeEmail         ContextSubType = 21 // Primarily email content
	ContextSubTypeChat          ContextSubType = 22 // Primarily chat/IM content
	ContextSubTypeSelling       ContextSubType = 30 // Content focused on selling products, whether digital or physical
	ContextSubTypeAppStore      ContextSubType = 31 // Application store/marketplace
	ContextSubTypeProductReview ContextSubType = 32 // Product reviews site primarily (which may sell product secondarily)

	// 500+ To be defined by the exchange
)
