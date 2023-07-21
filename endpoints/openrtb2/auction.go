package openrtb2

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/privacy"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gofrs/uuid"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	gpplib "github.com/prebid/go-gpp"
	"github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/native1"
	nativeRequests "github.com/prebid/openrtb/v19/native1/request"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/openrtb/v19/openrtb3"
	"github.com/prebid/prebid-server/bidadjustment"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/ortb"
	"golang.org/x/net/publicsuffix"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"

	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/lmt"
	"github.com/prebid/prebid-server/schain"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/stored_responses"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/httputil"
	"github.com/prebid/prebid-server/util/iputil"
	"github.com/prebid/prebid-server/util/uuidutil"
	"github.com/prebid/prebid-server/version"
)

const storedRequestTimeoutMillis = 50
const ampChannel = "amp"
const appChannel = "app"

var (
	dntKey      string = http.CanonicalHeaderKey("DNT")
	dntDisabled int8   = 0
	dntEnabled  int8   = 1
	notAmp      int8   = 0
)

var accountIdSearchPath = [...]struct {
	isApp bool
	key   []string
}{
	{true, []string{"app", "publisher", "ext", openrtb_ext.PrebidExtKey, "parentAccount"}},
	{true, []string{"app", "publisher", "id"}},
	{false, []string{"site", "publisher", "ext", openrtb_ext.PrebidExtKey, "parentAccount"}},
	{false, []string{"site", "publisher", "id"}},
}

func NewEndpoint(
	uuidGenerator uuidutil.UUIDGenerator,
	ex exchange.Exchange,
	validator openrtb_ext.BidderParamValidator,
	requestsById stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	cfg *config.Configuration,
	metricsEngine metrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName,
	storedRespFetcher stored_requests.Fetcher,
	hookExecutionPlanBuilder hooks.ExecutionPlanBuilder,
) (httprouter.Handle, error) {
	if ex == nil || validator == nil || requestsById == nil || accounts == nil || cfg == nil || metricsEngine == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}

	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	ipValidator := iputil.PublicNetworkIPValidator{
		IPv4PrivateNetworks: cfg.RequestValidation.IPv4PrivateNetworksParsed,
		IPv6PrivateNetworks: cfg.RequestValidation.IPv6PrivateNetworksParsed,
	}

	return httprouter.Handle((&endpointDeps{
		uuidGenerator,
		ex,
		validator,
		requestsById,
		empty_fetcher.EmptyFetcher{},
		accounts,
		cfg,
		metricsEngine,
		pbsAnalytics,
		disabledBidders,
		defRequest,
		defReqJSON,
		bidderMap,
		nil,
		nil,
		ipValidator,
		storedRespFetcher,
		hookExecutionPlanBuilder}).Auction), nil
}

type endpointDeps struct {
	uuidGenerator             uuidutil.UUIDGenerator
	ex                        exchange.Exchange
	paramsValidator           openrtb_ext.BidderParamValidator
	storedReqFetcher          stored_requests.Fetcher
	videoFetcher              stored_requests.Fetcher
	accounts                  stored_requests.AccountFetcher
	cfg                       *config.Configuration
	metricsEngine             metrics.MetricsEngine
	analytics                 analytics.PBSAnalyticsModule
	disabledBidders           map[string]string
	defaultRequest            bool
	defReqJSON                []byte
	bidderMap                 map[string]openrtb_ext.BidderName
	cache                     prebid_cache_client.Client
	debugLogRegexp            *regexp.Regexp
	privateNetworkIPValidator iputil.IPValidator
	storedRespFetcher         stored_requests.Fetcher
	hookExecutionPlanBuilder  hooks.ExecutionPlanBuilder
}

func (deps *endpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()

	hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)

	ao := analytics.AuctionObject{
		Status:    http.StatusOK,
		Errors:    make([]error, 0),
		StartTime: start,
	}

	labels := metrics.Labels{
		Source:        metrics.DemandUnknown,
		RType:         metrics.ReqTypeORTB2Web,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
	}
	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogAuctionObject(&ao)
	}()

	w.Header().Set("X-Prebid", version.BuildXPrebidHeader(version.Ver))

	req, impExtInfoMap, storedAuctionResponses, storedBidResponses, bidderImpReplaceImp, account, errL := deps.parseRequest(r, &labels, hookExecutor)
	if errortypes.ContainsFatalError(errL) && writeError(errL, w, &labels) {
		return
	}

	if rejectErr := hookexecution.FindFirstRejectOrNil(errL); rejectErr != nil {
		ao.RequestWrapper = req
		labels, ao = rejectAuctionRequest(*rejectErr, w, hookExecutor, req.BidRequest, account, labels, ao)
		return
	}

	tcf2Config := gdpr.NewTCF2Config(deps.cfg.GDPR.TCF2, account.GDPR)

	activities, activitiesErr := privacy.NewActivityControl(account.Privacy)
	if activitiesErr != nil {
		errL = append(errL, activitiesErr)
		if errortypes.ContainsFatalError(errL) {
			writeError(errL, w, &labels)
			return
		}
	}

	ctx := context.Background()

	timeout := deps.cfg.AuctionTimeouts.LimitAuctionTimeout(time.Duration(req.TMax) * time.Millisecond)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, start.Add(timeout))
		defer cancel()
	}

	// Read Usersyncs/Cookie
	decoder := usersync.Base64Decoder{}
	usersyncs := usersync.ReadCookie(r, decoder, &deps.cfg.HostCookie)
	usersync.SyncHostCookie(r, usersyncs, &deps.cfg.HostCookie)

	if req.Site != nil {
		if usersyncs.HasAnyLiveSyncs() {
			labels.CookieFlag = metrics.CookieFlagYes
		} else {
			labels.CookieFlag = metrics.CookieFlagNo
		}
	}

	// Set Integration Information
	err := deps.setIntegrationType(req, account)
	if err != nil {
		errL = append(errL, err)
		writeError(errL, w, &labels)
		return
	}
	secGPC := r.Header.Get("Sec-GPC")

	warnings := errortypes.WarningOnly(errL)

	auctionRequest := &exchange.AuctionRequest{
		BidRequestWrapper:          req,
		Account:                    *account,
		UserSyncs:                  usersyncs,
		RequestType:                labels.RType,
		StartTime:                  start,
		LegacyLabels:               labels,
		Warnings:                   warnings,
		GlobalPrivacyControlHeader: secGPC,
		ImpExtInfoMap:              impExtInfoMap,
		StoredAuctionResponses:     storedAuctionResponses,
		StoredBidResponses:         storedBidResponses,
		BidderImpReplaceImpID:      bidderImpReplaceImp,
		PubID:                      labels.PubID,
		HookExecutor:               hookExecutor,
		TCF2Config:                 tcf2Config,
		Activities:                 activities,
	}
	auctionResponse, err := deps.ex.HoldAuction(ctx, auctionRequest, nil)
	ao.RequestWrapper = req
	ao.Account = account
	var response *openrtb2.BidResponse
	if auctionResponse != nil {
		response = auctionResponse.BidResponse
	}
	ao.Response = response
	ao.SeatNonBid = auctionResponse.GetSeatNonBid()
	rejectErr, isRejectErr := hookexecution.CastRejectErr(err)
	if err != nil && !isRejectErr {
		if errortypes.ReadCode(err) == errortypes.BadInputErrorCode {
			writeError([]error{err}, w, &labels)
			return
		}
		labels.RequestStatus = metrics.RequestStatusErr
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/auction Critical error: %v", err)
		ao.Status = http.StatusInternalServerError
		ao.Errors = append(ao.Errors, err)
		return
	} else if isRejectErr {
		labels, ao = rejectAuctionRequest(*rejectErr, w, hookExecutor, req.BidRequest, account, labels, ao)
		return
	}

	err = setSeatNonBidRaw(req, auctionResponse)
	if err != nil {
		glog.Errorf("Error setting seat non-bid: %v", err)
	}
	labels, ao = sendAuctionResponse(w, hookExecutor, response, req.BidRequest, account, labels, ao)
}

// setSeatNonBidRaw is transitional function for setting SeatNonBid inside bidResponse.Ext
// Because,
// 1. today exchange.HoldAuction prepares and marshals some piece of response.Ext which is then used by auction.go, amp_auction.go and video_auction.go
// 2. As per discussion with Prebid Team we are planning to move away from - HoldAuction building openrtb2.BidResponse. instead respective auction modules will build this object
// 3. So, we will need this method to do first,  unmarshalling of response.Ext
func setSeatNonBidRaw(request *openrtb_ext.RequestWrapper, auctionResponse *exchange.AuctionResponse) error {
	if auctionResponse == nil || auctionResponse.BidResponse == nil {
		return nil
	}
	// unmarshalling is required here, until we are moving away from bidResponse.Ext, which is populated
	// by HoldAuction
	response := auctionResponse.BidResponse
	respExt := &openrtb_ext.ExtBidResponse{}
	if err := json.Unmarshal(response.Ext, &respExt); err != nil {
		return err
	}
	if setSeatNonBid(respExt, request, auctionResponse) {
		if respExtJson, err := json.Marshal(respExt); err == nil {
			response.Ext = respExtJson
			return nil
		} else {
			return err
		}
	}
	return nil
}

func rejectAuctionRequest(
	rejectErr hookexecution.RejectError,
	w http.ResponseWriter,
	hookExecutor hookexecution.HookStageExecutor,
	request *openrtb2.BidRequest,
	account *config.Account,
	labels metrics.Labels,
	ao analytics.AuctionObject,
) (metrics.Labels, analytics.AuctionObject) {
	response := &openrtb2.BidResponse{NBR: openrtb3.NoBidReason(rejectErr.NBR).Ptr()}
	if request != nil {
		response.ID = request.ID
	}

	ao.Response = response
	ao.Errors = append(ao.Errors, rejectErr)

	return sendAuctionResponse(w, hookExecutor, response, request, account, labels, ao)
}

func sendAuctionResponse(
	w http.ResponseWriter,
	hookExecutor hookexecution.HookStageExecutor,
	response *openrtb2.BidResponse,
	request *openrtb2.BidRequest,
	account *config.Account,
	labels metrics.Labels,
	ao analytics.AuctionObject,
) (metrics.Labels, analytics.AuctionObject) {
	hookExecutor.ExecuteAuctionResponseStage(response)

	if response != nil {
		stageOutcomes := hookExecutor.GetOutcomes()
		ao.HookExecutionOutcome = stageOutcomes

		ext, warns, err := hookexecution.EnrichExtBidResponse(response.Ext, stageOutcomes, request, account)
		if err != nil {
			err = fmt.Errorf("Failed to enrich Bid Response with hook debug information: %s", err)
			glog.Errorf(err.Error())
			ao.Errors = append(ao.Errors, err)
		} else {
			response.Ext = ext
		}

		if len(warns) > 0 {
			ao.Errors = append(ao.Errors, warns...)
		}
	}

	// Fixes #231
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	w.Header().Set("Content-Type", "application/json")

	// If an error happens when encoding the response, there isn't much we can do.
	// If we've sent _any_ bytes, then Go would have sent the 200 status code first.
	// That status code can't be un-sent... so the best we can do is log the error.
	if err := enc.Encode(response); err != nil {
		labels.RequestStatus = metrics.RequestStatusNetworkErr
		ao.Errors = append(ao.Errors, fmt.Errorf("/openrtb2/auction Failed to send response: %v", err))
	}

	return labels, ao
}

// parseRequest turns the HTTP request into an OpenRTB request. This is guaranteed to return:
//
//   - A context which times out appropriately, given the request.
//   - A cancellation function which should be called if the auction finishes early.
//
// If the errors list is empty, then the returned request will be valid according to the OpenRTB 2.5 spec.
// In case of "strong recommendations" in the spec, it tends to be restrictive. If a better workaround is
// possible, it will return errors with messages that suggest improvements.
//
// If the errors list has at least one element, then no guarantees are made about the returned request.
func (deps *endpointDeps) parseRequest(httpRequest *http.Request, labels *metrics.Labels, hookExecutor hookexecution.HookStageExecutor) (req *openrtb_ext.RequestWrapper, impExtInfoMap map[string]exchange.ImpExtInfo, storedAuctionResponses stored_responses.ImpsWithBidResponses, storedBidResponses stored_responses.ImpBidderStoredResp, bidderImpReplaceImpId stored_responses.BidderImpReplaceImpID, account *config.Account, errs []error) {
	errs = nil
	var err error
	var r io.ReadCloser = httpRequest.Body
	reqContentEncoding := httputil.ContentEncoding(httpRequest.Header.Get("Content-Encoding"))
	if reqContentEncoding != "" {
		if !deps.cfg.Compression.Request.IsSupported(reqContentEncoding) {
			errs = []error{fmt.Errorf("Content-Encoding of type %s is not supported", reqContentEncoding)}
			return
		} else {
			r, err = getCompressionEnabledReader(httpRequest.Body, reqContentEncoding)
			if err != nil {
				errs = []error{err}
				return
			}
		}
	}
	defer r.Close()
	limitedReqReader := &io.LimitedReader{
		R: r,
		N: deps.cfg.MaxRequestSize,
	}

	requestJson, err := ioutil.ReadAll(limitedReqReader)
	if err != nil {
		errs = []error{err}
		return
	}

	if limitedReqReader.N <= 0 {
		// Limited Reader returns 0 if the request was exactly at the max size or over the limit.
		// This is because it only reads up to N bytes. To check if the request was too large,
		//  we need to look at the next byte of its underlying reader, limitedReader.R.
		if _, err := limitedReqReader.R.Read(make([]byte, 1)); err != io.EOF {
			// Discard the rest of the request body so that the connection can be reused.
			io.Copy(io.Discard, httpRequest.Body)
			errs = []error{fmt.Errorf("request size exceeded max size of %d bytes.", deps.cfg.MaxRequestSize)}
			return
		}
	}

	req = &openrtb_ext.RequestWrapper{}
	req.BidRequest = &openrtb2.BidRequest{}

	requestJson, rejectErr := hookExecutor.ExecuteEntrypointStage(httpRequest, requestJson)
	if rejectErr != nil {
		errs = []error{rejectErr}
		if err = json.Unmarshal(requestJson, req.BidRequest); err != nil {
			glog.Errorf("Failed to unmarshal BidRequest during entrypoint rejection: %s", err)
		}
		return
	}

	timeout := parseTimeout(requestJson, time.Duration(storedRequestTimeoutMillis)*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	impInfo, errs := parseImpInfo(requestJson)
	if len(errs) > 0 {
		return nil, nil, nil, nil, nil, nil, errs
	}

	storedBidRequestId, hasStoredBidRequest, storedRequests, storedImps, errs := deps.getStoredRequests(ctx, requestJson, impInfo)
	if len(errs) > 0 {
		return
	}

	accountId, isAppReq, errs := getAccountIdFromRawRequest(hasStoredBidRequest, storedRequests[storedBidRequestId], requestJson)
	// fill labels here in order to pass correct metrics in case of errors
	if isAppReq {
		labels.Source = metrics.DemandApp
		labels.RType = metrics.ReqTypeORTB2App
		labels.PubID = accountId
	} else { // is Site request
		labels.Source = metrics.DemandWeb
		labels.PubID = accountId
	}
	if errs != nil {
		return
	}

	// Look up account
	account, errs = accountService.GetAccount(ctx, deps.cfg, deps.accounts, accountId, deps.metricsEngine)
	if len(errs) > 0 {
		return
	}

	hookExecutor.SetAccount(account)
	requestJson, rejectErr = hookExecutor.ExecuteRawAuctionStage(requestJson)
	if rejectErr != nil {
		errs = []error{rejectErr}
		if err = json.Unmarshal(requestJson, req.BidRequest); err != nil {
			glog.Errorf("Failed to unmarshal BidRequest during raw auction stage rejection: %s", err)
		}
		return
	}

	// retrieve storedRequests and storedImps once more in case stored data was changed by the raw auction hook
	if hasPayloadUpdatesAt(hooks.StageRawAuctionRequest.String(), hookExecutor.GetOutcomes()) {
		impInfo, errs = parseImpInfo(requestJson)
		if len(errs) > 0 {
			return nil, nil, nil, nil, nil, nil, errs
		}
		storedBidRequestId, hasStoredBidRequest, storedRequests, storedImps, errs = deps.getStoredRequests(ctx, requestJson, impInfo)
		if len(errs) > 0 {
			return
		}
	}

	// Fetch the Stored Request data and merge it into the HTTP request.
	if requestJson, impExtInfoMap, errs = deps.processStoredRequests(requestJson, impInfo, storedRequests, storedImps, storedBidRequestId, hasStoredBidRequest); len(errs) > 0 {
		return
	}

	//Stored auction responses should be processed after stored requests due to possible impression modification
	storedAuctionResponses, storedBidResponses, bidderImpReplaceImpId, errs = stored_responses.ProcessStoredResponses(ctx, requestJson, deps.storedRespFetcher, deps.bidderMap)
	if len(errs) > 0 {
		return nil, nil, nil, nil, nil, nil, errs
	}

	if err := json.Unmarshal(requestJson, req.BidRequest); err != nil {
		errs = []error{err}
		return
	}

	if err := mergeBidderParams(req); err != nil {
		errs = []error{err}
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(httpRequest, req)

	if err := ortb.SetDefaults(req); err != nil {
		errs = []error{err}
		return
	}

	if err := processInterstitials(req); err != nil {
		errs = []error{err}
		return
	}

	lmt.ModifyForIOS(req.BidRequest)

	hasStoredResponses := len(storedAuctionResponses) > 0
	errL := deps.validateRequest(req, false, hasStoredResponses, storedBidResponses, hasStoredBidRequest)
	if len(errL) > 0 {
		errs = append(errs, errL...)
	}

	return
}

func getCompressionEnabledReader(body io.ReadCloser, contentEncoding httputil.ContentEncoding) (io.ReadCloser, error) {
	switch contentEncoding {
	case httputil.ContentEncodingGZIP:
		return gzip.NewReader(body)
	default:
		return nil, fmt.Errorf("unsupported compression type '%s'", contentEncoding)
	}
}

// hasPayloadUpdatesAt checks if there are any successful payload updates at given stage
func hasPayloadUpdatesAt(stageName string, outcomes []hookexecution.StageOutcome) bool {
	for _, outcome := range outcomes {
		if stageName != outcome.Stage {
			continue
		}

		for _, group := range outcome.Groups {
			for _, invocationResult := range group.InvocationResults {
				if invocationResult.Status == hookexecution.StatusSuccess &&
					invocationResult.Action == hookexecution.ActionUpdate {
					return true
				}
			}
		}
	}

	return false
}

// parseTimeout returns parses tmax from the requestJson, or returns the default if it doesn't exist.
//
// requestJson should be the content of the POST body.
//
// If the request defines tmax explicitly, then this will return that duration in milliseconds.
// If not, it will return the default timeout.
func parseTimeout(requestJson []byte, defaultTimeout time.Duration) time.Duration {
	if tmax, dataType, _, err := jsonparser.Get(requestJson, "tmax"); dataType != jsonparser.NotExist && err == nil {
		if tmaxInt, err := strconv.Atoi(string(tmax)); err == nil && tmaxInt > 0 {
			return time.Duration(tmaxInt) * time.Millisecond
		}
	}
	return defaultTimeout
}

// mergeBidderParams merges bidder parameters in req.ext down to the imp[].ext level, with
// priority given to imp[].ext in case of a conflict. No validation of bidder parameters or
// of the ext json is performed. Unmarshal errors are not expected since the ext json was
// validated during the bid request unmarshal.
func mergeBidderParams(req *openrtb_ext.RequestWrapper) error {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return nil
	}

	prebid := reqExt.GetPrebid()
	if prebid == nil {
		return nil
	}

	bidderParamsJson := prebid.BidderParams
	if len(bidderParamsJson) == 0 {
		return nil
	}

	bidderParams := map[string]map[string]json.RawMessage{}
	if err := json.Unmarshal(bidderParamsJson, &bidderParams); err != nil {
		return nil
	}

	for i, imp := range req.GetImp() {
		impExt, err := imp.GetImpExt()
		if err != nil {
			continue
		}

		// merges bidder parameters passed at req.ext level with imp[].ext.BIDDER level
		if err := mergeBidderParamsImpExt(impExt, bidderParams); err != nil {
			return fmt.Errorf("error processing bidder parameters for imp[%d]: %s", i, err.Error())
		}

		// merges bidder parameters passed at req.ext level with imp[].ext.prebid.bidder.BIDDER level
		if err := mergeBidderParamsImpExtPrebid(impExt, bidderParams); err != nil {
			return fmt.Errorf("error processing bidder parameters for imp[%d]: %s", i, err.Error())
		}
	}

	return nil
}

// mergeBidderParamsImpExt merges bidder parameters in req.ext down to the imp[].ext.BIDDER
// level, giving priority to imp[].ext.BIDDER in case of a conflict. Unmarshal errors are not
// expected since the ext json was validated during the bid request unmarshal.
func mergeBidderParamsImpExt(impExt *openrtb_ext.ImpExt, reqExtParams map[string]map[string]json.RawMessage) error {
	extMap := impExt.GetExt()
	extMapModified := false

	for bidder, params := range reqExtParams {
		if !isPossibleBidder(bidder) {
			continue
		}

		impExtBidder, impExtBidderExists := extMap[bidder]
		if !impExtBidderExists || impExtBidder == nil {
			continue
		}

		impExtBidderMap := map[string]json.RawMessage{}
		if len(impExtBidder) > 0 {
			if err := json.Unmarshal(impExtBidder, &impExtBidderMap); err != nil {
				continue
			}
		}

		modified := false
		for key, value := range params {
			if _, present := impExtBidderMap[key]; !present {
				impExtBidderMap[key] = value
				modified = true
			}
		}

		if modified {
			impExtBidderJson, err := json.Marshal(impExtBidderMap)
			if err != nil {
				return fmt.Errorf("error marshalling ext.BIDDER: %s", err.Error())
			}
			extMap[bidder] = impExtBidderJson
			extMapModified = true
		}
	}

	if extMapModified {
		impExt.SetExt(extMap)
	}

	return nil
}

// mergeBidderParamsImpExtPrebid merges bidder parameters in req.ext down to the imp[].ext.prebid.bidder.BIDDER
// level, giving priority to imp[].ext.prebid.bidder.BIDDER in case of a conflict.
func mergeBidderParamsImpExtPrebid(impExt *openrtb_ext.ImpExt, reqExtParams map[string]map[string]json.RawMessage) error {
	prebid := impExt.GetPrebid()
	prebidModified := false

	if prebid == nil || len(prebid.Bidder) == 0 {
		return nil
	}

	for bidder, params := range reqExtParams {
		impExtPrebidBidder, impExtPrebidBidderExists := prebid.Bidder[bidder]
		if !impExtPrebidBidderExists || impExtPrebidBidder == nil {
			continue
		}

		impExtPrebidBidderMap := map[string]json.RawMessage{}
		if len(impExtPrebidBidder) > 0 {
			if err := json.Unmarshal(impExtPrebidBidder, &impExtPrebidBidderMap); err != nil {
				continue
			}
		}

		modified := false
		for key, value := range params {
			if _, present := impExtPrebidBidderMap[key]; !present {
				impExtPrebidBidderMap[key] = value
				modified = true
			}
		}

		if modified {
			impExtPrebidBidderJson, err := json.Marshal(impExtPrebidBidderMap)
			if err != nil {
				return fmt.Errorf("error marshalling ext.prebid.bidder.BIDDER: %s", err.Error())
			}
			prebid.Bidder[bidder] = impExtPrebidBidderJson
			prebidModified = true
		}
	}

	if prebidModified {
		impExt.SetPrebid(prebid)
	}

	return nil
}

func (deps *endpointDeps) validateRequest(req *openrtb_ext.RequestWrapper, isAmp bool, hasStoredResponses bool, storedBidResp stored_responses.ImpBidderStoredResp, hasStoredBidRequest bool) []error {
	errL := []error{}
	if req.ID == "" {
		return []error{errors.New("request missing required field: \"id\"")}
	}

	if req.TMax < 0 {
		return []error{fmt.Errorf("request.tmax must be nonnegative. Got %d", req.TMax)}
	}

	if req.LenImp() < 1 {
		return []error{errors.New("request.imp must contain at least one element.")}
	}

	if len(req.Cur) > 1 {
		req.Cur = req.Cur[0:1]
		errL = append(errL, &errortypes.Warning{Message: fmt.Sprintf("A prebid request can only process one currency. Taking the first currency in the list, %s, as the active currency", req.Cur[0])})
	}

	// If automatically filling source TID is enabled then validate that
	// source.TID exists and If it doesn't, fill it with a randomly generated UUID
	if deps.cfg.AutoGenSourceTID {
		if err := validateAndFillSourceTID(req, deps.cfg.GenerateRequestID, hasStoredBidRequest, isAmp); err != nil {
			return []error{err}
		}
	}

	var aliases map[string]string
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return []error{fmt.Errorf("request.ext is invalid: %v", err)}
	}

	reqPrebid := reqExt.GetPrebid()
	if err := deps.parseBidExt(req); err != nil {
		return []error{err}
	}

	if reqPrebid != nil {
		aliases = reqPrebid.Aliases

		if err := deps.validateAliases(aliases); err != nil {
			return []error{err}
		}

		if err := deps.validateAliasesGVLIDs(reqPrebid.AliasGVLIDs, aliases); err != nil {
			return []error{err}
		}

		if err := deps.validateBidAdjustmentFactors(reqPrebid.BidAdjustmentFactors, aliases); err != nil {
			return []error{err}
		}

		if err := validateSChains(reqPrebid.SChains); err != nil {
			return []error{err}
		}

		if err := deps.validateEidPermissions(reqPrebid.Data, aliases); err != nil {
			return []error{err}
		}

		if err := currency.ValidateCustomRates(reqPrebid.CurrencyConversions); err != nil {
			return []error{err}
		}
	}

	if err := mapSChains(req); err != nil {
		return []error{err}
	}

	if err := validateOrFillChannel(req, isAmp); err != nil {
		return []error{err}
	}

	if (req.Site == nil && req.App == nil) || (req.Site != nil && req.App != nil) {
		return append(errL, errors.New("request.site or request.app must be defined, but not both."))
	}

	if errs := validateRequestExt(req); len(errs) != 0 {
		if errortypes.ContainsFatalError(errs) {
			return append(errL, errs...)
		}
		errL = append(errL, errs...)
	}

	if err := deps.validateSite(req); err != nil {
		return append(errL, err)
	}

	if err := deps.validateApp(req); err != nil {
		return append(errL, err)
	}

	var gpp gpplib.GppContainer
	if req.BidRequest.Regs != nil && len(req.BidRequest.Regs.GPP) > 0 {
		gpp, err = gpplib.Parse(req.BidRequest.Regs.GPP)
		if err != nil {
			errL = append(errL, &errortypes.Warning{
				Message:     fmt.Sprintf("GPP consent string is invalid and will be ignored. (%v)", err),
				WarningCode: errortypes.InvalidPrivacyConsentWarningCode})
		}
	}

	if errs := deps.validateUser(req, aliases, gpp); errs != nil {
		if len(errs) > 0 {
			errL = append(errL, errs...)
		}
		if errortypes.ContainsFatalError(errs) {
			return errL
		}
	}

	if errs := validateRegs(req, gpp); errs != nil {
		if len(errs) > 0 {
			errL = append(errL, errs...)
		}
		if errortypes.ContainsFatalError(errs) {
			return errL
		}
	}

	if err := validateDevice(req.Device); err != nil {
		return append(errL, err)
	}

	if ccpaPolicy, err := ccpa.ReadFromRequestWrapper(req, gpp); err != nil {
		errL = append(errL, err)
		if errortypes.ContainsFatalError([]error{err}) {
			return errL
		}
	} else if _, err := ccpaPolicy.Parse(exchange.GetValidBidders(aliases)); err != nil {
		if _, invalidConsent := err.(*errortypes.Warning); invalidConsent {
			errL = append(errL, &errortypes.Warning{
				Message:     fmt.Sprintf("CCPA consent is invalid and will be ignored. (%v)", err),
				WarningCode: errortypes.InvalidPrivacyConsentWarningCode})
			regsExt, err := req.GetRegExt()
			if err != nil {
				return append(errL, err)
			}
			regsExt.SetUSPrivacy("")
		} else {
			return append(errL, err)
		}
	}

	impIDs := make(map[string]int, req.LenImp())
	for i, imp := range req.GetImp() {
		// check for unique imp id
		if firstIndex, ok := impIDs[imp.ID]; ok {
			errL = append(errL, fmt.Errorf(`request.imp[%d].id and request.imp[%d].id are both "%s". Imp IDs must be unique.`, firstIndex, i, imp.ID))
		}
		impIDs[imp.ID] = i

		errs := deps.validateImp(imp, aliases, i, hasStoredResponses, storedBidResp)
		if len(errs) > 0 {
			errL = append(errL, errs...)
		}
		if errortypes.ContainsFatalError(errs) {
			return errL
		}
	}

	return errL
}

// mapSChains maps an schain defined in an ORTB 2.4 location (req.ext.schain) to the ORTB 2.5 location
// (req.source.ext.schain) if no ORTB 2.5 schain (req.source.ext.schain, req.ext.prebid.schains) exists.
// An ORTB 2.4 schain is always deleted from the 2.4 location regardless of whether an ORTB 2.5 schain exists.
func mapSChains(req *openrtb_ext.RequestWrapper) error {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return fmt.Errorf("req.ext is invalid: %v", err)
	}
	sourceExt, err := req.GetSourceExt()
	if err != nil {
		return fmt.Errorf("source.ext is invalid: %v", err)
	}

	reqExtSChain := reqExt.GetSChain()
	reqExt.SetSChain(nil)

	if reqPrebid := reqExt.GetPrebid(); reqPrebid != nil && reqPrebid.SChains != nil {
		return nil
	} else if sourceExt.GetSChain() != nil {
		return nil
	} else if reqExtSChain != nil {
		sourceExt.SetSChain(reqExtSChain)
	}
	return nil
}

func validateAndFillSourceTID(req *openrtb_ext.RequestWrapper, generateRequestID bool, hasStoredBidRequest bool, isAmp bool) error {
	if req.Source == nil {
		req.Source = &openrtb2.Source{}
	}

	if req.Source.TID == "" || req.Source.TID == "{{UUID}}" || (generateRequestID && (isAmp || hasStoredBidRequest)) {
		rawUUID, err := uuid.NewV4()
		if err != nil {
			return errors.New("error creating a random UUID for source.tid")
		}
		req.Source.TID = rawUUID.String()
	}

	for _, impWrapper := range req.GetImp() {
		ie, _ := impWrapper.GetImpExt()
		if ie.GetTid() == "" || ie.GetTid() == "{{UUID}}" || (generateRequestID && (isAmp || hasStoredBidRequest)) {
			rawUUID, err := uuid.NewV4()
			if err != nil {
				return errors.New("imp.ext.tid missing in the imp and error creating a random UID")
			}
			ie.SetTid(rawUUID.String())
			impWrapper.RebuildImp()
		}
	}

	return nil
}

func (deps *endpointDeps) validateBidAdjustmentFactors(adjustmentFactors map[string]float64, aliases map[string]string) error {
	for bidderToAdjust, adjustmentFactor := range adjustmentFactors {
		if adjustmentFactor <= 0 {
			return fmt.Errorf("request.ext.prebid.bidadjustmentfactors.%s must be a positive number. Got %f", bidderToAdjust, adjustmentFactor)
		}
		if _, isBidder := deps.bidderMap[bidderToAdjust]; !isBidder {
			if _, isAlias := aliases[bidderToAdjust]; !isAlias {
				return fmt.Errorf("request.ext.prebid.bidadjustmentfactors.%s is not a known bidder or alias", bidderToAdjust)
			}
		}
	}
	return nil
}

func validateSChains(sChains []*openrtb_ext.ExtRequestPrebidSChain) error {
	_, err := schain.BidderToPrebidSChains(sChains)
	return err
}

func (deps *endpointDeps) validateEidPermissions(prebid *openrtb_ext.ExtRequestPrebidData, aliases map[string]string) error {
	if prebid == nil {
		return nil
	}

	uniqueSources := make(map[string]struct{}, len(prebid.EidPermissions))
	for i, eid := range prebid.EidPermissions {
		if len(eid.Source) == 0 {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] missing required field: "source"`, i)
		}

		if _, exists := uniqueSources[eid.Source]; exists {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] duplicate entry with field: "source"`, i)
		}
		uniqueSources[eid.Source] = struct{}{}

		if len(eid.Bidders) == 0 {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] missing or empty required field: "bidders"`, i)
		}

		if err := validateBidders(eid.Bidders, deps.bidderMap, aliases); err != nil {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] contains %v`, i, err)
		}
	}

	return nil
}

func validateBidders(bidders []string, knownBidders map[string]openrtb_ext.BidderName, knownAliases map[string]string) error {
	for _, bidder := range bidders {
		if bidder == "*" {
			if len(bidders) > 1 {
				return errors.New(`bidder wildcard "*" mixed with specific bidders`)
			}
		} else {
			_, isCoreBidder := knownBidders[bidder]
			_, isAlias := knownAliases[bidder]
			if !isCoreBidder && !isAlias {
				return fmt.Errorf(`unrecognized bidder "%v"`, bidder)
			}
		}
	}
	return nil
}

func (deps *endpointDeps) validateImp(imp *openrtb_ext.ImpWrapper, aliases map[string]string, index int, hasStoredResponses bool, storedBidResp stored_responses.ImpBidderStoredResp) []error {
	if imp.ID == "" {
		return []error{fmt.Errorf("request.imp[%d] missing required field: \"id\"", index)}
	}

	if len(imp.Metric) != 0 {
		return []error{fmt.Errorf("request.imp[%d].metric is not yet supported by prebid-server. Support may be added in the future", index)}
	}

	if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
		return []error{fmt.Errorf("request.imp[%d] must contain at least one of \"banner\", \"video\", \"audio\", or \"native\"", index)}
	}

	if err := validateBanner(imp.Banner, index, isInterstitial(imp)); err != nil {
		return []error{err}
	}

	if err := validateVideo(imp.Video, index); err != nil {
		return []error{err}
	}

	if err := validateAudio(imp.Audio, index); err != nil {
		return []error{err}
	}

	if err := fillAndValidateNative(imp.Native, index); err != nil {
		return []error{err}
	}

	if err := validatePmp(imp.PMP, index); err != nil {
		return []error{err}
	}

	errL := deps.validateImpExt(imp, aliases, index, hasStoredResponses, storedBidResp)
	if len(errL) != 0 {
		return errL
	}

	return nil
}

func isInterstitial(imp *openrtb_ext.ImpWrapper) bool {
	return imp.Instl == 1
}

func validateBanner(banner *openrtb2.Banner, impIndex int, isInterstitial bool) error {
	if banner == nil {
		return nil
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if banner.W != nil && *banner.W < 0 {
		return fmt.Errorf("request.imp[%d].banner.w must be a positive number", impIndex)
	}
	if banner.H != nil && *banner.H < 0 {
		return fmt.Errorf("request.imp[%d].banner.h must be a positive number", impIndex)
	}

	// The following fields are deprecated in the OpenRTB 2.5 spec but are still present
	// in the OpenRTB library we use. Enforce they are not specified.
	if banner.WMin != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"wmin\". Use the \"format\" array instead.", impIndex)
	}
	if banner.WMax != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"wmax\". Use the \"format\" array instead.", impIndex)
	}
	if banner.HMin != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"hmin\". Use the \"format\" array instead.", impIndex)
	}
	if banner.HMax != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"hmax\". Use the \"format\" array instead.", impIndex)
	}

	hasRootSize := banner.H != nil && banner.W != nil && *banner.H > 0 && *banner.W > 0
	if !hasRootSize && len(banner.Format) == 0 && !isInterstitial {
		return fmt.Errorf("request.imp[%d].banner has no sizes. Define \"w\" and \"h\", or include \"format\" elements.", impIndex)
	}

	for i, format := range banner.Format {
		if err := validateFormat(&format, impIndex, i); err != nil {
			return err
		}
	}

	return nil
}

func validateVideo(video *openrtb2.Video, impIndex int) error {
	if video == nil {
		return nil
	}

	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].video.mimes must contain at least one supported MIME type", impIndex)
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if video.W < 0 {
		return fmt.Errorf("request.imp[%d].video.w must be a positive number", impIndex)
	}
	if video.H < 0 {
		return fmt.Errorf("request.imp[%d].video.h must be a positive number", impIndex)
	}
	if video.MinBitRate < 0 {
		return fmt.Errorf("request.imp[%d].video.minbitrate must be a positive number", impIndex)
	}
	if video.MaxBitRate < 0 {
		return fmt.Errorf("request.imp[%d].video.maxbitrate must be a positive number", impIndex)
	}

	return nil
}

func validateAudio(audio *openrtb2.Audio, impIndex int) error {
	if audio == nil {
		return nil
	}

	if len(audio.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].audio.mimes must contain at least one supported MIME type", impIndex)
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if audio.Sequence < 0 {
		return fmt.Errorf("request.imp[%d].audio.sequence must be a positive number", impIndex)
	}
	if audio.MaxSeq < 0 {
		return fmt.Errorf("request.imp[%d].audio.maxseq must be a positive number", impIndex)
	}
	if audio.MinBitrate < 0 {
		return fmt.Errorf("request.imp[%d].audio.minbitrate must be a positive number", impIndex)
	}
	if audio.MaxBitrate < 0 {
		return fmt.Errorf("request.imp[%d].audio.maxbitrate must be a positive number", impIndex)
	}

	return nil
}

// fillAndValidateNative validates the request, and assigns the Asset IDs as recommended by the Native v1.2 spec.
func fillAndValidateNative(n *openrtb2.Native, impIndex int) error {
	if n == nil {
		return nil
	}

	if len(n.Request) == 0 {
		return fmt.Errorf("request.imp[%d].native missing required property \"request\"", impIndex)
	}
	var nativePayload nativeRequests.Request
	if err := json.Unmarshal(json.RawMessage(n.Request), &nativePayload); err != nil {
		return err
	}

	if err := validateNativeContextTypes(nativePayload.Context, nativePayload.ContextSubType, impIndex); err != nil {
		return err
	}
	if err := validateNativePlacementType(nativePayload.PlcmtType, impIndex); err != nil {
		return err
	}
	if err := fillAndValidateNativeAssets(nativePayload.Assets, impIndex); err != nil {
		return err
	}
	if err := validateNativeEventTrackers(nativePayload.EventTrackers, impIndex); err != nil {
		return err
	}

	serialized, err := json.Marshal(nativePayload)
	if err != nil {
		return err
	}
	n.Request = string(serialized)
	return nil
}

func validateNativeContextTypes(cType native1.ContextType, cSubtype native1.ContextSubType, impIndex int) error {
	if cType == 0 {
		// Context is only recommended, so none is a valid type.
		return nil
	}
	if cType < native1.ContextTypeContent || (cType > native1.ContextTypeProduct && cType < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
	}
	if cSubtype < 0 {
		return fmt.Errorf("request.imp[%d].native.request.contextsubtype value can't be less than 0. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
	}
	if cSubtype == 0 {
		return nil
	}
	if cSubtype >= native1.ContextSubTypeGeneral && cSubtype <= native1.ContextSubTypeUserGenerated {
		if cType != native1.ContextTypeContent && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= native1.ContextSubTypeSocial && cSubtype <= native1.ContextSubTypeChat {
		if cType != native1.ContextTypeSocial && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= native1.ContextSubTypeSelling && cSubtype <= native1.ContextSubTypeProductReview {
		if cType != native1.ContextTypeProduct && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= openrtb_ext.NativeExchangeSpecificLowerBound {
		return nil
	}

	return fmt.Errorf("request.imp[%d].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
}

func validateNativePlacementType(pt native1.PlacementType, impIndex int) error {
	if pt == 0 {
		// Placement Type is only recommended, not required.
		return nil
	}
	if pt < native1.PlacementTypeFeed || (pt > native1.PlacementTypeRecommendationWidget && pt < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40", impIndex)
	}
	return nil
}

func fillAndValidateNativeAssets(assets []nativeRequests.Asset, impIndex int) error {
	if len(assets) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets must be an array containing at least one object", impIndex)
	}

	assetIDs := make(map[int64]struct{}, len(assets))

	// If none of the asset IDs are defined by the caller, then prebid server should assign its own unique IDs. But
	// if the caller did assign its own asset IDs, then prebid server will respect those IDs
	assignAssetIDs := true
	for i := 0; i < len(assets); i++ {
		assignAssetIDs = assignAssetIDs && (assets[i].ID == 0)
	}

	for i := 0; i < len(assets); i++ {
		if err := validateNativeAsset(assets[i], impIndex, i); err != nil {
			return err
		}

		if assignAssetIDs {
			assets[i].ID = int64(i)
			continue
		}

		// Each asset should have a unique ID thats assigned by the caller
		if _, ok := assetIDs[assets[i].ID]; ok {
			return fmt.Errorf("request.imp[%d].native.request.assets[%d].id is already being used by another asset. Each asset ID must be unique.", impIndex, i)
		}

		assetIDs[assets[i].ID] = struct{}{}
	}

	return nil
}

func validateNativeAsset(asset nativeRequests.Asset, impIndex int, assetIndex int) error {
	assetErr := "request.imp[%d].native.request.assets[%d] must define exactly one of {title, img, video, data}"
	foundType := false

	if asset.Title != nil {
		foundType = true
		if err := validateNativeAssetTitle(asset.Title, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Img != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetImage(asset.Img, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Video != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetVideo(asset.Video, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Data != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetData(asset.Data, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if !foundType {
		return fmt.Errorf(assetErr, impIndex, assetIndex)
	}

	return nil
}

func validateNativeEventTrackers(trackers []nativeRequests.EventTracker, impIndex int) error {
	for i := 0; i < len(trackers); i++ {
		if err := validateNativeEventTracker(trackers[i], impIndex, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeAssetTitle(title *nativeRequests.Title, impIndex int, assetIndex int) error {
	if title.Len < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].title.len must be a positive number", impIndex, assetIndex)
	}
	return nil
}

func validateNativeEventTracker(tracker nativeRequests.EventTracker, impIndex int, eventIndex int) error {
	if tracker.Event < native1.EventTypeImpression || (tracker.Event > native1.EventTypeViewableVideo50 && tracker.Event < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex)
	}
	if len(tracker.Methods) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].method is required. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex)
	}
	for methodIndex, method := range tracker.Methods {
		if method < native1.EventTrackingMethodImage || (method > native1.EventTrackingMethodJS && method < openrtb_ext.NativeExchangeSpecificLowerBound) {
			return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].methods[%d] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex, methodIndex)
		}
	}

	return nil
}

func validateNativeAssetImage(img *nativeRequests.Image, impIndex int, assetIndex int) error {
	if img.W < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.w must be a positive integer", impIndex, assetIndex)
	}
	if img.H < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.h must be a positive integer", impIndex, assetIndex)
	}
	if img.WMin < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.wmin must be a positive integer", impIndex, assetIndex)
	}
	if img.HMin < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.hmin must be a positive integer", impIndex, assetIndex)
	}
	return nil
}

func validateNativeAssetVideo(video *nativeRequests.Video, impIndex int, assetIndex int) error {
	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.mimes must be an array with at least one MIME type", impIndex, assetIndex)
	}
	if video.MinDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.minduration must be a positive integer", impIndex, assetIndex)
	}
	if video.MaxDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.maxduration must be a positive integer", impIndex, assetIndex)
	}
	if err := validateNativeVideoProtocols(video.Protocols, impIndex, assetIndex); err != nil {
		return err
	}

	return nil
}

func validateNativeAssetData(data *nativeRequests.Data, impIndex int, assetIndex int) error {
	if data.Type < native1.DataAssetTypeSponsored || (data.Type > native1.DataAssetTypeCTAText && data.Type < 500) {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40", impIndex, assetIndex)
	}

	return nil
}

func validateNativeVideoProtocols(protocols []adcom1.MediaCreativeSubtype, impIndex int, assetIndex int) error {
	if len(protocols) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols must be an array with at least one element", impIndex, assetIndex)
	}
	for i := 0; i < len(protocols); i++ {
		if err := validateNativeVideoProtocol(protocols[i], impIndex, assetIndex, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeVideoProtocol(protocol adcom1.MediaCreativeSubtype, impIndex int, assetIndex int, protocolIndex int) error {
	if protocol < adcom1.CreativeVAST10 || protocol > adcom1.CreativeDAAST10Wrapper {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols[%d] is invalid. See Section 5.8: https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf#page=52", impIndex, assetIndex, protocolIndex)
	}
	return nil
}

func validateFormat(format *openrtb2.Format, impIndex, formatIndex int) error {
	usesHW := format.W != 0 || format.H != 0
	usesRatios := format.WMin != 0 || format.WRatio != 0 || format.HRatio != 0

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if format.W < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].w must be a positive number", impIndex, formatIndex)
	}
	if format.H < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].h must be a positive number", impIndex, formatIndex)
	}
	if format.WRatio < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].wratio must be a positive number", impIndex, formatIndex)
	}
	if format.HRatio < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].hratio must be a positive number", impIndex, formatIndex)
	}
	if format.WMin < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].wmin must be a positive number", impIndex, formatIndex)
	}

	if usesHW && usesRatios {
		return fmt.Errorf("Request imp[%d].banner.format[%d] should define *either* {w, h} *or* {wmin, wratio, hratio}, but not both. If both are valid, send two \"format\" objects in the request.", impIndex, formatIndex)
	}
	if !usesHW && !usesRatios {
		return fmt.Errorf("Request imp[%d].banner.format[%d] should define *either* {w, h} (for static size requirements) *or* {wmin, wratio, hratio} (for flexible sizes) to be non-zero.", impIndex, formatIndex)
	}
	if usesHW && (format.W == 0 || format.H == 0) {
		return fmt.Errorf("Request imp[%d].banner.format[%d] must define non-zero \"h\" and \"w\" properties.", impIndex, formatIndex)
	}
	if usesRatios && (format.WMin == 0 || format.WRatio == 0 || format.HRatio == 0) {
		return fmt.Errorf("Request imp[%d].banner.format[%d] must define non-zero \"wmin\", \"wratio\", and \"hratio\" properties.", impIndex, formatIndex)
	}
	return nil
}

func validatePmp(pmp *openrtb2.PMP, impIndex int) error {
	if pmp == nil {
		return nil
	}

	for dealIndex, deal := range pmp.Deals {
		if deal.ID == "" {
			return fmt.Errorf("request.imp[%d].pmp.deals[%d] missing required field: \"id\"", impIndex, dealIndex)
		}
	}
	return nil
}

func (deps *endpointDeps) validateImpExt(imp *openrtb_ext.ImpWrapper, aliases map[string]string, impIndex int, hasStoredResponses bool, storedBidResp stored_responses.ImpBidderStoredResp) []error {
	if len(imp.Ext) == 0 {
		return []error{fmt.Errorf("request.imp[%d].ext is required", impIndex)}
	}

	impExt, err := imp.GetImpExt()
	if err != nil {
		return []error{err}
	}

	prebid := impExt.GetOrCreatePrebid()
	prebidModified := false

	if prebid.Bidder == nil {
		prebid.Bidder = make(map[string]json.RawMessage)
	}

	ext := impExt.GetExt()
	extModified := false

	// promote imp[].ext.BIDDER to newer imp[].ext.prebid.bidder.BIDDER location, with the later taking precedence
	for k, v := range ext {
		if isPossibleBidder(k) {
			if _, exists := prebid.Bidder[k]; !exists {
				prebid.Bidder[k] = v
				prebidModified = true
			}
			delete(ext, k)
			extModified = true
		}
	}

	if hasStoredResponses && prebid.StoredAuctionResponse == nil {
		return []error{fmt.Errorf("request validation failed. The StoredAuctionResponse.ID field must be completely present with, or completely absent from, all impressions in request. No StoredAuctionResponse data found for request.imp[%d].ext.prebid \n", impIndex)}
	}

	if len(storedBidResp) > 0 {
		if err := validateStoredBidRespAndImpExtBidders(prebid.Bidder, storedBidResp, imp.ID); err != nil {
			return []error{err}
		}
	}

	errL := []error{}

	for bidder, ext := range prebid.Bidder {
		coreBidder := bidder
		if tmp, isAlias := aliases[bidder]; isAlias {
			coreBidder = tmp
		}

		if coreBidderNormalized, isValid := deps.bidderMap[coreBidder]; isValid {
			if err := deps.paramsValidator.Validate(coreBidderNormalized, ext); err != nil {
				return []error{fmt.Errorf("request.imp[%d].ext.prebid.bidder.%s failed validation.\n%v", impIndex, bidder, err)}
			}
		} else {
			if msg, isDisabled := deps.disabledBidders[bidder]; isDisabled {
				errL = append(errL, &errortypes.BidderTemporarilyDisabled{Message: msg})
				delete(prebid.Bidder, bidder)
				prebidModified = true
			} else {
				return []error{fmt.Errorf("request.imp[%d].ext.prebid.bidder contains unknown bidder: %s. Did you forget an alias in request.ext.prebid.aliases?", impIndex, bidder)}
			}
		}
	}

	if len(prebid.Bidder) == 0 {
		errL = append(errL, fmt.Errorf("request.imp[%d].ext.prebid.bidder must contain at least one bidder", impIndex))
		return errL
	}

	if prebidModified {
		impExt.SetPrebid(prebid)
	}
	if extModified {
		impExt.SetExt(ext)
	}

	return errL
}

// isPossibleBidder determines if a bidder name is a potential real bidder.
func isPossibleBidder(bidder string) bool {
	switch openrtb_ext.BidderName(bidder) {
	case openrtb_ext.BidderReservedContext:
		return false
	case openrtb_ext.BidderReservedData:
		return false
	case openrtb_ext.BidderReservedGPID:
		return false
	case openrtb_ext.BidderReservedPrebid:
		return false
	case openrtb_ext.BidderReservedSKAdN:
		return false
	case openrtb_ext.BidderReservedTID:
		return false
	case openrtb_ext.BidderReservedAE:
		return false
	default:
		return true
	}
}

func (deps *endpointDeps) parseBidExt(req *openrtb_ext.RequestWrapper) error {
	if _, err := req.GetRequestExt(); err != nil {
		return fmt.Errorf("request.ext is invalid: %v", err)
	}
	return nil
}

func (deps *endpointDeps) validateAliases(aliases map[string]string) error {
	for alias, coreBidder := range aliases {
		if _, isCoreBidderDisabled := deps.disabledBidders[coreBidder]; isCoreBidderDisabled {
			return fmt.Errorf("request.ext.prebid.aliases.%s refers to disabled bidder: %s", alias, coreBidder)
		}

		if _, isCoreBidder := deps.bidderMap[coreBidder]; !isCoreBidder {
			return fmt.Errorf("request.ext.prebid.aliases.%s refers to unknown bidder: %s", alias, coreBidder)
		}

		if alias == coreBidder {
			return fmt.Errorf("request.ext.prebid.aliases.%s defines a no-op alias. Choose a different alias, or remove this entry.", alias)
		}
	}
	return nil
}

func (deps *endpointDeps) validateAliasesGVLIDs(aliasesGVLIDs map[string]uint16, aliases map[string]string) error {
	for alias, vendorId := range aliasesGVLIDs {

		if _, aliasExist := aliases[alias]; !aliasExist {
			return fmt.Errorf("request.ext.prebid.aliasgvlids. vendorId %d refers to unknown bidder alias: %s", vendorId, alias)
		}

		if vendorId < 1 {
			return fmt.Errorf("request.ext.prebid.aliasgvlids. Invalid vendorId %d for alias: %s. Choose a different vendorId, or remove this entry.", vendorId, alias)
		}
	}
	return nil
}

func validateRequestExt(req *openrtb_ext.RequestWrapper) []error {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return []error{err}
	}

	prebid := reqExt.GetPrebid()
	// exit early if there is no request.ext.prebid to validate
	if prebid == nil {
		return nil
	}

	if prebid.Cache != nil {
		if prebid.Cache.Bids == nil && prebid.Cache.VastXML == nil {
			return []error{errors.New(`request.ext is invalid: request.ext.prebid.cache requires one of the "bids" or "vastxml" properties`)}
		}
	}

	if err := validateTargeting(prebid.Targeting); err != nil {
		return []error{err}
	}

	var errs []error
	if prebid.MultiBid != nil {
		validatedMultiBids, multBidErrs := openrtb_ext.ValidateAndBuildExtMultiBid(prebid)

		for _, err := range multBidErrs {
			errs = append(errs, &errortypes.Warning{
				WarningCode: errortypes.MultiBidWarningCode,
				Message:     err.Error(),
			})
		}

		// update the downstream multibid to avoid passing unvalidated ext to bidders, etc.
		prebid.MultiBid = validatedMultiBids
		reqExt.SetPrebid(prebid)
	}

	if !bidadjustment.Validate(prebid.BidAdjustments) {
		prebid.BidAdjustments = nil
		reqExt.SetPrebid(prebid)
		errs = append(errs, &errortypes.Warning{
			WarningCode: errortypes.BidAdjustmentWarningCode,
			Message:     "bid adjustment from request was invalid",
		})
	}

	return errs
}

func validateTargeting(t *openrtb_ext.ExtRequestTargeting) error {
	if t == nil {
		return nil
	}

	if (t.IncludeWinners == nil || !*t.IncludeWinners) && (t.IncludeBidderKeys == nil || !*t.IncludeBidderKeys) {
		return errors.New("ext.prebid.targeting: At least one of includewinners or includebidderkeys must be enabled to enable targeting support")
	}

	if t.PriceGranularity != nil {
		if err := validatePriceGranularity(t.PriceGranularity); err != nil {
			return err
		}
	}

	if t.MediaTypePriceGranularity.Video != nil {
		if err := validatePriceGranularity(t.MediaTypePriceGranularity.Video); err != nil {
			return err
		}
	}
	if t.MediaTypePriceGranularity.Banner != nil {
		if err := validatePriceGranularity(t.MediaTypePriceGranularity.Banner); err != nil {
			return err
		}
	}
	if t.MediaTypePriceGranularity.Native != nil {
		if err := validatePriceGranularity(t.MediaTypePriceGranularity.Native); err != nil {
			return err
		}
	}

	return nil
}

func validatePriceGranularity(pg *openrtb_ext.PriceGranularity) error {
	if pg.Precision == nil {
		return errors.New("Price granularity error: precision is required")
	} else if *pg.Precision < 0 {
		return errors.New("Price granularity error: precision must be non-negative")
	} else if *pg.Precision > openrtb_ext.MaxDecimalFigures {
		return fmt.Errorf("Price granularity error: precision of more than %d significant figures is not supported", openrtb_ext.MaxDecimalFigures)
	}

	var prevMax float64 = 0
	for _, gr := range pg.Ranges {
		if gr.Max <= prevMax {
			return errors.New(`Price granularity error: range list must be ordered with increasing "max"`)
		}

		if gr.Increment <= 0.0 {
			return errors.New("Price granularity error: increment must be a nonzero positive number")
		}
		prevMax = gr.Max
	}
	return nil
}

func (deps *endpointDeps) validateSite(req *openrtb_ext.RequestWrapper) error {
	if req.Site == nil {
		return nil
	}

	if req.Site.ID == "" && req.Site.Page == "" {
		return errors.New("request.site should include at least one of request.site.id or request.site.page.")
	}
	siteExt, err := req.GetSiteExt()
	if err != nil {
		return err
	}
	siteAmp := siteExt.GetAmp()
	if siteAmp != nil && (*siteAmp < 0 || *siteAmp > 1) {
		return errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
	}

	return nil
}

func (deps *endpointDeps) validateApp(req *openrtb_ext.RequestWrapper) error {
	if req.App == nil {
		return nil
	}

	if req.App.ID != "" {
		if _, found := deps.cfg.BlacklistedAppMap[req.App.ID]; found {
			return &errortypes.BlacklistedApp{Message: fmt.Sprintf("Prebid-server does not process requests from App ID: %s", req.App.ID)}
		}
	}

	_, err := req.GetAppExt()
	return err
}

func (deps *endpointDeps) validateUser(req *openrtb_ext.RequestWrapper, aliases map[string]string, gpp gpplib.GppContainer) []error {
	var errL []error

	if req == nil || req.BidRequest == nil || req.BidRequest.User == nil {
		return nil
	}
	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if req.User.Geo != nil && req.User.Geo.Accuracy < 0 {
		return append(errL, errors.New("request.user.geo.accuracy must be a positive number"))
	}

	if req.User.Consent != "" {
		for _, section := range gpp.Sections {
			if section.GetID() == constants.SectionTCFEU2 && section.GetValue() != req.User.Consent {
				errL = append(errL, &errortypes.Warning{
					Message:     "user.consent GDPR string conflicts with GPP (regs.gpp) GDPR string, using regs.gpp",
					WarningCode: errortypes.InvalidPrivacyConsentWarningCode})
			}
		}
	}
	userExt, err := req.GetUserExt()
	if err != nil {
		return append(errL, fmt.Errorf("request.user.ext object is not valid: %v", err))
	}
	// Check if the buyeruids are valid
	prebid := userExt.GetPrebid()
	if prebid != nil {
		if len(prebid.BuyerUIDs) < 1 {
			return append(errL, errors.New(`request.user.ext.prebid requires a "buyeruids" property with at least one ID defined. If none exist, then request.user.ext.prebid should not be defined.`))
		}
		for bidderName := range prebid.BuyerUIDs {
			if _, ok := deps.bidderMap[bidderName]; !ok {
				if _, ok := aliases[bidderName]; !ok {
					return append(errL, fmt.Errorf("request.user.ext.%s is neither a known bidder name nor an alias in request.ext.prebid.aliases", bidderName))
				}
			}
		}
	}
	// Check Universal User ID
	eids := userExt.GetEid()
	if eids != nil {
		eidsValue := *eids
		uniqueSources := make(map[string]struct{}, len(eidsValue))
		for eidIndex, eid := range eidsValue {
			if eid.Source == "" {
				return append(errL, fmt.Errorf("request.user.ext.eids[%d] missing required field: \"source\"", eidIndex))
			}
			if _, ok := uniqueSources[eid.Source]; ok {
				return append(errL, errors.New("request.user.ext.eids must contain unique sources"))
			}
			uniqueSources[eid.Source] = struct{}{}

			if len(eid.UIDs) == 0 {
				return append(errL, fmt.Errorf("request.user.ext.eids[%d].uids must contain at least one element or be undefined", eidIndex))
			}

			for uidIndex, uid := range eid.UIDs {
				if uid.ID == "" {
					return append(errL, fmt.Errorf("request.user.ext.eids[%d].uids[%d] missing required field: \"id\"", eidIndex, uidIndex))
				}
			}
		}
	}

	return errL
}

func validateRegs(req *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) []error {
	var errL []error

	if req == nil || req.BidRequest == nil || req.BidRequest.Regs == nil {
		return nil
	}

	if req.BidRequest.Regs.GDPR != nil && req.BidRequest.Regs.GPPSID != nil {
		gdpr := int8(0)
		for _, id := range req.BidRequest.Regs.GPPSID {
			if id == int8(constants.SectionTCFEU2) {
				gdpr = 1
				break
			}
		}
		if gdpr != *req.BidRequest.Regs.GDPR {
			errL = append(errL, &errortypes.Warning{
				Message:     "regs.gdpr signal conflicts with GPP (regs.gpp_sid) and will be ignored",
				WarningCode: errortypes.InvalidPrivacyConsentWarningCode})
		}
	}
	regsExt, err := req.GetRegExt()
	if err != nil {
		return append(errL, fmt.Errorf("request.regs.ext is invalid: %v", err))
	}

	gdpr := regsExt.GetGDPR()
	if gdpr != nil && *gdpr != 0 && *gdpr != 1 {
		return append(errL, errors.New("request.regs.ext.gdpr must be either 0 or 1"))
	}

	return errL
}

func validateDevice(device *openrtb2.Device) error {
	if device == nil {
		return nil
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if device.W < 0 {
		return errors.New("request.device.w must be a positive number")
	}
	if device.H < 0 {
		return errors.New("request.device.h must be a positive number")
	}
	if device.PPI < 0 {
		return errors.New("request.device.ppi must be a positive number")
	}
	if device.Geo != nil && device.Geo.Accuracy < 0 {
		return errors.New("request.device.geo.accuracy must be a positive number")
	}

	return nil
}

func validateOrFillChannel(reqWrapper *openrtb_ext.RequestWrapper, isAmp bool) error {
	requestExt, err := reqWrapper.GetRequestExt()
	if err != nil {
		return err
	}
	requestPrebid := requestExt.GetPrebid()

	if requestPrebid == nil || requestPrebid.Channel == nil {
		fillChannel(reqWrapper, isAmp)
	} else if requestPrebid.Channel.Name == "" {
		return errors.New("ext.prebid.channel.name can't be empty")
	}
	return nil
}

func fillChannel(reqWrapper *openrtb_ext.RequestWrapper, isAmp bool) error {
	var channelName string
	requestExt, err := reqWrapper.GetRequestExt()
	if err != nil {
		return err
	}
	requestPrebid := requestExt.GetPrebid()
	if isAmp {
		channelName = ampChannel
	}
	if reqWrapper.App != nil {
		channelName = appChannel
	}
	if channelName != "" {
		if requestPrebid == nil {
			requestPrebid = &openrtb_ext.ExtRequestPrebid{}
		}
		requestPrebid.Channel = &openrtb_ext.ExtRequestPrebidChannel{Name: channelName}
		requestExt.SetPrebid(requestPrebid)
		reqWrapper.RebuildRequest()
	}
	return nil

}

func sanitizeRequest(r *openrtb_ext.RequestWrapper, ipValidator iputil.IPValidator) {
	if r.Device != nil {
		if ip, ver := iputil.ParseIP(r.Device.IP); ip == nil || ver != iputil.IPv4 || !ipValidator.IsValid(ip, ver) {
			r.Device.IP = ""
		}

		if ip, ver := iputil.ParseIP(r.Device.IPv6); ip == nil || ver != iputil.IPv6 || !ipValidator.IsValid(ip, ver) {
			r.Device.IPv6 = ""
		}
	}
}

// setFieldsImplicitly uses _implicit_ information from the httpReq to set values on bidReq.
// This function does not consume the request body, which was set explicitly, but infers certain
// OpenRTB properties from the headers and other implicit info.
//
// This function _should not_ override any fields which were defined explicitly by the caller in the request.
func (deps *endpointDeps) setFieldsImplicitly(httpReq *http.Request, r *openrtb_ext.RequestWrapper) {
	sanitizeRequest(r, deps.privateNetworkIPValidator)

	setDeviceImplicitly(httpReq, r, deps.privateNetworkIPValidator)

	// Per the OpenRTB spec: A bid request must not contain both a Site and an App object. If neither are
	// present, we'll assume it's a site request.
	if r.App == nil {
		setSiteImplicitly(httpReq, r)
	}

	setAuctionTypeImplicitly(r)
}

// setDeviceImplicitly uses implicit info from httpReq to populate bidReq.Device
func setDeviceImplicitly(httpReq *http.Request, r *openrtb_ext.RequestWrapper, ipValidtor iputil.IPValidator) {
	setIPImplicitly(httpReq, r, ipValidtor)
	setUAImplicitly(httpReq, r)
	setDoNotTrackImplicitly(httpReq, r)

}

// setAuctionTypeImplicitly sets the auction type to 1 if it wasn't on the request,
// since header bidding is generally a first-price auction.
func setAuctionTypeImplicitly(r *openrtb_ext.RequestWrapper) {
	if r.AT == 0 {
		r.AT = 1
	}
}

func setSiteImplicitly(httpReq *http.Request, r *openrtb_ext.RequestWrapper) {
	referrerCandidate := httpReq.Referer()
	if referrerCandidate == "" && r.Site != nil && r.Site.Page != "" {
		referrerCandidate = r.Site.Page // If http referer is disabled and thus has empty value - use site.page instead
	}

	if referrerCandidate != "" {
		setSitePageIfEmpty(r.Site, referrerCandidate)
		if parsedUrl, err := url.Parse(referrerCandidate); err == nil {
			setSiteDomainIfEmpty(r.Site, parsedUrl.Host)
			if publisherDomain, err := publicsuffix.EffectiveTLDPlusOne(parsedUrl.Host); err == nil {
				setSitePublisherDomainIfEmpty(r.Site, publisherDomain)
			}
		}
	}

	if r.Site != nil {
		if siteExt, err := r.GetSiteExt(); err == nil && siteExt.GetAmp() == nil {
			siteExt.SetAmp(&notAmp)
		}
	}
}

func setSitePageIfEmpty(site *openrtb2.Site, sitePage string) {
	if site == nil {
		site = &openrtb2.Site{}
	}
	if site.Page == "" {
		site.Page = sitePage
	}
}

func setSiteDomainIfEmpty(site *openrtb2.Site, siteDomain string) {
	if site.Domain == "" {
		site.Domain = siteDomain
	}
}

func setSitePublisherDomainIfEmpty(site *openrtb2.Site, publisherDomain string) {
	if site.Publisher == nil {
		site.Publisher = &openrtb2.Publisher{}
	}
	if site.Publisher.Domain == "" {
		site.Publisher.Domain = publisherDomain
	}
}

func getJsonSyntaxError(testJSON []byte) (bool, string) {
	type JsonNode struct {
		raw   *json.RawMessage
		doc   map[string]*JsonNode
		ary   []*JsonNode
		which int
	}
	type jNode map[string]*JsonNode
	docErrdoc := &jNode{}
	docErr := json.Unmarshal(testJSON, docErrdoc)
	if uerror, ok := docErr.(*json.SyntaxError); ok {
		err := fmt.Sprintf("%s at offset %v", uerror.Error(), uerror.Offset)
		return true, err
	}
	return false, ""
}

func (deps *endpointDeps) getStoredRequests(ctx context.Context, requestJson []byte, impInfo []ImpExtPrebidData) (string, bool, map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	// Parse the Stored Request IDs from the BidRequest and Imps.
	storedBidRequestId, hasStoredBidRequest, err := getStoredRequestId(requestJson)
	if err != nil {
		return "", false, nil, nil, []error{err}
	}

	// Fetch the Stored Request data
	var storedReqIds []string
	if hasStoredBidRequest {
		storedReqIds = []string{storedBidRequestId}
	}

	impStoredReqIds := make([]string, 0, len(impInfo))
	impStoredReqIdsUniqueTracker := make(map[string]struct{}, len(impInfo))
	for _, impData := range impInfo {
		if impData.ImpExtPrebid.StoredRequest != nil && len(impData.ImpExtPrebid.StoredRequest.ID) > 0 {
			storedImpId := impData.ImpExtPrebid.StoredRequest.ID
			if _, present := impStoredReqIdsUniqueTracker[storedImpId]; !present {
				impStoredReqIds = append(impStoredReqIds, storedImpId)
				impStoredReqIdsUniqueTracker[storedImpId] = struct{}{}
			}
		}
	}

	storedRequests, storedImps, errs := deps.storedReqFetcher.FetchRequests(ctx, storedReqIds, impStoredReqIds)
	if len(errs) != 0 {
		return "", false, nil, nil, errs
	}

	return storedBidRequestId, hasStoredBidRequest, storedRequests, storedImps, errs
}

func (deps *endpointDeps) processStoredRequests(requestJson []byte, impInfo []ImpExtPrebidData, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage, storedBidRequestId string, hasStoredBidRequest bool) ([]byte, map[string]exchange.ImpExtInfo, []error) {
	bidRequestID, err := getBidRequestID(storedRequests[storedBidRequestId])
	if err != nil {
		return nil, nil, []error{err}
	}

	// Apply the Stored BidRequest, if it exists
	resolvedRequest := requestJson

	if hasStoredBidRequest {
		isAppRequest, err := checkIfAppRequest(requestJson)
		if err != nil {
			return nil, nil, []error{err}
		}
		if (deps.cfg.GenerateRequestID && isAppRequest) || bidRequestID == "{{UUID}}" {
			uuidPatch, err := generateUuidForBidRequest(deps.uuidGenerator)
			if err != nil {
				return nil, nil, []error{err}
			}
			uuidPatch, err = jsonpatch.MergePatch(storedRequests[storedBidRequestId], uuidPatch)
			if err != nil {
				errL := storedRequestErrorChecker(requestJson, storedRequests, storedBidRequestId)
				return nil, nil, errL
			}
			resolvedRequest, err = jsonpatch.MergePatch(requestJson, uuidPatch)
			if err != nil {
				errL := storedRequestErrorChecker(requestJson, storedRequests, storedBidRequestId)
				return nil, nil, errL
			}
		} else {
			resolvedRequest, err = jsonpatch.MergePatch(storedRequests[storedBidRequestId], requestJson)
			if err != nil {
				errL := storedRequestErrorChecker(requestJson, storedRequests, storedBidRequestId)
				return nil, nil, errL
			}
		}
	}

	// Apply default aliases, if they are provided
	if deps.defaultRequest {
		aliasedRequest, err := jsonpatch.MergePatch(deps.defReqJSON, resolvedRequest)
		if err != nil {
			hasErr, Err := getJsonSyntaxError(resolvedRequest)
			if hasErr {
				err = fmt.Errorf("Invalid JSON in Incoming Request: %s", Err)
			} else {
				hasErr, Err = getJsonSyntaxError(deps.defReqJSON)
				if hasErr {
					err = fmt.Errorf("Invalid JSON in Default Request Settings: %s", Err)
				}
			}
			return nil, nil, []error{err}
		}
		resolvedRequest = aliasedRequest
	}

	// Apply any Stored Imps, if they exist. Since the JSON Merge Patch overrides arrays,
	// and Prebid Server defers to the HTTP Request to resolve conflicts, it's safe to
	// assume that the request.imp data did not change when applying the Stored BidRequest.
	impExtInfoMap := make(map[string]exchange.ImpExtInfo, len(impInfo))
	resolvedImps := make([]json.RawMessage, 0, len(impInfo))
	for i, impData := range impInfo {
		if impData.ImpExtPrebid.StoredRequest != nil && len(impData.ImpExtPrebid.StoredRequest.ID) > 0 {
			resolvedImp, err := jsonpatch.MergePatch(storedImps[impData.ImpExtPrebid.StoredRequest.ID], impData.Imp)

			if err != nil {
				hasErr, errMessage := getJsonSyntaxError(impData.Imp)
				if hasErr {
					err = fmt.Errorf("Invalid JSON in Imp[%d] of Incoming Request: %s", i, errMessage)
				} else {
					hasErr, errMessage = getJsonSyntaxError(storedImps[impData.ImpExtPrebid.StoredRequest.ID])
					if hasErr {
						err = fmt.Errorf("imp.ext.prebid.storedrequest.id %s: Stored Imp has Invalid JSON: %s", impData.ImpExtPrebid.StoredRequest.ID, errMessage)
					}
				}
				return nil, nil, []error{err}
			}
			resolvedImps = append(resolvedImps, resolvedImp)
			impId, err := jsonparser.GetString(resolvedImp, "id")
			if err != nil {
				return nil, nil, []error{err}
			}

			echoVideoAttributes := false
			if impData.ImpExtPrebid.Options != nil {
				echoVideoAttributes = impData.ImpExtPrebid.Options.EchoVideoAttrs
			}

			// Extract Passthrough from Merged Imp
			passthrough, _, _, err := jsonparser.Get(resolvedImp, "ext", "prebid", "passthrough")
			if err != nil && err != jsonparser.KeyPathNotFoundError {
				return nil, nil, []error{err}
			}
			impExtInfoMap[impId] = exchange.ImpExtInfo{EchoVideoAttrs: echoVideoAttributes, StoredImp: storedImps[impData.ImpExtPrebid.StoredRequest.ID], Passthrough: passthrough}

		} else {
			resolvedImps = append(resolvedImps, impData.Imp)
			impId, err := jsonparser.GetString(impData.Imp, "id")
			if err != nil {
				if err == jsonparser.KeyPathNotFoundError {
					err = fmt.Errorf("request.imp[%d] missing required field: \"id\"\n", i)
				}
				return nil, nil, []error{err}
			}
			impExtInfoMap[impId] = exchange.ImpExtInfo{Passthrough: impData.ImpExtPrebid.Passthrough}
		}
	}
	if len(resolvedImps) > 0 {
		newImpJson, err := json.Marshal(resolvedImps)
		if err != nil {
			return nil, nil, []error{err}
		}
		resolvedRequest, err = jsonparser.Set(resolvedRequest, newImpJson, "imp")
		if err != nil {
			return nil, nil, []error{err}
		}
	}

	return resolvedRequest, impExtInfoMap, nil
}

// parseImpInfo parses the request JSON and returns impression and unmarshalled imp.ext.prebid
func parseImpInfo(requestJson []byte) (impData []ImpExtPrebidData, errs []error) {
	if impArray, dataType, _, err := jsonparser.Get(requestJson, "imp"); err == nil && dataType == jsonparser.Array {
		_, err = jsonparser.ArrayEach(impArray, func(imp []byte, _ jsonparser.ValueType, _ int, err error) {
			impExtData, _, _, err := jsonparser.Get(imp, "ext", "prebid")
			var impExtPrebid openrtb_ext.ExtImpPrebid
			if impExtData != nil {
				if err := json.Unmarshal(impExtData, &impExtPrebid); err != nil {
					errs = append(errs, err)
				}
			}
			newImpData := ImpExtPrebidData{imp, impExtPrebid}
			impData = append(impData, newImpData)
		})
	}
	return
}

type ImpExtPrebidData struct {
	Imp          json.RawMessage
	ImpExtPrebid openrtb_ext.ExtImpPrebid
}

// getStoredRequestId parses a Stored Request ID from some json, without doing a full (slow) unmarshal.
// It returns the ID, true/false whether a stored request key existed, and an error if anything went wrong
// (e.g. malformed json, id not a string, etc).
func getStoredRequestId(data []byte) (string, bool, error) {
	// These keys must be kept in sync with openrtb_ext.ExtStoredRequest
	storedRequestId, dataType, _, err := jsonparser.Get(data, "ext", openrtb_ext.PrebidExtKey, "storedrequest", "id")

	if dataType == jsonparser.NotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if dataType != jsonparser.String {
		return "", true, errors.New("ext.prebid.storedrequest.id must be a string")
	}
	return string(storedRequestId), true, nil
}

func getBidRequestID(data json.RawMessage) (string, error) {
	bidRequestID, dataType, _, err := jsonparser.Get(data, "id")
	if dataType == jsonparser.NotExist {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(bidRequestID), nil
}

// setIPImplicitly sets the IP address on bidReq, if it's not explicitly defined and we can figure it out.
func setIPImplicitly(httpReq *http.Request, r *openrtb_ext.RequestWrapper, ipValidator iputil.IPValidator) {
	if r.Device == nil || (r.Device.IP == "" && r.Device.IPv6 == "") {
		if ip, ver := httputil.FindIP(httpReq, ipValidator); ip != nil {
			switch ver {
			case iputil.IPv4:
				if r.Device == nil {
					r.Device = &openrtb2.Device{}
				}
				r.Device.IP = ip.String()
			case iputil.IPv6:
				if r.Device == nil {
					r.Device = &openrtb2.Device{}
				}
				r.Device.IPv6 = ip.String()
			}
		}
	}
}

// setUAImplicitly sets the User Agent on bidReq, if it's not explicitly defined and it's defined on the request.
func setUAImplicitly(httpReq *http.Request, r *openrtb_ext.RequestWrapper) {
	if r.Device == nil || r.Device.UA == "" {
		if ua := httpReq.UserAgent(); ua != "" {
			if r.Device == nil {
				r.Device = &openrtb2.Device{}
			}
			r.Device.UA = ua
		}
	}
}

func setDoNotTrackImplicitly(httpReq *http.Request, r *openrtb_ext.RequestWrapper) {
	if r.Device == nil || r.Device.DNT == nil {
		dnt := httpReq.Header.Get(dntKey)
		if dnt == "0" || dnt == "1" {
			if r.Device == nil {
				r.Device = &openrtb2.Device{}
			}

			switch dnt {
			case "0":
				r.Device.DNT = &dntDisabled
			case "1":
				r.Device.DNT = &dntEnabled
			}
		}
	}
}

// Write(return) errors to the client, if any. Returns true if errors were found.
func writeError(errs []error, w http.ResponseWriter, labels *metrics.Labels) bool {
	var rc bool = false
	if len(errs) > 0 {
		httpStatus := http.StatusBadRequest
		metricsStatus := metrics.RequestStatusBadInput
		for _, err := range errs {
			erVal := errortypes.ReadCode(err)
			if erVal == errortypes.BlacklistedAppErrorCode || erVal == errortypes.BlacklistedAcctErrorCode {
				httpStatus = http.StatusServiceUnavailable
				metricsStatus = metrics.RequestStatusBlacklisted
				break
			} else if erVal == errortypes.MalformedAcctErrorCode {
				httpStatus = http.StatusInternalServerError
				metricsStatus = metrics.RequestStatusAccountConfigErr
				break
			}
		}
		w.WriteHeader(httpStatus)
		labels.RequestStatus = metricsStatus
		for _, err := range errs {
			w.Write([]byte(fmt.Sprintf("Invalid request: %s\n", err.Error())))
		}
		rc = true
	}
	return rc
}

// Returns the account ID for the request
func getAccountID(pub *openrtb2.Publisher) string {
	if pub != nil {
		if pub.Ext != nil {
			var pubExt openrtb_ext.ExtPublisher
			err := json.Unmarshal(pub.Ext, &pubExt)
			if err == nil && pubExt.Prebid != nil && pubExt.Prebid.ParentAccount != nil && *pubExt.Prebid.ParentAccount != "" {
				return *pubExt.Prebid.ParentAccount
			}
		}
		if pub.ID != "" {
			return pub.ID
		}
	}
	return metrics.PublisherUnknown
}

func getAccountIdFromRawRequest(hasStoredRequest bool, storedRequest json.RawMessage, originalRequest []byte) (string, bool, []error) {
	request := originalRequest
	if hasStoredRequest {
		request = storedRequest
	}

	accountId, isAppReq, err := searchAccountId(request)
	if err != nil {
		return "", isAppReq, []error{err}
	}

	// In case the stored request did not have account data we specifically search it in the original request
	if accountId == "" && hasStoredRequest {
		accountId, _, err = searchAccountId(originalRequest)
		if err != nil {
			return "", isAppReq, []error{err}
		}
	}

	if accountId == "" {
		return metrics.PublisherUnknown, isAppReq, nil
	}

	return accountId, isAppReq, nil
}

func searchAccountId(request []byte) (string, bool, error) {
	for _, path := range accountIdSearchPath {
		accountId, exists, err := getStringValueFromRequest(request, path.key)
		if err != nil {
			return "", path.isApp, err
		}
		if exists {
			return accountId, path.isApp, nil
		}
	}
	return "", false, nil
}

func getStringValueFromRequest(request []byte, key []string) (string, bool, error) {
	val, dataType, _, err := jsonparser.Get(request, key...)
	if dataType == jsonparser.NotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if dataType != jsonparser.String {
		return "", true, fmt.Errorf("%s must be a string", strings.Join(key, "."))
	}
	return string(val), true, nil
}

func storedRequestErrorChecker(requestJson []byte, storedRequests map[string]json.RawMessage, storedBidRequestId string) []error {
	if hasErr, syntaxErr := getJsonSyntaxError(requestJson); hasErr {
		return []error{fmt.Errorf("Invalid JSON in Incoming Request: %s", syntaxErr)}
	}
	if hasErr, syntaxErr := getJsonSyntaxError(storedRequests[storedBidRequestId]); hasErr {
		return []error{fmt.Errorf("ext.prebid.storedrequest.id refers to Stored Request %s which contains Invalid JSON: %s", storedBidRequestId, syntaxErr)}
	}
	return nil
}

func generateUuidForBidRequest(uuidGenerator uuidutil.UUIDGenerator) ([]byte, error) {
	newBidRequestID, err := uuidGenerator.Generate()
	if err != nil {
		return nil, err
	}
	return []byte(`{"id":"` + newBidRequestID + `"}`), nil
}

func (deps *endpointDeps) setIntegrationType(req *openrtb_ext.RequestWrapper, account *config.Account) error {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return err
	}
	reqPrebid := reqExt.GetPrebid()

	if account == nil || account.DefaultIntegration == "" {
		return nil
	}
	if reqPrebid == nil {
		reqPrebid = &openrtb_ext.ExtRequestPrebid{Integration: account.DefaultIntegration}
		reqExt.SetPrebid(reqPrebid)
	} else if reqPrebid.Integration == "" {
		reqPrebid.Integration = account.DefaultIntegration
		reqExt.SetPrebid(reqPrebid)
	}
	return nil
}

func checkIfAppRequest(request []byte) (bool, error) {
	requestApp, dataType, _, err := jsonparser.Get(request, "app")
	if dataType == jsonparser.NotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if requestApp != nil {
		return true, nil
	}
	return false, nil
}

func validateStoredBidRespAndImpExtBidders(bidderExts map[string]json.RawMessage, storedBidResp stored_responses.ImpBidderStoredResp, impId string) error {
	if bidResponses, ok := storedBidResp[impId]; ok {
		if len(bidResponses) != len(bidderExts) {
			return generateStoredBidResponseValidationError(impId)
		}

		for bidderName := range bidResponses {
			if _, present := bidderExts[bidderName]; !present {
				return generateStoredBidResponseValidationError(impId)
			}
		}
	}
	return nil
}

func generateStoredBidResponseValidationError(impID string) error {
	return fmt.Errorf("request validation failed. Stored bid responses are specified for imp %s. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse", impID)
}
