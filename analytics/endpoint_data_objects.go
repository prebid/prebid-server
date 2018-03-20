package analytics

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"time"
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
	Type      RequestType
	Status    int
	Error     []error
	Request   *openrtb.BidRequest
	Response  *openrtb.BidResponse
	UserAgent string
}

//Loggable object of a transaction at /openrtb2/amp endpoint
type AmpObject struct {
	Type            RequestType
	Status          int
	Error           []error
	Request         *openrtb.BidRequest
	AuctionResponse *openrtb.BidResponse
	AmpResponse     map[string]string
	UserAgent       string
	Origin          string
}

//Loggable object of a transaction at /setuid
type SetUIDObject struct {
	Type     RequestType
	Status   int
	Bidder   string
	UID      string
	Usersync bool
	Expiry   time.Duration
	Error    []error
	Success  bool
}

//Loggable object of a transaction at /cookie_sync
type CookieSyncObject struct {
	Type    RequestType
	Status  int
	Error   []error
	Bidders string
}

func (ao AuctionObject) ToJson() string {
	if content, err := json.Marshal(ao); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Auction object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (cso CookieSyncObject) ToJson() string {
	if content, err := json.Marshal(cso); err != nil {
		return fmt.Sprintf("Transactional Logs Error: CookieSync object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (so SetUIDObject) ToJson() string {
	if content, err := json.Marshal(so); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Set UID object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (ao AmpObject) ToJson() string {
	if content, err := json.Marshal(ao); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Amp object badly formed %v", err)
	} else {
		return string(content)
	}
}
