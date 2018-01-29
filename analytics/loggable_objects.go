package analytics

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"net/http"
)

type RequestType string

const (
	COOKIE_SYNC = "cookie_sync"
	AUCTION     = "auction"
	SETUID      = "set_uid"
)

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

func (ao AuctionObject) String() string {
	if content, err := json.Marshal(ao); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Auction object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (cso CookieSyncObject) String() string {
	if content, err := json.Marshal(cso); err != nil {
		return fmt.Sprintf("Transactional Logs Error: CookieSync object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (so SetUIDObject) String() string {
	if content, err := json.Marshal(so); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Set UID object badly formed %v", err)
	} else {
		return string(content)
	}
}
