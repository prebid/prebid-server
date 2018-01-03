package analytics

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"net/http"
	"time"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/adapters"
)

const (
	BID_REQUEST  = "bid_request"
	BID_RESPONSE = "bid_response"
	AUCTION      = "/openrtb2/auction"
	COOKIE_SYNC  = "/cookiesync"
	SETUID       = "/setuid"
	GETUID       = "/getuid"

	//other event types
)

type RequestType string

type Event interface {
	LogEvent()
}

//For every event that occurs during a transaction
type BidRequests struct {
	BidderNames []openrtb_ext.BidderName
	Type        RequestType

	Requests map[openrtb_ext.BidderName]*openrtb.BidRequest

	//More relevant parameters
}

type BidResponse struct {
	Response string
	Type     RequestType
}


type AdapterRequests struct {
	AdapterName openrtb_ext.BidderName
	Requests []*adapters.RequestData
}


//Implements the Event interface
func (ar *BidRequests) LogEvent() {

}


func (a *AdapterRequests) LogEvent(){

}

//One for each request to an endpoint
type TransactionObject struct {
	Type                RequestType
	Status              int
	Events              []Event
	Error               []error
	Request             http.Request
	Response            http.Response
	Time                time.Time
	Account             string
	Bidder              openrtb_ext.BidderName
	PBSRegion           string
	UserRegion          string
	UidTracked          bool
	BidPrice            float64
	Domain              string
	ReferrerUrl         string
	AppID               string
	ResponseMediaType   pbs.MediaType
	Latform             string
	Timeout             time.Duration
	Size                openrtb.Format
	UserAgent           string
	AdUnitCode          string
	DealID              string
	AdServerTargeting   string
	TransactionID       string
	LimitAdTrackingFlag bool
	//relevant parameters
}

func SetupAnalytics(config *config.Configuration) Module {
	return FileLogger{}.Setup(config)
}

type Module interface {
	Log(*TransactionObject)
}

type FileLogger struct {
}

func (fl *FileLogger) Setup(cfg *config.Configuration) (*FileLogger) {
	return fl
}

func (fl *FileLogger) Log(object *TransactionObject) {}
