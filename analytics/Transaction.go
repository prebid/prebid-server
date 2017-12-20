package analytics

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
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
	Events              []Event
	Error               error
	Request             string
	Response            string
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
	//relevant paramters
}

//Means to log every transaction
func (r *TransactionObject) Log() {
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
}

//An interface just in case there's more types of things to log - possibly.
type Transaction interface {
	Log()
}

//Main interface object that user configures
type AnalyticsLogger interface {
	LogTransaction(*TransactionObject)
}

//to log into a file
type FileLogger struct {
	fileName string
}

//configure
func (f *FileLogger) Setup() {

}

//implement AnalyticsLogger interface
func (f *FileLogger) LogTransaction(lo Transaction) {
	//TODO: Write to file
}
