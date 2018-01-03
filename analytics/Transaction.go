package analytics

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"net/http"
	"time"
)

const (
	EVENT_TYPE = "bid_request"
	//other event types
)

type Event interface {
	LogEvent()
}

//For every event that occurs during a transaction
type BidRequest struct {
	BidderName string
	Request    string
	Response   string
	Time       time.Duration
	EventType  string
	//More relevant parameters
}

//Implements the Event interface
func (ar *BidRequest) LogEvent() {

}

//One for each request to an endpoint
type TransactionObject struct {
	Type                string
	Status              int
	Events              []Event
	Error               error
	Request             http.Request
	Response            http.Response
	Time                time.Time
	account             string
	bidder              openrtb_ext.BidderName
	PBSRegion           string
	userRegion          string
	uidTracked          bool
	bidPrice            float64
	domain              string
	referrerUrl         string
	appID               string
	responseMediaType   pbs.MediaType
	latform             string
	Timeout             time.Duration
	size                openrtb.Format
	userAgent           string
	adUnitCode          string
	dealID              string
	adServerTargeting   string
	transactionID       string
	limitAdTrackingFlag bool
	//relevant parameters
}
