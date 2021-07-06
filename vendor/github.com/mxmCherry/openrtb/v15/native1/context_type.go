package native1

// 7.1 Context Type IDs
//
// The context in which the ad appears - what type of content is surrounding the ad on the
// page at a high level.
// This maps directly to the new Deep Dive on In-Feed Ad Units. This
// denotes the primary context, but does not imply other content may not exist on the
// page - for example it's expected that most content platforms have some social
// components, etc.
type ContextType int64

const (
	ContextTypeContent ContextType = 1 // Content-centric context such as newsfeed, article, image gallery, video gallery, or similar.
	ContextTypeSocial  ContextType = 2 // Social-centric context such as social network feed, email, chat, or similar.
	ContextTypeProduct ContextType = 3 // Product context such as product listings, details, recommendations, reviews, or similar.

	// 500+ To be defined by the exchange.
)
