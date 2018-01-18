package analytics

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"log"
	"net/http"
)

type Module interface {
	logAuctionObject(ao *AuctionObject)
	logSetUIDObject(so *SetUIDObject)
	logCookieSyncObject(cso *CookieSyncObject)
}

type Analytics struct {
	Enabled        bool           `mapstructre:"enabled"`
	File           FileLogger     `mapstructure:"file"`
	Graphite       GraphiteLogger `mapstructure:"graphite"`
	EnabledLoggers []*Module
}

//RequestTypes
const (
	AUCTION     = "/openrtb2/auction"
	COOKIE_SYNC = "/cookiesync"
	SETUID      = "/setuid"
)

type RequestType string

//Loggable object of a transaction at /openrtb2/auction endpoint
type AuctionObject struct {
	Type               RequestType
	Status             int
	Error              []error
	AdapterBidRequests []LoggableAdapterRequests
	Request            openrtb.BidRequest
	Response           openrtb.BidResponse
	UserAgent          string
}

//Bid requests from each adapter. This information is available in /openrtb2/auction response but necessary to log in case there are errors
type LoggableAdapterRequests struct {
	Name     string
	Requests string
	Uri      string
	Header   http.Header
	Method   string
}

//Loggable object of a transaction at /setuid
type SetUIDObject struct {
	Type    RequestType
	Status  int
	Error   []error
	Cookie  string
	Success bool
}

//Loggable object of a transaction at /cookie_sync
type CookieSyncObject struct {
	Type     RequestType
	Status   int
	Error    []error
	Request  string
	Response string
}

func (a *Analytics) Setup() {
	if !a.Enabled {
		return
	}
	if a.EnabledLoggers == nil {
		a.EnabledLoggers = make([]*Module, 0)
	}
	if len(a.File.FileName) > 0 {
		a.addToEnabledLoggers(a.File.setupFileLogger())
	}
	if len(a.Graphite.Host) > 0 {
		a.addToEnabledLoggers(a.Graphite.setupGraphiteLogger())
	}
	if len(a.EnabledLoggers) == 0 {
		a.Enabled = false
	}
}

func (a *Analytics) addToEnabledLoggers(m Module) {
	a.EnabledLoggers = append(a.EnabledLoggers, &m)
}

func (a *Analytics) LogAuctionObject(ao *AuctionObject) {
	for _, logger := range a.EnabledLoggers {
		(*logger).logAuctionObject(ao)
	}
}

func (a *Analytics) LogSetUIDObject(so *SetUIDObject) {
	for _, logger := range a.EnabledLoggers {
		(*logger).logSetUIDObject(so)
	}
}

func (a *Analytics) LogCookieSyncObject(cso *CookieSyncObject) {
	for _, logger := range a.EnabledLoggers {
		(*logger).logCookieSyncObject(cso)
	}
}

func (ao *AuctionObject) log() (content []byte) {
	var err error
	if content, err = json.Marshal(ao); err != nil {
		log.Printf("Transactional Logs Error: Auction object badly formed %v", err)
	}
	return
}
