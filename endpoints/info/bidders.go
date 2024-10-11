package info

import (
	"net/http"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

var invalidEnabledOnlyMsg = []byte(`Invalid value for 'enabledonly' query param, must be of boolean type`)
var invalidBaseAdaptersOnlyMsg = []byte(`Invalid value for 'baseadaptersonly' query param, must be of boolean type`)

// NewBiddersEndpoint builds a handler for the /info/bidders endpoint.
func NewBiddersEndpoint(bidders config.BidderInfos, aliases map[string]string) httprouter.Handle {
	responseAll, err := prepareBiddersResponseAll(bidders, aliases)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint all bidders response: %v", err)
	}

	responseAllBaseOnly, err := prepareBiddersResponseAllBaseOnly(bidders)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint all bidders (base adapters only) response: %v", err)
	}

	responseEnabledOnly, err := prepareBiddersResponseEnabledOnly(bidders, aliases)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint enabled only response: %v", err)
	}

	responseEnabledOnlyBaseOnly, err := prepareBiddersResponseEnabledOnlyBaseOnly(bidders)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint enabled only (base adapters only) response: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enabledOnly, baseAdaptersOnly, errMsg := readQueryFlags(r)
		if errMsg != nil {
			writeBadRequest(w, errMsg)
			return
		}

		var response []byte
		switch {
		case !enabledOnly && !baseAdaptersOnly:
			response = responseAll
		case !enabledOnly && baseAdaptersOnly:
			response = responseAllBaseOnly
		case enabledOnly && !baseAdaptersOnly:
			response = responseEnabledOnly
		case enabledOnly && baseAdaptersOnly:
			response = responseEnabledOnlyBaseOnly
		}
		writeResponse(w, response)
	}
}

func readQueryFlags(r *http.Request) (enabledOnly, baseAdaptersOnly bool, errMsg []byte) {
	enabledOnly, ok := readQueryFlag(r, "enabledonly")
	if !ok {
		return false, false, invalidEnabledOnlyMsg
	}

	baseAdapterOnly, ok := readQueryFlag(r, "baseadaptersonly")
	if !ok {
		return false, false, invalidBaseAdaptersOnlyMsg
	}

	return enabledOnly, baseAdapterOnly, nil
}

func readQueryFlag(r *http.Request, queryParam string) (flag, ok bool) {
	q := r.URL.Query()

	v, exists := q[queryParam]

	if !exists || len(v) == 0 {
		return false, true
	}

	switch strings.ToLower(v[0]) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
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
	return jsonutil.Marshal(bidderNames)
}

func prepareBiddersResponseAllBaseOnly(bidders config.BidderInfos) ([]byte, error) {
	bidderNames := make([]string, 0, len(bidders))

	for name, info := range bidders {
		if len(info.AliasOf) == 0 {
			bidderNames = append(bidderNames, name)
		}
	}

	sort.Strings(bidderNames)
	return jsonutil.Marshal(bidderNames)
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
	return jsonutil.Marshal(bidderNames)
}

func prepareBiddersResponseEnabledOnlyBaseOnly(bidders config.BidderInfos) ([]byte, error) {
	bidderNames := make([]string, 0, len(bidders))

	for name, info := range bidders {
		if info.IsEnabled() && len(info.AliasOf) == 0 {
			bidderNames = append(bidderNames, name)
		}
	}

	sort.Strings(bidderNames)
	return jsonutil.Marshal(bidderNames)
}

func writeBadRequest(w http.ResponseWriter, data []byte) {
	w.WriteHeader(http.StatusBadRequest)
	writeWithErrorHandling(w, data)
}

func writeResponse(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	writeWithErrorHandling(w, data)
}

func writeWithErrorHandling(w http.ResponseWriter, data []byte) {
	if _, err := w.Write(data); err != nil {
		glog.Errorf("error writing response to /info/bidders: %v", err)
	}
}
