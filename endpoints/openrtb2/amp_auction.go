package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/usersync"
)

const defaultAmpRequestTimeoutMillis = 900

type AmpResponse struct {
	Targeting map[string]string                                       `json:"targeting"`
	Debug     *openrtb_ext.ExtResponseDebug                           `json:"debug,omitempty"`
	Errors    map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderError `json:"errors,omitempty"`
}

// NewAmpEndpoint modifies the OpenRTB endpoint to handle AMP requests. This will basically modify the parsing
// of the request, and the return value, using the OpenRTB machinery to handle everything in between.
func NewAmpEndpoint(
	ex exchange.Exchange,
	validator openrtb_ext.BidderParamValidator,
	requestsById stored_requests.Fetcher,
	categories stored_requests.CategoryFetcher,
	cfg *config.Configuration,
	met pbsmetrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName,
) (httprouter.Handle, error) {

	if ex == nil || validator == nil || requestsById == nil || cfg == nil || met == nil {
		return nil, errors.New("NewAmpEndpoint requires non-nil arguments.")
	}

	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	return httprouter.Handle((&endpointDeps{
		ex,
		validator,
		requestsById,
		empty_fetcher.EmptyFetcher{},
		categories,
		cfg,
		met,
		pbsAnalytics,
		disabledBidders,
		defRequest,
		defReqJSON,
		bidderMap,
		nil}).AmpAuction), nil

}

func (deps *endpointDeps) AmpAuction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	ao := analytics.AmpObject{
		Status: http.StatusOK,
		Errors: make([]error, 0),
	}

	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.

	// Set this as an AMP request in Metrics.

	start := time.Now()
	labels := pbsmetrics.Labels{
		Source:        pbsmetrics.DemandWeb,
		RType:         pbsmetrics.ReqTypeAMP,
		PubID:         pbsmetrics.PublisherUnknown,
		Browser:       getBrowserName(r),
		CookieFlag:    pbsmetrics.CookieFlagUnknown,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}
	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogAmpObject(&ao)
	}()

	// Add AMP headers
	origin := r.FormValue("__amp_source_origin")
	if len(origin) == 0 {
		// Just to be safe
		origin = r.Header.Get("Origin")
		ao.Origin = origin
	}

	// Headers "Access-Control-Allow-Origin", "Access-Control-Allow-Headers",
	// and "Access-Control-Allow-Credentials" are handled in CORS middleware
	w.Header().Set("AMP-Access-Control-Allow-Source-Origin", origin)
	w.Header().Set("Access-Control-Expose-Headers", "AMP-Access-Control-Allow-Source-Origin")

	req, errL := deps.parseAmpRequest(r)

	if fatalError(errL) {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errL {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		ao.Errors = append(ao.Errors, errL...)
		labels.RequestStatus = pbsmetrics.RequestStatusBadInput
		return
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if req.TMax > 0 {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(req.TMax)*time.Millisecond))
	} else {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(defaultAmpRequestTimeoutMillis)*time.Millisecond))
	}
	defer cancel()

	usersyncs := usersync.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie))
	if usersyncs.LiveSyncCount() == 0 {
		labels.CookieFlag = pbsmetrics.CookieFlagNo
	} else {
		labels.CookieFlag = pbsmetrics.CookieFlagYes
	}
	labels.PubID = effectivePubID(req.Site.Publisher)
	// Blacklist account now that we have resolved the value
	if acctIdErr := validateAccount(deps.cfg, labels.PubID); acctIdErr != nil {
		errL = append(errL, acctIdErr)
		erVal := errortypes.DecodeError(acctIdErr)
		if erVal == errortypes.BlacklistedAppCode || erVal == errortypes.BlacklistedAcctCode {
			w.WriteHeader(http.StatusServiceUnavailable)
			labels.RequestStatus = pbsmetrics.RequestStatusBlacklisted
		} else { //erVal == errortypes.AcctRequiredCode
			w.WriteHeader(http.StatusBadRequest)
			labels.RequestStatus = pbsmetrics.RequestStatusBadInput
		}
		for _, err := range errL {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		ao.Errors = append(ao.Errors, errL...)
		return
	}

	response, err := deps.ex.HoldAuction(ctx, req, usersyncs, labels, &deps.categories, nil)
	ao.AuctionResponse = response

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/amp Critical error: %v", err)
		ao.Status = http.StatusInternalServerError
		ao.Errors = append(ao.Errors, err)
		return
	}

	// Need to extract the targeting parameters from the response, as those are all that
	// go in the AMP response
	targets := map[string]string{}
	byteCache := []byte("\"hb_cache_id")
	for _, seatBids := range response.SeatBid {
		for _, bid := range seatBids.Bid {
			if bytes.Contains(bid.Ext, byteCache) {
				// Looking for cache_id to be set, as this should only be set on winning bids (or
				// deal bids), and AMP can only deliver cached ads in any case.
				// Note, this could cause issues if a targeting key value starts with "hb_cache_id",
				// but this is a very unlikely corner case. Doing this so we can catch "hb_cache_id"
				// and "hb_cache_id_{deal}", which allows for deal support in AMP.
				bidExt := &openrtb_ext.ExtBid{}
				err := json.Unmarshal(bid.Ext, bidExt)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "Critical error while unpacking AMP targets: %v", err)
					glog.Errorf("/openrtb2/amp Critical error unpacking targets: %v", err)
					ao.Errors = append(ao.Errors, fmt.Errorf("Critical error while unpacking AMP targets: %v", err))
					ao.Status = http.StatusInternalServerError
					return
				}
				for key, value := range bidExt.Prebid.Targeting {
					targets[key] = value
				}
			}
		}
	}
	// Extract any errors
	var extResponse openrtb_ext.ExtBidResponse
	eRErr := json.Unmarshal(response.Ext, &extResponse)
	if eRErr != nil {
		ao.Errors = append(ao.Errors, fmt.Errorf("AMP response: failed to unpack OpenRTB response.ext, debug info cannot be forwarded: %v", eRErr))
	}

	// Now JSONify the targets for the AMP response.
	ampResponse := AmpResponse{
		Targeting: targets,
		Errors:    extResponse.Errors,
	}

	ao.AmpTargetingValues = targets

	// add debug information if requested
	if req.Test == 1 && eRErr == nil {
		if extResponse.Debug != nil {
			ampResponse.Debug = extResponse.Debug
		} else {
			glog.Errorf("Test set on request but debug not present in response: %v", err)
			ao.Errors = append(ao.Errors, fmt.Errorf("Test set on request but debug not present in response: %v", err))
		}
	}

	// Fixes #231
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	// If an error happens when encoding the response, there isn't much we can do.
	// If we've sent _any_ bytes, then Go would have sent the 200 status code first.
	// That status code can't be un-sent... so the best we can do is log the error.
	if err := enc.Encode(ampResponse); err != nil {
		labels.RequestStatus = pbsmetrics.RequestStatusNetworkErr
		ao.Errors = append(ao.Errors, fmt.Errorf("/openrtb2/amp Failed to send response: %v", err))
	}
}

// parseRequest turns the HTTP request into an OpenRTB request.
// If the errors list is empty, then the returned request will be valid according to the OpenRTB 2.5 spec.
// In case of "strong recommendations" in the spec, it tends to be restrictive. If a better workaround is
// possible, it will return errors with messages that suggest improvements.
//
// If the errors list has at least one element, then no guarantees are made about the returned request.
func (deps *endpointDeps) parseAmpRequest(httpRequest *http.Request) (req *openrtb.BidRequest, errs []error) {
	// Load the stored request for the AMP ID.
	req, errs = deps.loadRequestJSONForAmp(httpRequest)
	if len(errs) > 0 {
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(httpRequest, req)

	// Need to ensure cache and targeting are turned on
	errs = defaultRequestExt(req)
	if len(errs) > 0 {
		return
	}

	// At this point, we should have a valid request that definitely has Targeting and Cache turned on

	errL := deps.validateRequest(req)
	if len(errL) > 0 {
		errs = append(errs, errL...)
	}

	return
}

// Load the stored OpenRTB request for an incoming AMP request, or return the errors found.
func (deps *endpointDeps) loadRequestJSONForAmp(httpRequest *http.Request) (req *openrtb.BidRequest, errs []error) {
	req = &openrtb.BidRequest{}
	errs = nil

	ampID := httpRequest.FormValue("tag_id")
	if ampID == "" {
		errs = []error{errors.New("AMP requests require an AMP tag_id")}
		return
	}

	debugParam := httpRequest.FormValue("debug")
	debug := debugParam == "1"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(storedRequestTimeoutMillis)*time.Millisecond)
	defer cancel()

	storedRequests, _, errs := deps.storedReqFetcher.FetchRequests(ctx, []string{ampID}, nil)
	if len(errs) > 0 {
		return nil, errs
	}
	if len(storedRequests) == 0 {
		errs = []error{fmt.Errorf("No AMP config found for tag_id '%s'", ampID)}
		return
	}

	// The fetched config becomes the entire OpenRTB request
	requestJSON := storedRequests[ampID]
	if err := json.Unmarshal(requestJSON, req); err != nil {
		errs = []error{err}
		return
	}

	if debug {
		req.Test = 1
	}

	// Two checks so users know which way the Imp check failed.
	if len(req.Imp) == 0 {
		errs = []error{fmt.Errorf("data for tag_id='%s' does not define the required imp array", ampID)}
		return
	}
	if len(req.Imp) > 1 {
		errs = []error{fmt.Errorf("data for tag_id '%s' includes %d imp elements. Only one is allowed", ampID, len(req.Imp))}
		return
	}

	if req.App != nil {
		errs = []error{errors.New("request.app must not exist in AMP stored requests.")}
		return
	}

	// Force HTTPS as AMP requires it, but pubs can forget to set it.
	if req.Imp[0].Secure == nil {
		secure := int8(1)
		req.Imp[0].Secure = &secure
	} else {
		*req.Imp[0].Secure = 1
	}

	err := deps.overrideWithParams(httpRequest, req)
	if err != nil {
		errs = []error{err}
	}

	return
}

func (deps *endpointDeps) overrideWithParams(httpRequest *http.Request, req *openrtb.BidRequest) error {
	if req.Site == nil {
		req.Site = &openrtb.Site{}
	}
	// Override the stored request sizes with AMP ones, if they exist.
	if req.Imp[0].Banner != nil {
		width := parseFormInt(httpRequest, "w", 0)
		height := parseFormInt(httpRequest, "h", 0)
		overrideWidth := parseFormInt(httpRequest, "ow", 0)
		overrideHeight := parseFormInt(httpRequest, "oh", 0)
		if format := makeFormatReplacement(overrideWidth, overrideHeight, width, height, httpRequest.FormValue("ms")); len(format) != 0 {
			req.Imp[0].Banner.Format = format
		} else if width != 0 {
			setWidths(req.Imp[0].Banner.Format, width)
		} else if height != 0 {
			setHeights(req.Imp[0].Banner.Format, height)
		}
	}

	canonicalURL := httpRequest.FormValue("curl")
	if canonicalURL != "" {
		req.Site.Page = canonicalURL
		// Fixes #683
		if parsedURL, err := url.Parse(canonicalURL); err == nil {
			domain := parsedURL.Host
			if colonIndex := strings.LastIndex(domain, ":"); colonIndex != -1 {
				domain = domain[:colonIndex]
			}
			req.Site.Domain = domain
		}
	}

	setAmpExt(req.Site, "1")

	slot := httpRequest.FormValue("slot")
	if slot != "" {
		req.Imp[0].TagID = slot
	}

	privacyPolicies := privacy.Policies{
		GDPR: gdpr.Policy{
			Consent: httpRequest.URL.Query().Get("gdpr_consent"),
		},
		CCPA: ccpa.Policy{
			Value: httpRequest.URL.Query().Get("us_privacy"),
		},
	}
	if err := privacyPolicies.Write(req); err != nil {
		return err
	}

	if timeout, err := strconv.ParseInt(httpRequest.FormValue("timeout"), 10, 64); err == nil {
		req.TMax = timeout - deps.cfg.AMPTimeoutAdjustment
	}

	return nil
}

func makeFormatReplacement(overrideWidth uint64, overrideHeight uint64, width uint64, height uint64, multisize string) []openrtb.Format {
	if overrideWidth != 0 && overrideHeight != 0 {
		return []openrtb.Format{{
			W: overrideWidth,
			H: overrideHeight,
		}}
	} else if overrideWidth != 0 && height != 0 {
		return []openrtb.Format{{
			W: overrideWidth,
			H: height,
		}}
	} else if width != 0 && overrideHeight != 0 {
		return []openrtb.Format{{
			W: width,
			H: overrideHeight,
		}}
	} else if parsedSizes := parseMultisize(multisize); len(parsedSizes) != 0 {
		return parsedSizes
	} else if width != 0 && height != 0 {
		return []openrtb.Format{{
			W: width,
			H: height,
		}}
	}

	return nil
}

func setWidths(formats []openrtb.Format, width uint64) {
	for i := 0; i < len(formats); i++ {
		formats[i].W = width
	}
}

func setHeights(formats []openrtb.Format, height uint64) {
	for i := 0; i < len(formats); i++ {
		formats[i].H = height
	}
}

func parseMultisize(multisize string) []openrtb.Format {
	if multisize == "" {
		return nil
	}

	sizeStrings := strings.Split(multisize, ",")
	sizes := make([]openrtb.Format, 0, len(sizeStrings))
	for _, sizeString := range sizeStrings {
		wh := strings.Split(sizeString, "x")
		if len(wh) != 2 {
			return nil
		}
		f := openrtb.Format{
			W: parseIntErrorless(wh[0], 0),
			H: parseIntErrorless(wh[1], 0),
		}
		if f.W == 0 && f.H == 0 {
			return nil
		}

		sizes = append(sizes, f)
	}
	return sizes
}

func parseFormInt(req *http.Request, value string, defaultTo uint64) uint64 {
	return parseIntErrorless(req.FormValue(value), defaultTo)
}

func parseIntErrorless(value string, defaultTo uint64) uint64 {
	if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
		return parsed
	}
	return defaultTo
}

// AMP won't function unless ext.prebid.targeting and ext.prebid.cache.bids are defined.
// If the user didn't include them, default those here.
func defaultRequestExt(req *openrtb.BidRequest) (errs []error) {
	errs = nil
	extRequest := &openrtb_ext.ExtRequest{}
	if req.Ext != nil && len(req.Ext) > 0 {
		if err := json.Unmarshal(req.Ext, extRequest); err != nil {
			errs = []error{err}
			return
		}
	}

	setDefaults := false
	// Ensure Targeting and caching is on
	if extRequest.Prebid.Targeting == nil {
		setDefaults = true
		extRequest.Prebid.Targeting = &openrtb_ext.ExtRequestTargeting{
			// Fixes #452
			IncludeWinners:    true,
			IncludeBidderKeys: true,
			PriceGranularity:  openrtb_ext.PriceGranularityFromString("med"),
		}
	}
	if extRequest.Prebid.Cache == nil {
		setDefaults = true
		extRequest.Prebid.Cache = &openrtb_ext.ExtRequestPrebidCache{
			Bids: &openrtb_ext.ExtRequestPrebidCacheBids{},
		}
	} else if extRequest.Prebid.Cache.Bids == nil {
		setDefaults = true
		extRequest.Prebid.Cache.Bids = &openrtb_ext.ExtRequestPrebidCacheBids{}
	}
	if setDefaults {
		newExt, err := json.Marshal(extRequest)
		if err == nil {
			req.Ext = newExt
		} else {
			errs = []error{err}
		}
	}

	return
}

func setAmpExt(site *openrtb.Site, value string) {
	if len(site.Ext) > 0 {
		if _, dataType, _, _ := jsonparser.Get(site.Ext, "amp"); dataType == jsonparser.NotExist {
			if val, err := jsonparser.Set(site.Ext, []byte(value), "amp"); err == nil {
				site.Ext = val
			}
		}
	} else {
		site.Ext = json.RawMessage(`{"amp":` + value + `}`)
	}
}
