package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chasex/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"log"
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
	Type               RequestType
	Status             int
	AdapterBidRequests []LoggableAdapterRequests
	Error              []error
	Request            openrtb.BidRequest
	Response           openrtb.BidResponse
	UserAgent          string
	//relevant parameters
}

type CookieSyncObject struct {
	Type     RequestType
	Status   int
	Events   []LoggableEvent
	Error    []error
	Request  http.Request
	Response http.Response
}

type LoggableAdapterRequests struct {
	Name     string
	Requests string
	Uri      string
	Header   http.Header
	Method   string
}

func (to *AuctionObject) Log() (content []byte) {
	content, err := json.Marshal(to)
	fmt.Printf("err %v", err)
	return
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

func (a *AuctionObject) MakeLoggableAdapterRequests(name openrtb_ext.BidderName, reqData []*adapters.RequestData) []LoggableAdapterRequests {
	ar := make([]LoggableAdapterRequests, len(reqData))
	for i, req := range reqData {
		ar[i] = LoggableAdapterRequests{
			Name:     string(name),
			Requests: string(req.Body),
			Uri:      req.Uri,
			Method:   req.Method,
			Header:   req.Headers,
		}
	}
	return ar
}
