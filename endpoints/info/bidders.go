package info

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
)

var invalidEnabledOnly = []byte(`Invalid value for 'enabledonly' query param, must be of boolean type`)

// NewBiddersEndpoint builds a handler for the /info/bidders endpoint.
func NewBiddersEndpoint(bidders config.BidderInfos, aliases map[string]string) httprouter.Handle {
	responseAll, err := prepareBiddersResponseAll(bidders, aliases)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint all bidders response: %v", err)
	}

	responseEnabledOnly, err := prepareBiddersResponseEnabledOnly(bidders, aliases)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint enabled only response: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var writeErr error
		switch readEnabledOnly(r) {
		case "true":
			w.Header().Set("Content-Type", "application/json")
			_, writeErr = w.Write(responseEnabledOnly)
		case "false":
			w.Header().Set("Content-Type", "application/json")
			_, writeErr = w.Write(responseAll)
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, writeErr = w.Write(invalidEnabledOnly)
		}

		if writeErr != nil {
			glog.Errorf("error writing response to /info/bidders: %v", writeErr)
		}
	}
}

func readEnabledOnly(r *http.Request) string {
	q := r.URL.Query()

	v, exists := q["enabledonly"]

	if !exists || len(v) == 0 {
		// if the enabledOnly query parameter is not specified, default to false to match
		// previous behavior of returning all adapters regardless of their enabled status.
		return "false"
	}

	return strings.ToLower(v[0])
}

func prepareBiddersResponseAll(bidders config.BidderInfos, aliases map[string]string) ([]byte, error) {
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

func prepareBiddersResponseEnabledOnly(bidders config.BidderInfos, aliases map[string]string) ([]byte, error) {
	bidderNames := make([]string, 0, len(bidders)+len(aliases))

	for name, info := range bidders {
		if info.IsEnabled() {
			bidderNames = append(bidderNames, name)
		}
	}

	for name, bidder := range aliases {
		if info, ok := bidders[bidder]; ok && info.IsEnabled() {
			bidderNames = append(bidderNames, name)
		}
	}

	sort.Strings(bidderNames)

	return json.Marshal(bidderNames)
}
