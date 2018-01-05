package analytics

import (
	"github.com/chasex/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	"time"
	"log"
	"encoding/json"
	"encoding/base64"
	"bytes"
	"fmt"
	"net/http"
)

const (
	BID_REQUEST = "bid_request"
	TRANSACTION = "transaction"
	AUCTION     = "/openrtb2/auction"
	COOKIE_SYNC = "/cookiesync"
	SETUID      = "/setuid"
	GETUID      = "/getuid"

	//other event types
)

type RequestType string

type LoggableEvent interface {
	Log() (content []byte)
}

//One for each request to an endpoint
type AuctionObject struct {
	Type                RequestType
	Status              int
	AdapterRequests		[]AdapterRequests
	Events              []LoggableEvent
	Error               []error
	Request             openrtb.BidRequest
	Response            openrtb.BidResponse
	Time                time.Time
	Account             string
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

type CookieSyncObject struct{
	Type                RequestType
	Status              int
	Events              []LoggableEvent
	Error               []error
	Request             http.Request
	Response            openrtb.BidResponse
	Time                time.Time
	Account             string
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
}

type AdapterRequests struct {
	Type     RequestType
	Requests []*adapters.RequestData
}

func (a *AdapterRequests) Log() (content []byte) {
	for _, data := range a.Requests{
		var b []byte
		base64.StdEncoding.Decode(b, data.Body)
		data.Body = b
	}
	var err error
	if content, err = json.Marshal(a); err!=nil{
		fmt.Printf("Error in adapter requests: %v", err)
	}
	return
}

func (to *AuctionObject) Log() (content []byte) {

	e := to.Events
	to.Events = make([]LoggableEvent, 0)
	var b bytes.Buffer

	for _, eve := range e {
		b.Write(eve.Log())
	}

	c, err := json.Marshal(to)
	fmt.Printf("err %v", err)
	b.Write(c)
	return b.Bytes()
}

func SetupAnalytics(config *config.Configuration) Module {
	fl := FileLogger{}
	fl.Setup(config)
	return &fl
}

type Module interface {
	LogToModule(LoggableEvent)
}

type FileLogger struct {
	*glog.Logger
}

func (fl *FileLogger) Setup(cfg *config.Configuration) *FileLogger {
	options := glog.LogOptions{
		File:  "./transactions.log",
		Flag:  glog.LstdFlags,
		Level: glog.Ldebug,
		Mode:  glog.R_Day,
	}
	var err error
	fl.Logger, err = glog.New(options)
	if err != nil {
		log.Printf("File Logger could not be initialized. Error: %v", err)
	}
	return fl
}

func (fl *FileLogger) LogToModule(event LoggableEvent) {
	var b bytes.Buffer
	b.Write(event.Log())
	fl.Debug(b.String())
	fl.Flush()
}
