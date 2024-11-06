package analytics

import (
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Module must be implemented by analytics modules to extract the required information and logging
// activities. Do not use marshal the parameter objects directly as they can change over time. Use a separate
// model for each analytics module and transform as appropriate.
type Module interface {
	LogAuctionObject(*AuctionObject)
	LogVideoObject(*VideoObject)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject)
	LogNotificationEventObject(*NotificationEvent)
	Shutdown()
}

// Loggable object of a transaction at /openrtb2/auction endpoint
type AuctionObject struct {
	Status               int
	Errors               []error
	Response             *openrtb2.BidResponse
	Account              *config.Account
	StartTime            time.Time
	HookExecutionOutcome []hookexecution.StageOutcome
	SeatNonBid           []openrtb_ext.SeatNonBid
	RequestWrapper       *openrtb_ext.RequestWrapper
}

// Loggable object of a transaction at /openrtb2/amp endpoint
type AmpObject struct {
	Status               int
	Errors               []error
	AuctionResponse      *openrtb2.BidResponse
	AmpTargetingValues   map[string]string
	Origin               string
	StartTime            time.Time
	HookExecutionOutcome []hookexecution.StageOutcome
	SeatNonBid           []openrtb_ext.SeatNonBid
	RequestWrapper       *openrtb_ext.RequestWrapper
}

// Loggable object of a transaction at /openrtb2/video endpoint
type VideoObject struct {
	Status         int
	Errors         []error
	Response       *openrtb2.BidResponse
	VideoRequest   *openrtb_ext.BidRequestVideo
	VideoResponse  *openrtb_ext.BidResponseVideo
	StartTime      time.Time
	SeatNonBid     []openrtb_ext.SeatNonBid
	RequestWrapper *openrtb_ext.RequestWrapper
}

// Loggable object of a transaction at /setuid
type SetUIDObject struct {
	Status  int
	Bidder  string
	UID     string
	Errors  []error
	Success bool
}

// Loggable object of a transaction at /cookie_sync
type CookieSyncObject struct {
	Status       int
	Errors       []error
	BidderStatus []*CookieSyncBidder
}

type CookieSyncBidder struct {
	BidderCode   string        `json:"bidder"`
	NoCookie     bool          `json:"no_cookie,omitempty"`
	UsersyncInfo *UsersyncInfo `json:"usersync,omitempty"`
}

type UsersyncInfo struct {
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SupportCORS bool   `json:"supportCORS,omitempty"`
}

// NotificationEvent object of a transaction at /event
type NotificationEvent struct {
	Request *EventRequest   `json:"request"`
	Account *config.Account `json:"account"`
}
