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
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/util/uuidutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	accountService "github.com/prebid/prebid-server/v3/account"
	"github.com/prebid/prebid-server/v3/amp"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/stored_responses"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/iputil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/version"
)

const defaultAmpRequestTimeoutMillis = 900

var nilBody []byte = nil

type AmpResponse struct {
	Targeting map[string]string `json:"targeting"`
	ORTB2     ORTB2             `json:"ortb2"`
}

type ORTB2 struct {
	Ext openrtb_ext.ExtBidResponse `json:"ext"`
}

// NewAmpEndpoint modifies the OpenRTB endpoint to handle AMP requests. This will basically modify the parsing
// of the request, and the return value, using the OpenRTB machinery to handle everything in between.
func NewAmpEndpoint(
	uuidGenerator uuidutil.UUIDGenerator,
	ex exchange.Exchange,
	requestValidator ortb.RequestValidator,
	requestsById stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	cfg *config.Configuration,
	metricsEngine metrics.MetricsEngine,
	analyticsRunner analytics.Runner,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName,
	storedRespFetcher stored_requests.Fetcher,
	hookExecutionPlanBuilder hooks.ExecutionPlanBuilder,
	tmaxAdjustments *exchange.TmaxAdjustmentsPreprocessed,
) (httprouter.Handle, error) {

	if ex == nil || requestValidator == nil || requestsById == nil || accounts == nil || cfg == nil || metricsEngine == nil {
		return nil, errors.New("NewAmpEndpoint requires non-nil arguments.")
	}

	defRequest := len(defReqJSON) > 0

	ipValidator := iputil.PublicNetworkIPValidator{
		IPv4PrivateNetworks: cfg.RequestValidation.IPv4PrivateNetworksParsed,
		IPv6PrivateNetworks: cfg.RequestValidation.IPv6PrivateNetworksParsed,
	}

	return httprouter.Handle((&endpointDeps{
		uuidGenerator,
		ex,
		requestValidator,
		requestsById,
		empty_fetcher.EmptyFetcher{},
		accounts,
		cfg,
		metricsEngine,
		analyticsRunner,
		disabledBidders,
		defRequest,
		defReqJSON,
		bidderMap,
		nil,
		nil,
		ipValidator,
		storedRespFetcher,
		hookExecutionPlanBuilder,
		tmaxAdjustments,
		openrtb_ext.NormalizeBidderName,
	}).AmpAuction), nil

}

func (deps *endpointDeps) AmpAuction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()

	hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAmp, deps.metricsEngine)

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
	activityControl := privacy.ActivityControl{}

	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogAmpObject(&ao, activityControl)
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
	w.Header().Set("X-Prebid", version.BuildXPrebidHeader(version.Ver))
	setBrowsingTopicsHeader(w, r)

	// There is no body for AMP requests, so we pass a nil body and ignore the return value.
	_, rejectErr := hookExecutor.ExecuteEntrypointStage(r, nilBody)
	reqWrapper, storedAuctionResponses, storedBidResponses, bidderImpReplaceImp, errL := deps.parseAmpRequest(r)
	ao.Errors = append(ao.Errors, errL...)
	// Process reject after parsing amp request, so we can use reqWrapper.
	// There is no body for AMP requests, so we pass a nil body and ignore the return value.
	if rejectErr != nil {
		labels, ao = rejectAmpRequest(*rejectErr, w, hookExecutor, reqWrapper, nil, labels, ao, nil)
		return
	}

	if errortypes.ContainsFatalError(errL) {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errortypes.FatalOnly(errL) {
			fmt.Fprintf(w, "Invalid request: %s\n", err.Error())
		}
		labels.RequestStatus = metrics.RequestStatusBadInput
		return
	}

	ao.RequestWrapper = reqWrapper

	ctx := context.Background()
	var cancel context.CancelFunc
	if reqWrapper.TMax > 0 {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(reqWrapper.TMax)*time.Millisecond))
	} else {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(defaultAmpRequestTimeoutMillis)*time.Millisecond))
	}
	defer cancel()

	// Read UserSyncs/Cookie from Request
	usersyncs := usersync.ReadCookie(r, usersync.Base64Decoder{}, &deps.cfg.HostCookie)
	usersync.SyncHostCookie(r, usersyncs, &deps.cfg.HostCookie)
	if usersyncs.HasAnyLiveSyncs() {
		labels.CookieFlag = metrics.CookieFlagYes
	} else {
		labels.CookieFlag = metrics.CookieFlagNo
	}

	labels.PubID = getAccountID(reqWrapper.Site.Publisher)
	// Look up account now that we have resolved the pubID value
	account, acctIDErrs := accountService.GetAccount(ctx, deps.cfg, deps.accounts, labels.PubID, deps.metricsEngine)
	if len(acctIDErrs) > 0 {
		// best attempt to rebuild the request for analytics. we're already in an error state, so ignoring a
		// potential error from this call
		reqWrapper.RebuildRequest()

		errL = append(errL, acctIDErrs...)
		httpStatus := http.StatusBadRequest
		metricsStatus := metrics.RequestStatusBadInput
		for _, er := range errL {
			errCode := errortypes.ReadCode(er)
			if errCode == errortypes.BlockedAppErrorCode || errCode == errortypes.AccountDisabledErrorCode {
				httpStatus = http.StatusServiceUnavailable
				metricsStatus = metrics.RequestStatusBlockedApp
				break
			}
			if errCode == errortypes.MalformedAcctErrorCode {
				httpStatus = http.StatusInternalServerError
				metricsStatus = metrics.RequestStatusAccountConfigErr
				break
			}
		}
		w.WriteHeader(httpStatus)
		labels.RequestStatus = metricsStatus
		for _, err := range errortypes.FatalOnly(errL) {
			fmt.Fprintf(w, "Invalid request: %s\n", err.Error())
		}
		ao.Errors = append(ao.Errors, acctIDErrs...)
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	if errs := deps.setFieldsImplicitly(r, reqWrapper, account); len(errs) > 0 {
		errL = append(errL, errs...)
	}

	hasStoredAuctionResponses := len(storedAuctionResponses) > 0
	errs := deps.validateRequest(account, r, reqWrapper, true, hasStoredAuctionResponses, storedBidResponses, false)
	errL = append(errL, errs...)
	ao.Errors = append(ao.Errors, errs...)
	if errortypes.ContainsFatalError(errs) {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errortypes.FatalOnly(errs) {
			fmt.Fprintf(w, "Invalid request: %s\n", err.Error())
		}
		labels.RequestStatus = metrics.RequestStatusBadInput
		return
	}

	tcf2Config := gdpr.NewTCF2Config(deps.cfg.GDPR.TCF2, account.GDPR)

	activityControl = privacy.NewActivityControl(&account.Privacy)

	hookExecutor.SetActivityControl(activityControl)
	hookExecutor.SetAccount(account)

	secGPC := r.Header.Get("Sec-GPC")

	auctionRequest := &exchange.AuctionRequest{
		BidRequestWrapper:          reqWrapper,
		Account:                    *account,
		UserSyncs:                  usersyncs,
		RequestType:                labels.RType,
		StartTime:                  start,
		LegacyLabels:               labels,
		GlobalPrivacyControlHeader: secGPC,
		StoredAuctionResponses:     storedAuctionResponses,
		StoredBidResponses:         storedBidResponses,
		BidderImpReplaceImpID:      bidderImpReplaceImp,
		PubID:                      labels.PubID,
		HookExecutor:               hookExecutor,
		QueryParams:                r.URL.Query(),
		TCF2Config:                 tcf2Config,
		Activities:                 activityControl,
		TmaxAdjustments:            deps.tmaxAdjustments,
	}

	auctionResponse, err := deps.ex.HoldAuction(ctx, auctionRequest, nil)
	defer func() {
		if !auctionRequest.BidderResponseStartTime.IsZero() {
			deps.metricsEngine.RecordOverheadTime(metrics.MakeAuctionResponse, time.Since(auctionRequest.BidderResponseStartTime))
		}
	}()
	var response *openrtb2.BidResponse
	if auctionResponse != nil {
		response = auctionResponse.BidResponse
	}
	ao.SeatNonBid = auctionResponse.GetSeatNonBid()
	ao.AuctionResponse = response
	rejectErr, isRejectErr := hookexecution.CastRejectErr(err)
	if err != nil && !isRejectErr {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/amp Critical error: %v", err)
		ao.Status = http.StatusInternalServerError
		ao.Errors = append(ao.Errors, err)
		return
	}

	// hold auction rebuilds the request wrapper first thing, so there is likely
	// no work to do here, but added a rebuild just in case this behavior changes.
	if err := reqWrapper.RebuildRequest(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/amp Critical error: %v", err)
		ao.Status = http.StatusInternalServerError
		ao.Errors = append(ao.Errors, err)
		return
	}

	if isRejectErr {
		labels, ao = rejectAmpRequest(*rejectErr, w, hookExecutor, reqWrapper, account, labels, ao, errL)
		return
	}

	labels, ao = sendAmpResponse(w, hookExecutor, auctionResponse, reqWrapper, account, labels, ao, errL)
}

func rejectAmpRequest(
	rejectErr hookexecution.RejectError,
	w http.ResponseWriter,
	hookExecutor hookexecution.HookStageExecutor,
	reqWrapper *openrtb_ext.RequestWrapper,
	account *config.Account,
	labels metrics.Labels,
	ao analytics.AmpObject,
	errs []error,
) (metrics.Labels, analytics.AmpObject) {
	response := &openrtb2.BidResponse{NBR: openrtb3.NoBidReason(rejectErr.NBR).Ptr()}
	ao.AuctionResponse = response
	ao.Errors = append(ao.Errors, rejectErr)

	return sendAmpResponse(w, hookExecutor, &exchange.AuctionResponse{BidResponse: response}, reqWrapper, account, labels, ao, errs)
}

func sendAmpResponse(
	w http.ResponseWriter,
	hookExecutor hookexecution.HookStageExecutor,
	auctionResponse *exchange.AuctionResponse,
	reqWrapper *openrtb_ext.RequestWrapper,
	account *config.Account,
	labels metrics.Labels,
	ao analytics.AmpObject,
	errs []error,
) (metrics.Labels, analytics.AmpObject) {
	var response *openrtb2.BidResponse
	if auctionResponse != nil {
		response = auctionResponse.BidResponse
	}
	hookExecutor.ExecuteAuctionResponseStage(response)
	// Need to extract the targeting parameters from the response, as those are all that
	// go in the AMP response
	targets := map[string]string{}
	byteCache := []byte("\"hb_cache_id")
	if response != nil {
		for _, seatBids := range response.SeatBid {
			for _, bid := range seatBids.Bid {
				if bytes.Contains(bid.Ext, byteCache) {
					// Looking for cache_id to be set, as this should only be set on winning bids (or
					// deal bids), and AMP can only deliver cached ads in any case.
					// Note, this could cause issues if a targeting key value starts with "hb_cache_id",
					// but this is a very unlikely corner case. Doing this so we can catch "hb_cache_id"
					// and "hb_cache_id_{deal}", which allows for deal support in AMP.
					bidExt := &openrtb_ext.ExtBid{}
					err := jsonutil.Unmarshal(bid.Ext, bidExt)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintf(w, "Critical error while unpacking AMP targets: %v", err)
						glog.Errorf("/openrtb2/amp Critical error unpacking targets: %v", err)
						ao.Errors = append(ao.Errors, fmt.Errorf("Critical error while unpacking AMP targets: %v", err))
						ao.Status = http.StatusInternalServerError
						return labels, ao
					}
					for key, value := range bidExt.Prebid.Targeting {
						targets[key] = value
					}
				}
			}
		}
	}

	// Extract global targeting
	var extResponse openrtb_ext.ExtBidResponse
	eRErr := jsonutil.Unmarshal(response.Ext, &extResponse)
	if eRErr != nil {
		ao.Errors = append(ao.Errors, fmt.Errorf("AMP response: failed to unpack OpenRTB response.ext, debug info cannot be forwarded: %v", eRErr))
	}
	// Extract global targeting
	extPrebid := extResponse.Prebid
	if extPrebid != nil {
		for key, value := range extPrebid.Targeting {
			_, exists := targets[key]
			if !exists {
				targets[key] = value
			}
		}
	}
	// Now JSONify the targets for the AMP response.
	ampResponse := AmpResponse{Targeting: targets}
	ao, ampResponse.ORTB2.Ext = getExtBidResponse(hookExecutor, auctionResponse, reqWrapper, account, ao, errs)

	ao.AmpTargetingValues = targets

	// Fixes #231
	enc := json.NewEncoder(w) // nosemgrep: json-encoder-needs-type
	enc.SetEscapeHTML(false)
	// Explicitly set content type to text/plain, which had previously been
	// the implied behavior from the time the project was launched.
	// It's unclear why text/plain was chosen or if it was an oversight,
	// nevertheless we will keep it as such for compatibility reasons.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// If an error happens when encoding the response, there isn't much we can do.
	// If we've sent _any_ bytes, then Go would have sent the 200 status code first.
	// That status code can't be un-sent... so the best we can do is log the error.
	if err := enc.Encode(ampResponse); err != nil {
		labels.RequestStatus = metrics.RequestStatusNetworkErr
		ao.Errors = append(ao.Errors, fmt.Errorf("/openrtb2/amp Failed to send response: %v", err))
	}

	return labels, ao
}

func getExtBidResponse(
	hookExecutor hookexecution.HookStageExecutor,
	auctionResponse *exchange.AuctionResponse,
	reqWrapper *openrtb_ext.RequestWrapper,
	account *config.Account,
	ao analytics.AmpObject,
	errs []error,
) (analytics.AmpObject, openrtb_ext.ExtBidResponse) {
	var response *openrtb2.BidResponse
	if auctionResponse != nil {
		response = auctionResponse.BidResponse
	}
	// Extract any errors
	var extResponse openrtb_ext.ExtBidResponse
	eRErr := jsonutil.Unmarshal(response.Ext, &extResponse)
	if eRErr != nil {
		ao.Errors = append(ao.Errors, fmt.Errorf("AMP response: failed to unpack OpenRTB response.ext, debug info cannot be forwarded: %v", eRErr))
	}

	warnings := extResponse.Warnings
	if warnings == nil {
		warnings = make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)
	}
	for _, v := range errortypes.WarningOnly(errs) {
		if errortypes.ReadScope(v) == errortypes.ScopeDebug && !(reqWrapper != nil && reqWrapper.Test == 1) {
			continue
		}
		bidderErr := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.ReadCode(v),
			Message: v.Error(),
		}
		warnings[openrtb_ext.BidderReservedGeneral] = append(warnings[openrtb_ext.BidderReservedGeneral], bidderErr)
	}

	extBidResponse := openrtb_ext.ExtBidResponse{
		Errors:   extResponse.Errors,
		Warnings: warnings,
	}

	// add debug information if requested
	if reqWrapper != nil {
		if reqWrapper.Test == 1 && eRErr == nil {
			if extResponse.Debug != nil {
				extBidResponse.Debug = extResponse.Debug
			} else {
				glog.Errorf("Test set on request but debug not present in response.")
				ao.Errors = append(ao.Errors, fmt.Errorf("test set on request but debug not present in response"))
			}
		}

		stageOutcomes := hookExecutor.GetOutcomes()
		ao.HookExecutionOutcome = stageOutcomes
		modules, warns, err := hookexecution.GetModulesJSON(stageOutcomes, reqWrapper.BidRequest, account)
		if err != nil {
			err := fmt.Errorf("Failed to get modules outcome: %s", err)
			glog.Errorf(err.Error())
			ao.Errors = append(ao.Errors, err)
		} else if modules != nil {
			extBidResponse.Prebid = &openrtb_ext.ExtResponsePrebid{Modules: modules}
		}

		if len(warns) > 0 {
			ao.Errors = append(ao.Errors, warns...)
		}
	}

	setSeatNonBid(&extBidResponse, reqWrapper, auctionResponse)

	return ao, extBidResponse
}

// parseRequest turns the HTTP request into an OpenRTB request.
// If the errors list is empty, then the returned request will be valid according to the OpenRTB 2.5 spec.
// In case of "strong recommendations" in the spec, it tends to be restrictive. If a better workaround is
// possible, it will return errors with messages that suggest improvements.
//
// If the errors list has at least one element, then no guarantees are made about the returned request.
func (deps *endpointDeps) parseAmpRequest(httpRequest *http.Request) (req *openrtb_ext.RequestWrapper, storedAuctionResponses stored_responses.ImpsWithBidResponses, storedBidResponses stored_responses.ImpBidderStoredResp, bidderImpReplaceImp stored_responses.BidderImpReplaceImpID, errs []error) {
	// Load the stored request for the AMP ID.
	reqNormal, storedAuctionResponses, storedBidResponses, bidderImpReplaceImp, e := deps.loadRequestJSONForAmp(httpRequest)
	if errs = append(errs, e...); errortypes.ContainsFatalError(errs) {
		return
	}

	// move to using the request wrapper
	req = &openrtb_ext.RequestWrapper{BidRequest: reqNormal}

	// normalize to openrtb 2.6
	if err := openrtb_ext.ConvertUpTo26(req); err != nil {
		errs = append(errs, err)
	}
	if errortypes.ContainsFatalError(errs) {
		return
	}

	// Need to ensure cache and targeting are turned on
	e = initAmpTargetingAndCache(req)
	if errs = append(errs, e...); errortypes.ContainsFatalError(errs) {
		return
	}

	if err := ortb.SetDefaults(req, deps.cfg.TmaxDefault); err != nil {
		errs = append(errs, err)
		return
	}

	return
}

// Load the stored OpenRTB request for an incoming AMP request, or return the errors found.
func (deps *endpointDeps) loadRequestJSONForAmp(httpRequest *http.Request) (req *openrtb2.BidRequest, storedAuctionResponses stored_responses.ImpsWithBidResponses, storedBidResponses stored_responses.ImpBidderStoredResp, bidderImpReplaceImp stored_responses.BidderImpReplaceImpID, errs []error) {
	req = &openrtb2.BidRequest{}
	errs = nil

	ampParams, err := amp.ParseParams(httpRequest)
	if err != nil {
		return nil, nil, nil, nil, []error{err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deps.cfg.StoredRequestsTimeout)*time.Millisecond)
	defer cancel()

	storedRequests, _, errs := deps.storedReqFetcher.FetchRequests(ctx, []string{ampParams.StoredRequestID}, nil)
	if len(errs) > 0 {
		return nil, nil, nil, nil, errs
	}
	if len(storedRequests) == 0 {
		errs = []error{fmt.Errorf("No AMP config found for tag_id '%s'", ampParams.StoredRequestID)}
		return
	}

	// The fetched config becomes the entire OpenRTB request
	requestJSON := storedRequests[ampParams.StoredRequestID]
	if err := jsonutil.UnmarshalValid(requestJSON, req); err != nil {
		errs = []error{err}
		return
	}

	storedAuctionResponses, storedBidResponses, bidderImpReplaceImp, errs = stored_responses.ProcessStoredResponses(ctx, &openrtb_ext.RequestWrapper{BidRequest: req}, deps.storedRespFetcher)
	if err != nil {
		errs = []error{err}
		return
	}

	if deps.cfg.GenerateRequestID || req.ID == "{{UUID}}" {
		newBidRequestId, err := deps.uuidGenerator.Generate()
		if err != nil {
			errs = []error{err}
			return
		}
		req.ID = newBidRequestId
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

	setAmpExtDirect(req.Site, "1")

	setEffectiveAmpPubID(req, ampParams.Account)

	if ampParams.Slot != "" {
		req.Imp[0].TagID = ampParams.Slot
	}

	if err := setConsentedProviders(req, ampParams); err != nil {
		return []error{err}
	}

	policyWriter, policyWriterErr := amp.ReadPolicy(ampParams, deps.cfg.GDPR.Enabled)
	if policyWriterErr != nil {
		return []error{policyWriterErr}
	}
	if err := policyWriter.Write(req); err != nil {
		return []error{err}
	}

	if ampParams.Timeout != nil {
		req.TMax = int64(*ampParams.Timeout) - deps.cfg.AMPTimeoutAdjustment
	}

	var errors []error
	if warn := setTargeting(req, ampParams.Targeting); warn != nil {
		errors = append(errors, warn)
	}

	if err := setTrace(req, ampParams.Trace); err != nil {
		return append(errors, err)
	}

	return errors
}

// setConsentedProviders sets the addtl_consent value to user.ext.ConsentedProvidersSettings.consented_providers
// in its orginal Google Additional Consent string format and user.ext.consented_providers_settings.consented_providers
// that is an array of ints that contains the elements found in addtl_consent
func setConsentedProviders(req *openrtb2.BidRequest, ampParams amp.Params) error {
	if len(ampParams.AdditionalConsent) > 0 {
		reqWrap := &openrtb_ext.RequestWrapper{BidRequest: req}

		userExt, err := reqWrap.GetUserExt()
		if err != nil {
			return err
		}

		// Parse addtl_consent, that is supposed to come formatted as a Google Additional Consent string, into array of ints
		consentedProvidersList := openrtb_ext.ParseConsentedProvidersString(ampParams.AdditionalConsent)

		// Set user.ext.consented_providers_settings.consented_providers if elements where found
		if len(consentedProvidersList) > 0 {
			cps := userExt.GetConsentedProvidersSettingsOut()
			if cps == nil {
				cps = &openrtb_ext.ConsentedProvidersSettingsOut{}
			}
			cps.ConsentedProvidersList = append(cps.ConsentedProvidersList, consentedProvidersList...)
			userExt.SetConsentedProvidersSettingsOut(cps)
		}

		// Copy addtl_consent into user.ext.ConsentedProvidersSettings.consented_providers as is
		cps := userExt.GetConsentedProvidersSettingsIn()
		if cps == nil {
			cps = &openrtb_ext.ConsentedProvidersSettingsIn{}
		}
		cps.ConsentedProvidersString = ampParams.AdditionalConsent
		userExt.SetConsentedProvidersSettingsIn(cps)

		if err := reqWrap.RebuildRequest(); err != nil {
			return err
		}
	}
	return nil
}

// setTargeting merges "targeting" to imp[0].ext.data
func setTargeting(req *openrtb2.BidRequest, targeting string) error {
	if len(targeting) == 0 {
		return nil
	}

	targetingData := exchange.WrapJSONInData([]byte(targeting))

	if len(req.Imp[0].Ext) > 0 {
		newImpExt, err := jsonpatch.MergePatch(req.Imp[0].Ext, targetingData)
		if err != nil {
			warn := errortypes.Warning{
				WarningCode: errortypes.BadInputErrorCode,
				Message:     fmt.Sprintf("unable to merge imp.ext with targeting data, check targeting data is correct: %s", err.Error()),
			}

			return &warn
		}
		req.Imp[0].Ext = newImpExt
		return nil
	}

	req.Imp[0].Ext = targetingData
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
func initAmpTargetingAndCache(req *openrtb_ext.RequestWrapper) []error {
	extRequest, err := req.GetRequestExt()
	if err != nil {
		return []error{err}
	}

	prebid := extRequest.GetPrebid()
	prebidModified := false

	// create prebid object if missing
	if prebid == nil {
		prebid = &openrtb_ext.ExtRequestPrebid{}
	}

	// create targeting object if missing
	if prebid.Targeting == nil {
		prebid.Targeting = &openrtb_ext.ExtRequestTargeting{}
		prebidModified = true
	}

	// create cache object if missing
	if prebid.Cache == nil {
		prebid.Cache = &openrtb_ext.ExtRequestPrebidCache{}
		prebidModified = true
	}
	if prebid.Cache.Bids == nil {
		prebid.Cache.Bids = &openrtb_ext.ExtRequestPrebidCacheBids{}
		prebidModified = true
	}

	if prebidModified {
		extRequest.SetPrebid(prebid)
	}
	return nil
}

func setAmpExtDirect(site *openrtb2.Site, value string) {
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

// Sets the effective publisher ID for amp request
func setEffectiveAmpPubID(req *openrtb2.BidRequest, account string) {
	// ACCOUNT_ID is the unresolved macro name and should be ignored.
	if account == "" || account == "ACCOUNT_ID" {
		return
	}

	var pub *openrtb2.Publisher
	if req.App != nil {
		if req.App.Publisher == nil {
			req.App.Publisher = &openrtb2.Publisher{}
		}
		pub = req.App.Publisher
	} else if req.Site != nil {
		if req.Site.Publisher == nil {
			req.Site.Publisher = &openrtb2.Publisher{}
		}
		pub = req.Site.Publisher
	}

	if pub.ID == "" {
		pub.ID = account
	}
}

func setTrace(req *openrtb2.BidRequest, value string) error {
	if value == "" {
		return nil
	}

	ext, err := jsonutil.Marshal(map[string]map[string]string{"prebid": {"trace": value}})
	if err != nil {
		return err
	}

	if len(req.Ext) > 0 {
		ext, err = jsonpatch.MergePatch(req.Ext, ext)
		if err != nil {
			return err
		}
	}
	req.Ext = ext

	return nil
}

// setSeatNonBid populates bidresponse.ext.prebid.seatnonbid if bidrequest.ext.prebid.returnallbidstatus is true
func setSeatNonBid(finalExtBidResponse *openrtb_ext.ExtBidResponse, request *openrtb_ext.RequestWrapper, auctionResponse *exchange.AuctionResponse) bool {
	if finalExtBidResponse == nil || auctionResponse == nil || request == nil {
		return false
	}
	reqExt, err := request.GetRequestExt()
	if err != nil {
		return false
	}
	prebid := reqExt.GetPrebid()
	if prebid == nil || !prebid.ReturnAllBidStatus {
		return false
	}
	if finalExtBidResponse.Prebid == nil {
		finalExtBidResponse.Prebid = &openrtb_ext.ExtResponsePrebid{}
	}
	finalExtBidResponse.Prebid.SeatNonBid = auctionResponse.GetSeatNonBid()
	return true
}
