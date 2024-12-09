package info

import (
	"net/http"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

var invalidEnabledOnlyMsg = []byte(`Invalid value for 'enabledonly' query param, must be of boolean type`)
var invalidBaseAdaptersOnlyMsg = []byte(`Invalid value for 'baseadaptersonly' query param, must be of boolean type`)

// NewBiddersEndpoint builds a handler for the /info/bidders endpoint.
func NewBiddersEndpoint(bidders config.BidderInfos) httprouter.Handle {
	responseAll, err := prepareBiddersResponseAll(bidders)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint all bidders response: %v", err)
	}

	responseAllBaseOnly, err := prepareBiddersResponseAllBaseOnly(bidders)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint all bidders (base adapters only) response: %v", err)
	}

	responseEnabledOnly, err := prepareBiddersResponseEnabledOnly(bidders)
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

type bidderPredicate func(config.BidderInfo) bool

func prepareResponse(bidders config.BidderInfos, p bidderPredicate) ([]byte, error) {
	bidderNames := make([]string, 0, len(bidders))

	for name, info := range bidders {
		if p(info) {
			bidderNames = append(bidderNames, name)
		}
	}

	sort.Strings(bidderNames)
	return jsonutil.Marshal(bidderNames)
}

func prepareBiddersResponseAll(bidders config.BidderInfos) ([]byte, error) {
	filterNone := func(_ config.BidderInfo) bool { return true }
	return prepareResponse(bidders, filterNone)
}

func prepareBiddersResponseAllBaseOnly(bidders config.BidderInfos) ([]byte, error) {
	filterBaseOnly := func(info config.BidderInfo) bool { return len(info.AliasOf) == 0 }
	return prepareResponse(bidders, filterBaseOnly)
}

func prepareBiddersResponseEnabledOnly(bidders config.BidderInfos) ([]byte, error) {
	filterEnabledOnly := func(info config.BidderInfo) bool { return info.IsEnabled() }
	return prepareResponse(bidders, filterEnabledOnly)
}

func prepareBiddersResponseEnabledOnlyBaseOnly(bidders config.BidderInfos) ([]byte, error) {
	filterEnabledOnlyBaseOnly := func(info config.BidderInfo) bool { return info.IsEnabled() && len(info.AliasOf) == 0 }
	return prepareResponse(bidders, filterEnabledOnlyBaseOnly)
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
