package info

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// NewBiddersEndpoint implements /info/bidders
func NewBiddersEndpoint() httprouter.Handle {
	bidderNames := make([]string, 0, len(openrtb_ext.BidderMap))
	for bidderName := range openrtb_ext.BidderMap {
		bidderNames = append(bidderNames, bidderName)
	}

	jsonData, err := json.Marshal(bidderNames)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint response: %v", err)
	}

	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(jsonData); err != nil {
			glog.Errorf("error writing response to /info/bidders: %v", err)
		}
	})
}

// TODO: Implement this
func NewBidderDetailsEndpoint() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		w.Write([]byte(ps.ByName("bidderName")))
	})
}
