package analytics

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/usersync"
)

type RequestType string

const (
	COOKIE_SYNC RequestType = "/cookie_sync"
	AUCTION     RequestType = "/openrtb2/auction"
	SETUID      RequestType = "/set_uid"
	AMP         RequestType = "/openrtb2/amp"
)

//Loggable object of a transaction at /openrtb2/auction endpoint
type AuctionObject struct {
	Type     RequestType
	Status   int
	Errors   []error
	Request  *openrtb.BidRequest
	Response *openrtb.BidResponse
}

//Loggable object of a transaction at /openrtb2/amp endpoint
type AmpObject struct {
	Type               RequestType
	Status             int
	Errors             []error
	Request            *openrtb.BidRequest
	AuctionResponse    *openrtb.BidResponse
	AmpTargetingValues map[string]string
	Origin             string
}

//Loggable object of a transaction at /setuid
type SetUIDObject struct {
	Type    RequestType
	Status  int
	Bidder  string
	UID     string
	Errors  []error
	Success bool
}

//Loggable object of a transaction at /cookie_sync
type CookieSyncObject struct {
	Type         RequestType
	Status       int
	Errors       []error
	BidderStatus []*usersync.CookieSyncBidders
}
