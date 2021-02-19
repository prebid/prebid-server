package info

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
)

// NewBiddersEndpoint builds a handler for the /info/bidders endpoint.
func NewBiddersEndpoint(bidders config.BidderInfos, aliases map[string]string) httprouter.Handle {
	response, err := prepareBiddersResponse(bidders, aliases)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint response: %v", err)
	}

	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(response); err != nil {
			glog.Errorf("error writing response to /info/bidders: %v", err)
		}
	}
}

func prepareBiddersResponse(bidders config.BidderInfos, aliases map[string]string) ([]byte, error) {
	bidderNames := make([]string, 0, len(bidders)+len(aliases))

	for name := range bidders {
		bidderNames = append(bidderNames, name)
	}

	for name := range aliases {
		bidderNames = append(bidderNames, name)
	}

	sort.Strings(bidderNames)

	return json.Marshal(bidderNames)
}
