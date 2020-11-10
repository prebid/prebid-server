package analytics

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

/*
  	PBSAnalyticsModule must be implemented by any analytics module that does transactional logging.

	New modules can use the /analytics/endpoint_data_objects, extract the
	information required and are responsible for handling all their logging activities inside LogAuctionObject, LogAmpObject
	LogCookieSyncObject and LogSetUIDObject method implementations.
*/

type PBSAnalyticsModule interface {
	LogAuctionObject(*AuctionObject)
	LogVideoObject(*VideoObject)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject)
	LogNotificationEventObject(*NotificationEvent)
}

//Loggable object of a transaction at /openrtb2/auction endpoint
type AuctionObject struct {
	Status   int
	Errors   []error
	Request  *openrtb.BidRequest
	Response *openrtb.BidResponse
	Account  *config.Account
}

//Loggable object of a transaction at /openrtb2/amp endpoint
type AmpObject struct {
	Status             int
	Errors             []error
	Request            *openrtb.BidRequest
	AuctionResponse    *openrtb.BidResponse
	AmpTargetingValues map[string]string
	Origin             string
}

//Loggable object of a transaction at /openrtb2/video endpoint
type VideoObject struct {
	Status        int
	Errors        []error
	Request       *openrtb.BidRequest
	Response      *openrtb.BidResponse
	VideoRequest  *openrtb_ext.BidRequestVideo
	VideoResponse *openrtb_ext.BidResponseVideo
}

//Loggable object of a transaction at /setuid
type SetUIDObject struct {
	Status  int
	Bidder  string
	UID     string
	Errors  []error
	Success bool
}

//Loggable object of a transaction at /cookie_sync
type CookieSyncObject struct {
	Status       int
	Errors       []error
	BidderStatus []*usersync.CookieSyncBidders
}

// NotificationEvent is a loggable object
type NotificationEvent struct {
	Request *EventRequest   `json:"request"`
	Account *config.Account `json:"account"`
}
