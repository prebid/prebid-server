package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/amp"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/iputil"
)

const defaultAmpRequestTimeoutMillis = 900

type AmpResponse struct {
	Targeting map[string]string                                         `json:"targeting"`
	Debug     *openrtb_ext.ExtResponseDebug                             `json:"debug,omitempty"`
	Errors    map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage `json:"errors,omitempty"`
	Warnings  map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage `json:"warnings,omitempty"`
}

// NewAmpEndpoint modifies the OpenRTB endpoint to handle AMP requests. This will basically modify the parsing
// of the request, and the return value, using the OpenRTB machinery to handle everything in between.
func NewAmpEndpoint(
	ex exchange.Exchange,
	validator openrtb_ext.BidderParamValidator,
	requestsById stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	cfg *config.Configuration,
	met metrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName,
) (httprouter.Handle, error) {

	if ex == nil || validator == nil || requestsById == nil || accounts == nil || cfg == nil || met == nil {
		return nil, errors.New("NewAmpEndpoint requires non-nil arguments.")
	}

	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	ipValidator := iputil.PublicNetworkIPValidator{
		IPv4PrivateNetworks: cfg.RequestValidation.IPv4PrivateNetworksParsed,
		IPv6PrivateNetworks: cfg.RequestValidation.IPv6PrivateNetworksParsed,
	}

	return httprouter.Handle((&endpointDeps{
		ex,
		validator,
		requestsById,
		empty_fetcher.EmptyFetcher{},
		accounts,
		cfg,
		met,
		pbsAnalytics,
		disabledBidders,
		defRequest,
		defReqJSON,
		bidderMap,
		nil,
		nil,
		ipValidator}).AmpAuction), nil

}

func (deps *endpointDeps) AmpAuction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()

	ao := analytics.AmpObject{
		Status:    http.StatusOK,
		Errors:    make([]error, 0),
		StartTime: start,
	}

	// Set this as an AMP request in Metrics.

	labels := metrics.Labels{
		Source:        metrics.DemandWeb,
		RType:         metrics.ReqTypeAMP,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
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
	ao.Errors = append(ao.Errors, errL...)

	if errortypes.ContainsFatalError(errL) {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errortypes.FatalOnly(errL) {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		labels.RequestStatus = metrics.RequestStatusBadInput
		return
	}

	ao.Request = req

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
		labels.CookieFlag = metrics.CookieFlagNo
	} else {
		labels.CookieFlag = metrics.CookieFlagYes
	}
	labels.PubID = getAccountID(req.Site.Publisher)
	// Look up account now that we have resolved the pubID value
	account, acctIDErrs := accountService.GetAccount(ctx, deps.cfg, deps.accounts, labels.PubID)
	if len(acctIDErrs) > 0 {
		errL = append(errL, acctIDErrs...)
		httpStatus := http.StatusBadRequest
		metricsStatus := metrics.RequestStatusBadInput
		for _, er := range errL {
			errCode := errortypes.ReadCode(er)
			if errCode == errortypes.BlacklistedAppErrorCode || errCode == errortypes.BlacklistedAcctErrorCode {
				httpStatus = http.StatusServiceUnavailable
				metricsStatus = metrics.RequestStatusBlacklisted
				break
			}
		}
		w.WriteHeader(httpStatus)
		labels.RequestStatus = metricsStatus
		for _, err := range errortypes.FatalOnly(errL) {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		ao.Errors = append(ao.Errors, acctIDErrs...)
		return
	}

	secGPC := r.Header.Get("Sec-GPC")

	auctionRequest := exchange.AuctionRequest{
		BidRequest:                 req,
		Account:                    *account,
		UserSyncs:                  usersyncs,
		RequestType:                labels.RType,
		StartTime:                  start,
		LegacyLabels:               labels,
		GlobalPrivacyControlHeader: secGPC,
	}

	response, err := deps.ex.HoldAuction(ctx, auctionRequest, nil)
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

	warnings := extResponse.Warnings
	if warnings == nil {
		warnings = make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)
	}
	for _, v := range errortypes.WarningOnly(errL) {
		bidderErr := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.ReadCode(v),
			Message: v.Error(),
		}
		warnings[openrtb_ext.BidderReservedGeneral] = append(warnings[openrtb_ext.BidderReservedGeneral], bidderErr)
	}

	// Now JSONify the targets for the AMP response.
	ampResponse := AmpResponse{
		Targeting: targets,
		Errors:    extResponse.Errors,
		Warnings:  warnings,
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
		labels.RequestStatus = metrics.RequestStatusNetworkErr
		ao.Errors = append(ao.Errors, fmt.Errorf("/openrtb2/amp Failed to send response: %v", err))
	}
}

// parseRequest turns the HTTP request into an OpenRTB request.
// If the errors list is empty, then the returned request will be valid according to the OpenRTB 2.5 spec.
// In case of "strong recommendations" in the spec, it tends to be restrictive. If a better workaround is
// possible, it will return errors with messages that suggest improvements.
//
// If the errors list has at least one element, then no guarantees are made about the returned request.
func (deps *endpointDeps) parseAmpRequest(httpRequest *http.Request) (req *openrtb2.BidRequest, errs []error) {
	// Load the stored request for the AMP ID.
	req, e := deps.loadRequestJSONForAmp(httpRequest)
	if errs = append(errs, e...); errortypes.ContainsFatalError(errs) {
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(httpRequest, req)

	// Need to ensure cache and targeting are turned on
	e = defaultRequestExt(req)
	if errs = append(errs, e...); errortypes.ContainsFatalError(errs) {
		return
	}

	// At this point, we should have a valid request that definitely has Targeting and Cache turned on

	e = deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: req})
	errs = append(errs, e...)
	return
}

// Load the stored OpenRTB request for an incoming AMP request, or return the errors found.
func (deps *endpointDeps) loadRequestJSONForAmp(httpRequest *http.Request) (req *openrtb2.BidRequest, errs []error) {
	req = &openrtb2.BidRequest{}
	errs = nil

	ampParams, err := amp.ParseParams(httpRequest)
	if err != nil {
		return nil, []error{err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(storedRequestTimeoutMillis)*time.Millisecond)
	defer cancel()

	storedRequests, _, errs := deps.storedReqFetcher.FetchRequests(ctx, []string{ampParams.StoredRequestID}, nil)
	if len(errs) > 0 {
		return nil, errs
	}
	if len(storedRequests) == 0 {
		errs = []error{fmt.Errorf("No AMP config found for tag_id '%s'", ampParams.StoredRequestID)}
		return
	}

	// The fetched config becomes the entire OpenRTB request
	requestJSON := storedRequests[ampParams.StoredRequestID]
	if err := json.Unmarshal(requestJSON, req); err != nil {
		errs = []error{err}
		return
	}

	if ampParams.Debug {
		req.Test = 1
	}

	// Two checks so users know which way the Imp check failed.
	if len(req.Imp) == 0 {
		errs = []error{fmt.Errorf("data for tag_id='%s' does not define the required imp array", ampParams.StoredRequestID)}
		return
	}
	if len(req.Imp) > 1 {
		errs = []error{fmt.Errorf("data for tag_id '%s' includes %d imp elements. Only one is allowed", ampParams.StoredRequestID, len(req.Imp))}
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

	errs = deps.overrideWithParams(ampParams, req)
	return
}

func (deps *endpointDeps) overrideWithParams(ampParams amp.Params, req *openrtb2.BidRequest) []error {
	if req.Site == nil {
		req.Site = &openrtb2.Site{}
	}

	// Override the stored request sizes with AMP ones, if they exist.
	if req.Imp[0].Banner != nil {
		if format := makeFormatReplacement(ampParams.Size); len(format) != 0 {
			req.Imp[0].Banner.Format = format
		} else if ampParams.Size.Width != 0 {
			setWidths(req.Imp[0].Banner.Format, ampParams.Size.Width)
		} else if ampParams.Size.Height != 0 {
			setHeights(req.Imp[0].Banner.Format, ampParams.Size.Height)
		}
	}

	if ampParams.CanonicalURL != "" {
		req.Site.Page = ampParams.CanonicalURL
		// Fixes #683
		if parsedURL, err := url.Parse(ampParams.CanonicalURL); err == nil {
			domain := parsedURL.Host
			if colonIndex := strings.LastIndex(domain, ":"); colonIndex != -1 {
				domain = domain[:colonIndex]
			}
			req.Site.Domain = domain
		}
	}

	setAmpExt(req.Site, "1")

	setEffectiveAmpPubID(req, ampParams.Account)

	if ampParams.Slot != "" {
		req.Imp[0].TagID = ampParams.Slot
	}

	policyWriter, policyWriterErr := readPolicy(ampParams.Consent)
	if policyWriterErr != nil {
		return []error{policyWriterErr}
	}
	if err := policyWriter.Write(req); err != nil {
		return []error{err}
	}

	if ampParams.Timeout != nil {
		req.TMax = int64(*ampParams.Timeout) - deps.cfg.AMPTimeoutAdjustment
	}

	return nil
}

func makeFormatReplacement(size amp.Size) []openrtb2.Format {
	var formats []openrtb2.Format
	if size.OverrideWidth != 0 && size.OverrideHeight != 0 {
		formats = []openrtb2.Format{{
			W: size.OverrideWidth,
			H: size.OverrideHeight,
		}}
	} else if size.OverrideWidth != 0 && size.Height != 0 {
		formats = []openrtb2.Format{{
			W: size.OverrideWidth,
			H: size.Height,
		}}
	} else if size.Width != 0 && size.OverrideHeight != 0 {
		formats = []openrtb2.Format{{
			W: size.Width,
			H: size.OverrideHeight,
		}}
	} else if size.Width != 0 && size.Height != 0 {
		formats = []openrtb2.Format{{
			W: size.Width,
			H: size.Height,
		}}
	}

	return append(formats, size.Multisize...)
}

func setWidths(formats []openrtb2.Format, width int64) {
	for i := 0; i < len(formats); i++ {
		formats[i].W = width
	}
}

func setHeights(formats []openrtb2.Format, height int64) {
	for i := 0; i < len(formats); i++ {
		formats[i].H = height
	}
}

// AMP won't function unless ext.prebid.targeting and ext.prebid.cache.bids are defined.
// If the user didn't include them, default those here.
func defaultRequestExt(req *openrtb2.BidRequest) (errs []error) {
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

func setAmpExt(site *openrtb2.Site, value string) {
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

func readPolicy(consent string) (privacy.PolicyWriter, error) {
	if len(consent) == 0 {
		return privacy.NilPolicyWriter{}, nil
	}

	if gdpr.ValidateConsent(consent) {
		return gdpr.ConsentWriter{consent}, nil
	}

	if ccpa.ValidateConsent(consent) {
		return ccpa.ConsentWriter{consent}, nil
	}

	return privacy.NilPolicyWriter{}, &errortypes.Warning{
		Message:     fmt.Sprintf("Consent '%s' is not recognized as either CCPA or GDPR TCF.", consent),
		WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
	}
}

// Sets the effective publisher ID for amp request
func setEffectiveAmpPubID(req *openrtb2.BidRequest, account string) {
	var pub *openrtb2.Publisher
	if req.App != nil {
		if req.App.Publisher == nil {
			req.App.Publisher = new(openrtb2.Publisher)
		}
		pub = req.App.Publisher
	} else if req.Site != nil {
		if req.Site.Publisher == nil {
			req.Site.Publisher = new(openrtb2.Publisher)
		}
		pub = req.Site.Publisher
	}

	if pub.ID == "" {
		// ACCOUNT_ID is the unresolved macro name and should be ignored.
		if account != "" && account != "ACCOUNT_ID" {
			pub.ID = account
		}
	}
}
