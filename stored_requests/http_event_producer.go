package stored_requests

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/buger/jsonparser"

	"github.com/golang/glog"
)

// NewHTTPEvents makes an EventProducer which creates events by pinging an external HTTP API
// for updates periodically. If refreshRate is negative, then the data will never be refreshed.
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
func NewHTTPEvents(client *http.Client, endpoint string, refreshRate time.Duration) *httpEvents {
	e := &httpEvents{
		client:        client,
		endpoint:      endpoint,
		lastUpdate:    time.Now().UTC(),
		updates:       make(chan Update, 1),
		invalidations: make(chan Invalidation, 1),
	}
	e.fetchAll()

	go e.refresh(time.Tick(refreshRate))
	return e
}

type httpEvents struct {
	client        *http.Client
	lastUpdate    time.Time
	endpoint      string
	updates       chan Update
	invalidations chan Invalidation
}

func (e *httpEvents) Updates() <-chan Update {
	return e.updates
}

func (e *httpEvents) Invalidations() <-chan Invalidation {
	return e.invalidations
}

func (e *httpEvents) fetchAll() {
	resp, err := e.client.Get(e.endpoint)
	if respObj, ok := e.parse(e.endpoint, resp, err); ok {
		if len(respObj.StoredRequests) > 0 || len(respObj.StoredImps) > 0 {
			e.updates <- Update{
				Requests: respObj.StoredRequests,
				Imps:     respObj.StoredImps,
			}
		}
	}
}

func (e *httpEvents) refresh(ticker <-chan time.Time) {
	for {
		select {
		case thisTime := <-ticker:
			thisTimeInUTC := thisTime.UTC()
			thisEndpoint := e.endpoint + "?last-modified=" + e.lastUpdate.Format(time.RFC3339)
			resp, err := e.client.Get(thisEndpoint)
			if respObj, ok := e.parse(thisEndpoint, resp, err); ok {
				invalidations := Invalidation{
					Requests: extractInvalidations(respObj.StoredRequests),
					Imps:     extractInvalidations(respObj.StoredImps),
				}
				if len(respObj.StoredRequests) > 0 || len(respObj.StoredImps) > 0 {
					e.updates <- Update{
						Requests: respObj.StoredRequests,
						Imps:     respObj.StoredImps,
					}
				}
				if len(invalidations.Requests) > 0 || len(invalidations.Imps) > 0 {
					e.invalidations <- invalidations
				}
				e.lastUpdate = thisTimeInUTC
			}
		}
	}
}

// proceess unpacks the HTTP response and sends the relevant events to the channels.
// It returns true if everything was successful, and false if any errors occurred.
func (e *httpEvents) parse(endpoint string, resp *http.Response, err error) (*responseContract, bool) {
	if err != nil {
		glog.Errorf("Failed call: GET %s for Stored Requests: %v", endpoint, err)
		return nil, false
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Failed to read body of GET %s for Stored Requests: %v", endpoint, err)
		return nil, false
	}

	var respObj responseContract
	if err := json.Unmarshal(respBytes, &respObj); err != nil {
		glog.Errorf("Failed to unmarshal body of GET %s for Stored Requests: %v", endpoint, err)
		return nil, false
	}

	return &respObj, true
}

func extractInvalidations(changes map[string]json.RawMessage) []string {
	deletedIDs := make([]string, 0, len(changes))
	for id, msg := range changes {
		if value, _, _, err := jsonparser.Get(msg, "deleted"); err == nil && bytes.Equal(value, []byte("true")) {
			delete(changes, id)
			deletedIDs = append(deletedIDs, id)
		}
	}
	return deletedIDs
}

type responseContract struct {
	StoredRequests map[string]json.RawMessage `json:"requests"`
	StoredImps     map[string]json.RawMessage `json:"imps"`
}

type Update struct {
	Requests map[string]json.RawMessage `json:"requests"`
	Imps     map[string]json.RawMessage `json:"imps"`
}

type Invalidation struct {
	Requests []string `json:"requests"`
	Imps     []string `json:"imps"`
}
