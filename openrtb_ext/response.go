package openrtb_ext

// ExtBidResponse defines the contract for bidresponse.ext
type ExtBidResponse struct {
	Debug *ExtResponseDebug `json:"debug,omitempty"`
 	Errors *ExtResponseErrors `json:"errors,omitempty"`
	ResponseTimeMillis *ExtResponseTimeMillis `json:"responsetimemillis,omitempty"`
	Usersync *ExtResponseUserSync `json:"usersync,omitempty"`
}

// ExtResponseDebug defines the contract for bidresponse.ext.debug
type ExtResponseDebug struct {
	ServerCalls *ExtResponseServerCalls `json:"servercalls,omitempty"`
}

// ExtResponseServerCalls defines the contract for bidresponse.ext.debug.servercalls
type ExtResponseServerCalls struct {
	Appnexus []*ExtServerCall `json:"appnexus,omitempty"`
}

// ExtResponseErrors defines the contract for bidresponse.ext.errors
type ExtResponseErrors struct {
	Appnexus []string `json:"appnexus,omitempty"`
}

// ExtResponseTimeMillis defines the contract for bidresponse.ext.responsetimemillis
type ExtResponseTimeMillis struct {
	Appnexus int `json:"appnexus"`
}

// ExtResponseUserSync defines the contract for bidresponse.ext.usersync
type ExtResponseUserSync struct {
	Appnexus *ExtResponseSyncData `json:"appnexus,omitempty"`
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

// ExtServerCall defines the contract for a bidresponse.ext.debug.servercalls.{bidder}[i]
type ExtServerCall struct {
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
