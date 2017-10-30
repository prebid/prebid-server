package openrtb_ext

// ExtSeatBid defines the contract for bidresponse.seatbid.ext
type ExtSeatBid struct {
	Prebid ExtSeatBidPrebid `json:"prebid"`
}

// ExtSeatBidPrebid defines the contract for bidresponse.seatbid.ext.prebid
type ExtSeatBidPrebid struct {
	Cookie             ExtSeatBidCookie `json:"cookie"`
	Debug              ExtSeatBidDebug  `json:"debug"`
	Errors             []string         `json:"errors"`
	ResponseTimeMillis int              `json:"responsetimemillis"`
	UserSync           ExtUserSyncs     `json:"usersync"`
}

// ExtSeatBidCookie defines the contract for bidresponse.seatbid.ext.prebid.cookie
type ExtSeatBidCookie struct {
	Status CookieStatus `json:"status"`
}

// ExtSeatBidDebug defines the contract for bidresponse.seatbid.ext.prebid.debug
type ExtSeatBidDebug struct {
	ServerCalls []ExtServerCall `json:"servercalls"`
}

// ExtUserSyncs defines the contract for bidresponse.seatbid.ext.prebid.usersync
type ExtUserSyncs struct {
	Syncs []ExtUserSync
}

// ExtUserSyncs defines the contract for bidresponse.seatbid.ext.prebid.usersync.syncs[i]
type ExtUserSync struct {
	Url  string       `json:"url"`
	Type UserSyncType `json:"type"`
}

// SeatBidCookieExt defines the contract for a bidresponse.seatbid.ext.prebid.debug.servercalls[i]
type ExtServerCall struct {
	Uri          string `json:"uri"`
	RequestBody  string `json:"requestbody"`
	Responsebody string `json:"responsebody"`
	Status       int    `json:"status"`
}

// CookieStatus describes the allowed values for bidresponse.seatbid.ext.prebid.cookie.status
type CookieStatus string

const (
	CookieNone      CookieStatus = "none"
	CookieExpired                = "expired"
	CookieAvailable              = "available"
)

// UserSyncType describes the allowed values for bidresponse.seatbid.ext.prebid.usersync.syncs[i].type
type UserSyncType string

const (
	UserSyncIframe UserSyncType = "iframe"
	UserSyncPixel               = "pixel"
)
