package stored_requests

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang/glog"
)

// NewHTTPEvents makes an EventProducer which creates events by pinging an external HTTP API
// for updates periodically.
//
// It expects the following endpoint to exist remotely:
//
// GET {endpoint}
//   -- Returns all the known Stored Requests and Stored Imps.
// GET {endpoint}?last-modified={timestamp}
//   -- Returns the Stored Requests and Stored Imps which have been updated since the last timestamp.
//      This timestamp will be sent in the rfc3339 format, using UTC and no timezone shift.
//      For more info, see: https://tools.ietf.org/html/rfc3339
//
// The responses should be JSON like this:
//
// {
//   "requests": {
//     "request1": { ... stored request data ... },
//     "request2": { ... stored request data ... },
//     "request3": { ... stored request data ... },
//   },
//   "imps": {
//     "imp1": { ... stored data for imp1 ... },
//     "imp2": { ... stored data for imp2 ... },
//   }
// }
//
// To signal deletions, the endpoint may return { "deleted": true }
// in place of the Stored Data if the "last-modified" param existed.
//
func NewHTTPEvents(client *http.Client, endpoint string, refreshRate time.Duration) {
	e := &httpEvents{
		client:        client,
		endpoint:      endpoint,
		updates:       make(chan map[string]json.RawMessage, 50),
		invalidations: make(chan []string),
	}
	timestamp := time.Now()
	e.fetchAll()

	go e.refresh(timestamp, time.Tick(refreshRate))
}

type httpEvents struct {
	client        *http.Client
	endpoint      string
	updates       chan map[string]json.RawMessage
	invalidations chan []string
}

func (e *httpEvents) Updates() <-chan map[string]json.RawMessage {
	return e.updates
}

func (e *httpEvents) Invalidations() <-chan []string {
	return e.invalidations
}

func (e *httpEvents) fetchAll() {
	resp, err := e.client.Get(e.endpoint)
	if err != nil {
		glog.Errorf("Failed call: GET %s for Stored Requests: %v", e.endpoint, err)
		return
	}
}

func (e *httpEvents) refresh(lastRefresh time.Time, ticker <-chan time.Time) {
	for {
		select {
		case _ = <-ticker:

		}
	}
}

type responseContract struct {
	StoredRequests map[string]json.RawMessage `json:"requests"`
	StoredImps     map[string]json.RawMessage `json:"imps"`
}
