package openrtb2

import "encoding/json"

// 3.2.16 Object: Content
//
// This object describes the content in which the impression will appear, which may be syndicated or nonsyndicated content.
// This object may be useful when syndicated content contains impressions and does not necessarily match the publisher’s general content.
// The exchange might or might not have knowledge of the page where the content is running, as a result of the syndication method.
// For example might be a video impression embedded in an iframe on an unknown web property or device.
type Content struct {

	// Attribute:
	//   id
	// Type:
	//   string
	// Description:
	//   ID uniquely identifying the content.
	ID string `json:"id,omitempty"`

	// Attribute:
	//   episode
	// Type:
	//   integer
	// Description:
	//   Episode number.
	Episode int64 `json:"episode,omitempty"`

	// Attribute:
	//   title
	// Type:
	//   string
	// Description:
	//   Content title.
	//   Video Examples: “Search Committee” (television), “A New
	//   Hope” (movie), or “Endgame” (made for web).
	//   Non-Video Example: “Why an Antarctic Glacier Is Melting So
	//   Quickly” (Time magazine article).
	Title string `json:"title,omitempty"`

	// Attribute:
	//   series
	// Type:
	//   string
	// Description:
	//   Content series.
	//   Video Examples: “The Office” (television), “Star Wars” (movie),
	//   or “Arby ‘N’ The Chief” (made for web).
	//   Non-Video Example: “Ecocentric” (Time Magazine blog).
	Series string `json:"series,omitempty"`

	// Attribute:
	//   season
	// Type:
	//   string
	// Description:
	//   Content season (e.g., “Season 3”).
	Season string `json:"season,omitempty"`

	// Attribute:
	//   artist
	// Type:
	//   string
	// Description:
	//   Artist credited with the content.
	Artist string `json:"artist,omitempty"`

	// Attribute:
	//   genre
	// Type:
	//   string
	// Description:
	//   Genre that best describes the content (e.g., rock, pop, etc).
	Genre string `json:"genre,omitempty"`

	// Attribute:
	//   album
	// Type:
	//   string
	// Description:
	//   Album to which the content belongs; typically for audio.
	Album string `json:"album,omitempty"`

	// Attribute:
	//   isrc
	// Type:
	//   string
	// Description:
	//   International Standard Recording Code conforming to ISO-
	//   3901.
	ISRC string `json:"isrc,omitempty"`

	// Attribute:
	//   producer
	// Type:
	//   object
	// Description:
	//   Details about the content Producer (Section 3.2.17).
	Producer *Producer `json:"producer,omitempty"`

	// Attribute:
	//   url
	// Type:
	//   string
	// Description:
	//   URL of the content, for buy-side contextualization or review.
	URL string `json:"url,omitempty"`

	// Attribute:
	//   cat
	// Type:
	//   string array
	// Description:
	//   Array of IAB content categories that describe the content
	//   producer. Refer to List 5.1.
	Cat []string `json:"cat,omitempty"`

	// Attribute:
	//   prodq
	// Type:
	//   integer
	// Description:
	//   Production quality. Refer to List 5.13
	ProdQ *ProductionQuality `json:"prodq,omitempty"`

	// Attribute:
	//   videoquality
	// Type:
	//   integer; DEPRECATED
	// Description:
	//   Note: Deprecated in favor of prodq.
	//   Video quality. Refer to List 5.13.
	VideoQuality *ProductionQuality `json:"videoquality,omitempty"`

	// Attribute:
	//   context
	// Type:
	//   integer
	// Description:
	//   Type of content (game, video, text, etc.). Refer to List 5.18.
	Context ContentContext `json:"context,omitempty"`

	// Attribute:
	//   contentrating
	// Type:
	//   string
	// Description:
	//   Content rating (e.g., MPAA).
	ContentRating string `json:"contentrating,omitempty"`

	// Attribute:
	//   userrating
	// Type:
	//   string
	// Description:
	//   User rating of the content (e.g., number of stars, likes, etc.).
	UserRating string `json:"userrating,omitempty"`

	// Attribute:
	//   qagmediarating
	// Type:
	//   integer
	// Description:
	//   Media rating per IQG guidelines. Refer to List 5.19.
	QAGMediaRating IQGMediaRating `json:"qagmediarating,omitempty"`

	// Attribute:
	//   keywords
	// Type:
	//   string
	// Description:
	//   Comma separated list of keywords describing the content.
	Keywords string `json:"keywords,omitempty"`

	// Attribute:
	//   livestream
	// Type:
	//   integer
	// Description:
	//   0 = not live, 1 = content is live (e.g., stream, live blog).
	LiveStream int8 `json:"livestream,omitempty"`

	// Attribute:
	//   sourcerelationship
	// Type:
	//   integer
	// Description:
	//   0 = indirect, 1 = direct.
	SourceRelationship int8 `json:"sourcerelationship,omitempty"`

	// Attribute:
	//   len
	// Type:
	//   integer
	// Description:
	//   Length of content in seconds; appropriate for video or audio.
	Len int64 `json:"len,omitempty"`

	// Attribute:
	//   language
	// Type:
	//   string
	// Description:
	//   Content language using ISO-639-1-alpha-2.
	Language string `json:"language,omitempty"`

	// Attribute:
	//   embeddable
	// Type:
	//   integer
	// Description:
	//   Indicator of whether or not the content is embeddable (e.g.,
	//   an embeddable video player), where 0 = no, 1 = yes.
	Embeddable int8 `json:"embeddable,omitempty"`

	// Attribute:
	//   data
	// Type:
	//   object array
	// Description:
	//   Additional content data. Each Data object (Section 3.2.21)
	//   represents a different data source.
	Data []Data `json:"data,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
