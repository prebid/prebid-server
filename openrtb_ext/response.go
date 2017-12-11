package openrtb_ext

// ExtBidResponse defines the contract for bidresponse.ext
type ExtBidResponse struct {
	Debug *ExtResponseDebug `json:"debug,omitempty"`
	// ExtResponseErrors defines the contract for bidresponse.ext.errors
	Errors map[BidderName][]string `json:"errors,omitempty"`
	// ExtResponseTimeMillis defines the contract for bidresponse.ext.responsetimemillis
	ResponseTimeMillis map[BidderName]int `json:"responsetimemillis,omitempty"`
	// ExtResponseUserSync defines the contract for bidresponse.ext.usersync
	Usersync map[BidderName]*ExtResponseSyncData `json:"usersync,omitempty"`
}

// ExtResponseDebug defines the contract for bidresponse.ext.debug
type ExtResponseDebug struct {
	// HttpCalls defines the contract for bidresponse.ext.debug.httpcalls
	HttpCalls map[BidderName][]*ExtHttpCall `json:"httpcalls,omitempty"`
}

// ExtResponseSyncData defines the contract for bidresponse.ext.usersync.{bidder}
type ExtResponseSyncData struct {
	Status CookieStatus `json:"status"`
	// Syncs must have length > 0
	Syncs []*ExtUserSync `json:"syncs"`
}

// ExtUserSync defines the contract for bidresponse.ext.usersync.{bidder}.syncs[i]
type ExtUserSync struct {
	Url  string       `json:"url"`
	Type UserSyncType `json:"type"`
}

// ExtHttpCall defines the contract for a bidresponse.ext.debug.httpcalls.{bidder}[i]
type ExtHttpCall struct {
	Uri          string `json:"uri"`
	RequestBody  string `json:"requestbody"`
	ResponseBody string `json:"responsebody"`
	Status       int    `json:"status"`
}

// CookieStatus describes the allowed values for bidresponse.ext.usersync.{bidder}.status
type CookieStatus string

const (
	CookieNone      CookieStatus = "none"
	CookieExpired   CookieStatus = "expired"
	CookieAvailable CookieStatus = "available"
)

// UserSyncType describes the allowed values for bidresponse.ext.usersync.{bidder}.syncs[i].type
type UserSyncType string

const (
	UserSyncIframe UserSyncType = "iframe"
	UserSyncPixel  UserSyncType = "pixel"
)
