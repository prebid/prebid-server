package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests"
	"net/http"
	"strings"
	"time"
)

type AmpResponse struct {
	targeting map[string]string `json:"targeting"`
}

// We need to modify the OpenRTB endpoint to handle AMP requests. This will basically modify the parsing
// of the request, and the return value, using the OpenRTB machinery to handle everything inbetween.
func NewAmpEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsById stored_requests.Fetcher, cfg *config.Configuration, met *pbsmetrics.Metrics) (httprouter.Handle, error) {
	if ex == nil || validator == nil || requestsById == nil || cfg == nil || met == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}

	return httprouter.Handle((&endpointDeps{ex, validator, requestsById, cfg, met}).AmpAuction), nil
}

func (deps *endpointDeps) AmpAuction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()

	deps.metrics.RequestMeter.Mark(1)
	// Set this as an AMP request in Metrics. Should it also count as an OpenRTB request?
	deps.metrics.AmpRequestMeter.Mark(1)

	req, errL := deps.parseRequest(r)
	isSafari := false
	if ua := user_agent.New(r.Header.Get("User-Agent")); ua != nil {
		name, _ := ua.Browser()
		if name == "Safari" {
			isSafari = true
			deps.metrics.SafariRequestMeter.Mark(1)
		}
	}

	if len(errL) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errL {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		deps.metrics.ErrorMeter.Mark(1)
		return
	}

	ctx := context.Background()
	cancel := func() {}
	if req.TMax > 0 {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(req.TMax)*time.Millisecond))
	} else {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(5000)*time.Millisecond))
	}
	defer cancel()

	usersyncs := pbs.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie.OptOutCookie))
	if req.App != nil {
		deps.metrics.AppRequestMeter.Mark(1)
	} else if usersyncs.LiveSyncCount() == 0 {
		deps.metrics.NoCookieMeter.Mark(1)
		if isSafari {
			deps.metrics.SafariNoCookieMeter.Mark(1)
		}
	}

	response, err := deps.ex.HoldAuction(ctx, req, usersyncs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/amp Critical error: %v", err)
		return
	}

	// Need to extract the targeting parameters from the response, as those are all that
	// go in the AMP response
	targets := map[string]string{}
	for _, seatBids := range response.SeatBid {
		for _, bid := range seatBids.Bid {
			if strings.Contains(string(bid.Ext), "\"hb_cache_id") {
				// Looking for cache_id to be set, as this should only be set on winning bids (or
				// deal bids), and AMP can only deliver cached ads in any case.
				// Note, this could casue issues if a targeting key value starts with "hb_cache_id",
				// but this is a very unlikely corner case. Doing this so we can catch "hb_cache_id"
				// and "hb_cache_id_{deal}", which allows for deal support in AMP.
				bidExt := &openrtb_ext.ExtBid{}
				err := json.Unmarshal(bid.Ext, bidExt)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "Critical error while unpacking AMP targets: %v", err)
					glog.Errorf("/openrtb2/amp Critical error unpacking targets: %v", err)
					return
				}
				for key, value := range bidExt.Prebid.Targeting {
					targets[key] = value
				}
			}
		}
	}

	// Now JSONify the tragets for the AMP response.
	ampResponse := &AmpResponse{
		targeting: targets,
	}

	// Fixes #231
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	// If an error happens when encoding the response, there isn't much we can do.
	// If we've sent _any_ bytes, then Go would have sent the 200 status code first.
	// That status code can't be un-sent... so the best we can do is log the error.
	if err := enc.Encode(ampResponse); err != nil {
		glog.Errorf("/openrtb2/amp Error encoding response: %v", err)
	}
}

// parseRequest turns the HTTP request into an OpenRTB request.
// If the errors list is empty, then the returned request will be valid according to the OpenRTB 2.5 spec.
// In case of "strong recommendations" in the spec, it tends to be restrictive. If a better workaround is
// possible, it will return errors with messages that suggest improvements.
//
// If the errors list has at least one element, then no guarantees are made about the returned request.
func (deps *endpointDeps) parseAmpRequest(httpRequest *http.Request) (req *openrtb.BidRequest, errs []error) {
	req = &openrtb.BidRequest{}
	errs = nil

	ampId := httpRequest.FormValue("config")
	if len(ampId) == 0 {
		errs = []error{errors.New("AMP requests require an AMP config ID")}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(50)*time.Millisecond)
	defer cancel()

	storedRequests, errs := deps.storedReqFetcher.FetchRequests(ctx, []string{ampId})
	if len(errs) > 0 {
		return nil, errs
	}
	if len(storedRequests) == 0 {
		errs = []error{fmt.Errorf("No AMP config found for ID '%s'", ampId)}
		return
	}
	// An extra check just to be safe.
	if len(storedRequests) > 1 {
		errs = []error{fmt.Errorf("Error fetching AMP config for ID '%s', multiple configs found.", ampId)}
		return
	}
	// The fetched config becomes the entire OpenRTB request
	requestJson := storedRequests[ampId]
	if err := json.Unmarshal(requestJson, req); err != nil {
		errs = []error{err}
		return
	}

	// Two checks so users know which way the Imp check failed.
	if len(req.Imp) == 0 {
		errs = []error{fmt.Errorf("AMP config ID '%s' does not include an Imp object. One id required.", ampId)}
		return
	}
	if len(req.Imp) > 1 {
		errs = []error{fmt.Errorf("AMP config ID '%s' includes multiple Imp objects. We must have only one.", ampId)}
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(httpRequest, req)

	// Need to ensure cache and targeting are turned on
	extRequest := &openrtb_ext.ExtRequest{}
	if err := json.Unmarshal(req.Ext, extRequest); err != nil {
		errs = []error{err}
		return
	}

	updateExt := false
	// Ensure Targeting is on, if not set an empty (default) targeting object
	if extRequest.Prebid.Targeting == nil {
		extRequest.Prebid.Targeting = &openrtb_ext.ExtRequestTargeting{}
		updateExt = true
	}

	// Ensure caching is on, if not set an empty (default) cache object.
	if extRequest.Prebid.Cache == nil {
		extRequest.Prebid.Cache = &openrtb_ext.ExtRequestPrebidCache{}
		updateExt = true
	}
	if extRequest.Prebid.Cache.Bids == nil {
		extRequest.Prebid.Cache.Bids = &openrtb_ext.ExtRequestPrebidCacheBids{}
		updateExt = true
	}

	// Repack the Request.Ext if needed
	if updateExt {
		var err error
		req.Ext, err = json.Marshal(extRequest)
		if err != nil {
			errs = []error{err}
			return
		}
	}

	// At this point, we should have a valid request that definitely has Targeting and Cache turned on

	if err := deps.validateRequest(req); err != nil {
		errs = []error{err}
		return
	}
	return
}
