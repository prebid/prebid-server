package filesystem

import (
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type logAuction struct {
	Status               int
	Errors               []error
	Request              *openrtb2.BidRequest
	Response             *openrtb2.BidResponse
	Account              *config.Account
	StartTime            time.Time
	HookExecutionOutcome []hookexecution.StageOutcome
}

type logVideo struct {
	Status        int
	Errors        []error
	Request       *openrtb2.BidRequest
	Response      *openrtb2.BidResponse
	VideoRequest  *openrtb_ext.BidRequestVideo
	VideoResponse *openrtb_ext.BidResponseVideo
	StartTime     time.Time
}

type logSetUID struct {
	Status  int
	Bidder  string
	UID     string
	Errors  []error
	Success bool
}

type logUserSync struct {
	Status       int
	Errors       []error
	BidderStatus []*analytics.CookieSyncBidder
}

type logAMP struct {
	Status               int
	Errors               []error
	Request              *openrtb2.BidRequest
	AuctionResponse      *openrtb2.BidResponse
	AmpTargetingValues   map[string]string
	Origin               string
	StartTime            time.Time
	HookExecutionOutcome []hookexecution.StageOutcome
}

type logNotificationEvent struct {
	Request *analytics.EventRequest `json:"request"`
	Account *config.Account         `json:"account"`
}
