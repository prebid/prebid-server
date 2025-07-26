package openrtb2

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/privacy"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	accountService "github.com/prebid/prebid-server/v3/account"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/iputil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/prebid/prebid-server/v3/util/uuidutil"
	"github.com/prebid/prebid-server/v3/version"
)

var defaultRequestTimeout int64 = 5000

func NewVideoEndpoint(
	uuidGenerator uuidutil.UUIDGenerator,
	ex exchange.Exchange,
	requestValidator ortb.RequestValidator,
	requestsById stored_requests.Fetcher,
	videoFetcher stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	cfg *config.Configuration,
	met metrics.MetricsEngine,
	analyticsRunner analytics.Runner,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName,
	cache prebid_cache_client.Client,
	tmaxAdjustments *exchange.TmaxAdjustmentsPreprocessed,
) (httprouter.Handle, error) {

	if ex == nil || requestValidator == nil || requestsById == nil || accounts == nil || cfg == nil || met == nil {
		return nil, errors.New("NewVideoEndpoint requires non-nil arguments.")
	}

	defRequest := len(defReqJSON) > 0

	ipValidator := iputil.PublicNetworkIPValidator{
		IPv4PrivateNetworks: cfg.RequestValidation.IPv4PrivateNetworksParsed,
		IPv6PrivateNetworks: cfg.RequestValidation.IPv6PrivateNetworksParsed,
	}

	videoEndpointRegexp := regexp.MustCompile(`[<>]`)

	return httprouter.Handle((&endpointDeps{
		uuidGenerator,
		ex,
		requestValidator,
		requestsById,
		videoFetcher,
		accounts,
		cfg,
		met,
		analyticsRunner,
		disabledBidders,
		defRequest,
		defReqJSON,
		bidderMap,
		cache,
		videoEndpointRegexp,
		ipValidator,
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		tmaxAdjustments,
		openrtb_ext.NormalizeBidderName}).VideoAuctionEndpoint), nil
}

/*
 1. Parse "storedrequestid" field from simplified endpoint request body.
 2. If config flag to require that field is set (which it will be for us) and this field is not given then error out here.
 3. Load the stored request JSON for the given storedrequestid, if the id was invalid then error out here.
 4. Use "json-patch" 3rd party library to merge the request body JSON data into the stored request JSON data.
 5. Unmarshal the merged JSON data into a Go structure.
 6. Add fields from merged JSON data that correspond to an OpenRTB request into the OpenRTB bid request we are building.
    a. Unmarshal certain OpenRTB defined structs directly into the OpenRTB bid request.
    b. In cases where customized logic is needed just copy/fill the fields in directly.
 7. Call setFieldsImplicitly from auction.go to get basic data from the HTTP request into an OpenRTB bid request to start building the OpenRTB bid request.
 8. Loop through ad pods to build array of Imps into OpenRTB request, for each pod:
    a. Load the stored impression to use as the basis for impressions generated for this pod from the configid field.
    b. NumImps = adpoddurationsec / MIN_VALUE(allowedDurations)
    c. Build impression array for this pod:
    I.Create array of NumImps entries initialized to the base impression loaded from the configid.
 1. If requireexactdurations = true, iterate over allowdDurations and for (NumImps / len(allowedDurations)) number of Imps set minduration = maxduration = allowedDurations[i]
 2. If requireexactdurations = false, set maxduration = MAX_VALUE(allowedDurations)
    II. Set Imp.id field to "podX_Y" where X is the pod index and Y is the impression index within this pod.
    d. Append impressions for this pod to the overall list of impressions in the OpenRTB bid request.
 9. Call validateRequest() function from auction.go to validate the generated request.
 10. Call HoldAuction() function to run the auction for the OpenRTB bid request that was built in the previous step.
 11. Build proper response format.
*/
func (deps *endpointDeps) VideoAuctionEndpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	start := time.Now()

	vo := analytics.VideoObject{
		Status:    http.StatusOK,
		Errors:    make([]error, 0),
		StartTime: start,
	}

	labels := metrics.Labels{
		Source:        metrics.DemandUnknown,
		RType:         metrics.ReqTypeVideo,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
	}

	debugQuery := r.URL.Query().Get("debug")
	cacheTTL := int64(3600)
	if deps.cfg.CacheURL.DefaultTTLs.Video > 0 {
		cacheTTL = int64(deps.cfg.CacheURL.DefaultTTLs.Video)
	}
	debugLog := exchange.DebugLog{
		Enabled:       strings.EqualFold(debugQuery, "true"),
		CacheType:     prebid_cache_client.TypeXML,
		TTL:           cacheTTL,
		Regexp:        deps.debugLogRegexp,
		DebugOverride: exchange.IsDebugOverrideEnabled(r.Header.Get(exchange.DebugOverrideHeader), deps.cfg.Debug.OverrideToken),
	}
	debugLog.DebugEnabledOrOverridden = debugLog.Enabled || debugLog.DebugOverride

	activityControl := privacy.ActivityControl{}

	defer func() {
		if len(debugLog.CacheKey) > 0 && vo.VideoResponse == nil {
			err := debugLog.PutDebugLogError(deps.cache, deps.cfg.CacheURL.ExpectedTimeMillis, vo.Errors)
			if err != nil {
				vo.Errors = append(vo.Errors, err)
			}
		}
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogVideoObject(&vo, activityControl)
	}()

	w.Header().Set("X-Prebid", version.BuildXPrebidHeader(version.Ver))
	setBrowsingTopicsHeader(w, r)

	lr := &io.LimitedReader{
		R: r.Body,
		N: deps.cfg.MaxRequestSize,
	}
	requestJson, err := io.ReadAll(lr)
	if err != nil {
		handleError(&labels, w, []error{err}, &vo, &debugLog)
		return
	}

	resolvedRequest := requestJson
	if debugLog.DebugEnabledOrOverridden {
		debugLog.Data.Request = string(requestJson)
		if headerBytes, err := jsonutil.Marshal(r.Header); err == nil {
			debugLog.Data.Headers = string(headerBytes)
		} else {
			debugLog.Data.Headers = fmt.Sprintf("Unable to marshal headers data: %s", err.Error())
		}
	}

	//load additional data - stored simplified req
	storedRequestId, err := getVideoStoredRequestId(requestJson)

	if err != nil {
		if deps.cfg.VideoStoredRequestRequired {
			handleError(&labels, w, []error{err}, &vo, &debugLog)
			return
		}
	} else {
		storedRequest, errs := deps.loadStoredVideoRequest(context.Background(), storedRequestId)
		if len(errs) > 0 {
			handleError(&labels, w, errs, &vo, &debugLog)
			return
		}

		//merge incoming req with stored video req
		resolvedRequest, err = jsonpatch.MergePatch(storedRequest, requestJson)
		if err != nil {
			handleError(&labels, w, []error{err}, &vo, &debugLog)
			return
		}
	}
	//unmarshal and validate combined result
	videoBidReq, errL, podErrors := deps.parseVideoRequest(resolvedRequest, r.Header)
	if len(errL) > 0 {
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}

	vo.VideoRequest = videoBidReq

	var bidReq = &openrtb2.BidRequest{}
	if deps.defaultRequest {
		if err := jsonutil.UnmarshalValid(deps.defReqJSON, bidReq); err != nil {
			err = fmt.Errorf("Invalid JSON in Default Request Settings: %s", err)
			handleError(&labels, w, []error{err}, &vo, &debugLog)
			return
		}
	}

	//create full open rtb req from full video request
	mergeData(videoBidReq, bidReq)
	// If debug query param is set, force the response to enable test flag
	if debugLog.DebugEnabledOrOverridden {
		bidReq.Test = 1
	}

	initialPodNumber := len(videoBidReq.PodConfig.Pods)
	if len(podErrors) > 0 {
		//remove incorrect pods
		videoBidReq = cleanupVideoBidRequest(videoBidReq, podErrors)
	}

	//create impressions array
	imps, podErrors := deps.createImpressions(videoBidReq, podErrors)

	if len(podErrors) == initialPodNumber {
		resPodErr := make([]string, 0)
		for _, podEr := range podErrors {
			resPodErr = append(resPodErr, strings.Join(podEr.ErrMsgs, ", "))
		}
		err := fmt.Errorf("all pods are incorrect: %s", strings.Join(resPodErr, "; "))
		errL = append(errL, err)
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}

	bidReq.Imp = imps
	bidReq.ID = "bid_id" //TODO: look at prebid.js

	// all code after this line should use the bidReqWrapper instead of bidReq directly
	bidReqWrapper := &openrtb_ext.RequestWrapper{BidRequest: bidReq}

	if err := openrtb_ext.ConvertUpTo26(bidReqWrapper); err != nil {
		handleError(&labels, w, []error{err}, &vo, &debugLog)
		return
	}

	if err := ortb.SetDefaults(bidReqWrapper, deps.cfg.TmaxDefault); err != nil {
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}

	ctx := context.Background()
	timeout := deps.cfg.AuctionTimeouts.LimitAuctionTimeout(time.Duration(bidReqWrapper.TMax) * time.Millisecond)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, start.Add(timeout))
		defer cancel()
	}

	// Read Usersyncs/Cookie
	decoder := usersync.Base64Decoder{}
	usersyncs := usersync.ReadCookie(r, decoder, &deps.cfg.HostCookie)
	usersync.SyncHostCookie(r, usersyncs, &deps.cfg.HostCookie)

	if bidReqWrapper.App != nil {
		labels.Source = metrics.DemandApp
		labels.PubID = getAccountID(bidReqWrapper.App.Publisher)
	} else { // both bidReqWrapper.App == nil and bidReqWrapper.Site != nil are true
		labels.Source = metrics.DemandWeb
		if usersyncs.HasAnyLiveSyncs() {
			labels.CookieFlag = metrics.CookieFlagYes
		} else {
			labels.CookieFlag = metrics.CookieFlagNo
		}
		labels.PubID = getAccountID(bidReqWrapper.Site.Publisher)
	}

	// Look up account now that we have resolved the pubID value
	account, acctIDErrs := accountService.GetAccount(ctx, deps.cfg, deps.accounts, labels.PubID, deps.metricsEngine)
	if len(acctIDErrs) > 0 {
		handleError(&labels, w, acctIDErrs, &vo, &debugLog)
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	if errs := deps.setFieldsImplicitly(r, bidReqWrapper, account); len(errs) > 0 {
		errL = append(errL, errs...)
	}

	errs := deps.validateRequest(account, r, bidReqWrapper, false, false, nil, false)
	errL = append(errL, errs...)
	if errortypes.ContainsFatalError(errL) {
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}

	activityControl = privacy.NewActivityControl(&account.Privacy)

	warnings := errortypes.WarningOnly(errL)

	secGPC := r.Header.Get("Sec-GPC")
	auctionRequest := &exchange.AuctionRequest{
		BidRequestWrapper:          bidReqWrapper,
		Account:                    *account,
		UserSyncs:                  usersyncs,
		RequestType:                labels.RType,
		StartTime:                  start,
		LegacyLabels:               labels,
		Warnings:                   warnings,
		GlobalPrivacyControlHeader: secGPC,
		PubID:                      labels.PubID,
		HookExecutor:               hookexecution.EmptyHookExecutor{},
		TmaxAdjustments:            deps.tmaxAdjustments,
		Activities:                 activityControl,
	}

	auctionResponse, err := deps.ex.HoldAuction(ctx, auctionRequest, &debugLog)
	defer func() {
		if !auctionRequest.BidderResponseStartTime.IsZero() {
			deps.metricsEngine.RecordOverheadTime(metrics.MakeAuctionResponse, time.Since(auctionRequest.BidderResponseStartTime))
		}
	}()
	vo.RequestWrapper = bidReqWrapper
	var response *openrtb2.BidResponse
	if auctionResponse != nil {
		response = auctionResponse.BidResponse
	}
	vo.Response = response
	vo.SeatNonBid = auctionResponse.GetSeatNonBid()
	if err != nil {
		errL := []error{err}
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}

	//build simplified response
	bidResp, err := buildVideoResponse(response, podErrors)
	if err != nil {
		errL := []error{err}
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}
	if bidReq.Test == 1 {
		err = setSeatNonBidRaw(bidReqWrapper, auctionResponse)
		if err != nil {
			glog.Errorf("Error setting seat non-bid: %v", err)
		}
		bidResp.Ext = response.Ext
	}

	if len(bidResp.AdPods) == 0 && debugLog.DebugEnabledOrOverridden {
		err := debugLog.PutDebugLogError(deps.cache, deps.cfg.CacheURL.ExpectedTimeMillis, vo.Errors)
		if err != nil {
			vo.Errors = append(vo.Errors, err)
		} else {
			bidResp.AdPods = append(bidResp.AdPods, &openrtb_ext.AdPod{
				Targeting: []openrtb_ext.VideoTargeting{
					{
						HbCacheID: debugLog.CacheKey,
					},
				},
			})
		}
	}

	vo.VideoResponse = bidResp

	resp, err := jsonutil.Marshal(bidResp)
	if err != nil {
		errL := []error{err}
		handleError(&labels, w, errL, &vo, &debugLog)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func cleanupVideoBidRequest(videoReq *openrtb_ext.BidRequestVideo, podErrors []PodError) *openrtb_ext.BidRequestVideo {
	for i := len(podErrors) - 1; i >= 0; i-- {
		videoReq.PodConfig.Pods = append(videoReq.PodConfig.Pods[:podErrors[i].PodIndex], videoReq.PodConfig.Pods[podErrors[i].PodIndex+1:]...)
	}
	return videoReq
}

func handleError(labels *metrics.Labels, w http.ResponseWriter, errL []error, vo *analytics.VideoObject, debugLog *exchange.DebugLog) {
	if debugLog != nil && debugLog.DebugEnabledOrOverridden {
		if rawUUID, err := uuid.NewV4(); err == nil {
			debugLog.CacheKey = rawUUID.String()
		}
		errL = append(errL, fmt.Errorf("[Debug cache ID: %s]", debugLog.CacheKey))
	}
	labels.RequestStatus = metrics.RequestStatusErr
	var errors string
	var status int = http.StatusInternalServerError
	for _, er := range errL {
		erVal := errortypes.ReadCode(er)
		if erVal == errortypes.BlockedAppErrorCode || erVal == errortypes.AccountDisabledErrorCode {
			status = http.StatusServiceUnavailable
			labels.RequestStatus = metrics.RequestStatusBlockedApp
			break
		} else if erVal == errortypes.AcctRequiredErrorCode {
			status = http.StatusBadRequest
			labels.RequestStatus = metrics.RequestStatusBadInput
			break
		} else if erVal == errortypes.MalformedAcctErrorCode {
			status = http.StatusInternalServerError
			labels.RequestStatus = metrics.RequestStatusAccountConfigErr
			break
		}
		errors = fmt.Sprintf("%s %s", errors, er.Error())
	}
	w.WriteHeader(status)
	vo.Status = status
	fmt.Fprintf(w, "Critical error while running the video endpoint: %v", errors)
	glog.Errorf("/openrtb2/video Critical error: %v", errors)
	vo.Errors = append(vo.Errors, errL...)
}

func (deps *endpointDeps) createImpressions(videoReq *openrtb_ext.BidRequestVideo, podErrors []PodError) ([]openrtb2.Imp, []PodError) {
	videoDur := videoReq.PodConfig.DurationRangeSec
	minDuration, maxDuration := minMax(videoDur)
	reqExactDur := videoReq.PodConfig.RequireExactDuration
	videoData := videoReq.Video

	finalImpsArray := make([]openrtb2.Imp, 0)
	for ind, pod := range videoReq.PodConfig.Pods {

		//load stored impression
		storedImpressionId := string(pod.ConfigId)
		storedImp, errs := deps.loadStoredImp(storedImpressionId)
		if errs != nil {
			err := fmt.Sprintf("unable to load configid %s, Pod id: %d", storedImpressionId, pod.PodId)
			podErr := PodError{}
			podErr.PodId = pod.PodId
			podErr.PodIndex = ind
			podErr.ErrMsgs = append(podErr.ErrMsgs, err)
			podErrors = append(podErrors, podErr)
			continue
		}

		numImps := pod.AdPodDurationSec / minDuration
		if reqExactDur {
			// In case of impressions number is less than durations array, we bump up impressions number up to duration array size
			// with this handler we will have one impression per specified duration
			numImps = max(numImps, len(videoDur))
		}
		impDivNumber := numImps / len(videoDur)

		impsArray := make([]openrtb2.Imp, numImps)
		for impInd := range impsArray {
			newImp := createImpressionTemplate(storedImp, videoData)
			impsArray[impInd] = newImp
			if reqExactDur {
				//floor := int(math.Floor(ind/impDivNumber))
				durationIndex := impInd / impDivNumber
				if durationIndex > len(videoDur)-1 {
					durationIndex = len(videoDur) - 1
				}
				impsArray[impInd].Video.MaxDuration = int64(videoDur[durationIndex])
				impsArray[impInd].Video.MinDuration = int64(videoDur[durationIndex])
				//fmt.Println("Imp ind  ", impInd, "duration ", videoDur[durationIndex])
			} else {
				impsArray[impInd].Video.MaxDuration = int64(maxDuration)
			}

			impsArray[impInd].ID = fmt.Sprintf("%d_%d", pod.PodId, impInd)
		}
		finalImpsArray = append(finalImpsArray, impsArray...)

	}
	return finalImpsArray, podErrors
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func createImpressionTemplate(imp openrtb2.Imp, video *openrtb2.Video) openrtb2.Imp {
	//for every new impression we need to have it's own copy of video object, because we customize it in further processing
	newVideo := *video
	imp.Video = &newVideo
	return imp
}

func (deps *endpointDeps) loadStoredImp(storedImpId string) (openrtb2.Imp, []error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deps.cfg.StoredRequestsTimeout)*time.Millisecond)
	defer cancel()

	impr := openrtb2.Imp{}
	_, imp, err := deps.storedReqFetcher.FetchRequests(ctx, []string{}, []string{storedImpId})
	if err != nil {
		return impr, err
	}

	if err := jsonutil.UnmarshalValid(imp[storedImpId], &impr); err != nil {
		return impr, []error{err}
	}
	return impr, nil
}

func minMax(array []int) (int, int) {
	var max = array[0]
	var min = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func buildVideoResponse(bidresponse *openrtb2.BidResponse, podErrors []PodError) (*openrtb_ext.BidResponseVideo, error) {

	adPods := make([]*openrtb_ext.AdPod, 0)
	anyBidsReturned := false
	for _, seatBid := range bidresponse.SeatBid {
		for _, bid := range seatBid.Bid {
			anyBidsReturned = true

			var tempRespBidExt openrtb_ext.ExtBid
			if err := jsonutil.UnmarshalValid(bid.Ext, &tempRespBidExt); err != nil {
				return nil, err
			}
			if findTargetingByKey(tempRespBidExt.Prebid.Targeting, formatTargetingKey(openrtb_ext.VastCacheKey, seatBid.Seat)) == "" {
				continue
			}

			impId := bid.ImpID
			podNum := strings.Split(impId, "_")[0]
			podId, _ := strconv.ParseInt(podNum, 0, 64)

			videoTargeting := openrtb_ext.VideoTargeting{
				HbPb:       findTargetingByKey(tempRespBidExt.Prebid.Targeting, formatTargetingKey(openrtb_ext.PbKey, seatBid.Seat)),
				HbPbCatDur: findTargetingByKey(tempRespBidExt.Prebid.Targeting, formatTargetingKey(openrtb_ext.CategoryDurationKey, seatBid.Seat)),
				HbCacheID:  findTargetingByKey(tempRespBidExt.Prebid.Targeting, formatTargetingKey(openrtb_ext.VastCacheKey, seatBid.Seat)),
				HbDeal:     findTargetingByKey(tempRespBidExt.Prebid.Targeting, formatTargetingKey(openrtb_ext.DealKey, seatBid.Seat)),
			}

			adPod := findAdPod(podId, adPods)
			if adPod == nil {
				adPod = &openrtb_ext.AdPod{
					PodId:     podId,
					Targeting: make([]openrtb_ext.VideoTargeting, 0),
				}
				adPods = append(adPods, adPod)
			}
			adPod.Targeting = append(adPod.Targeting, videoTargeting)

		}
	}

	//check if there are any bids in response.
	//if there are no bids - empty response should be returned, no cache errors
	if len(adPods) == 0 && anyBidsReturned {
		//means there is a global cache error, we need to reject all bids
		err := errors.New("caching failed for all bids")
		return nil, err
	}

	// If there were incorrect pods, we put them back to response with error message
	if len(podErrors) > 0 {
		for _, podEr := range podErrors {
			adPodEr := &openrtb_ext.AdPod{
				PodId:  int64(podEr.PodId),
				Errors: podEr.ErrMsgs,
			}
			adPods = append(adPods, adPodEr)
		}
	}

	return &openrtb_ext.BidResponseVideo{AdPods: adPods}, nil
}

func formatTargetingKey(key openrtb_ext.TargetingKey, bidderName string) string {
	fullKey := fmt.Sprintf("%s_%s", string(key), bidderName)
	if len(fullKey) > exchange.MaxKeyLength {
		return string(fullKey[0:exchange.MaxKeyLength])
	}
	return fullKey
}

func findTargetingByKey(targetingMap map[string]string, keyWithoutPrefix string) string {
	for k, v := range targetingMap {
		prefixIndex := strings.Index(k, "_")
		// find potentially truncated key in original key name without prefixes
		if prefixIndex > 0 && strings.HasPrefix(keyWithoutPrefix, k[prefixIndex:]) {
			return v
		}
	}
	return ""
}

func findAdPod(podInd int64, pods []*openrtb_ext.AdPod) *openrtb_ext.AdPod {
	for _, pod := range pods {
		if pod.PodId == podInd {
			return pod
		}
	}
	return nil
}

func (deps *endpointDeps) loadStoredVideoRequest(ctx context.Context, storedRequestId string) ([]byte, []error) {
	storedRequests, _, errs := deps.videoFetcher.FetchRequests(ctx, []string{storedRequestId}, []string{})
	jsonString := storedRequests[storedRequestId]
	return jsonString, errs
}

func getVideoStoredRequestId(request []byte) (string, error) {
	value, dataType, _, err := jsonparser.Get(request, "storedrequestid")
	if dataType != jsonparser.String || err != nil {
		return "", &errortypes.BadInput{Message: "Unable to find required stored request id"}
	}
	return string(value), nil
}

func mergeData(videoRequest *openrtb_ext.BidRequestVideo, bidRequest *openrtb2.BidRequest) error {
	if videoRequest.Site != nil {
		bidRequest.Site = videoRequest.Site
		bidRequest.Site.Content = &videoRequest.Content
	}

	if videoRequest.App != nil {
		bidRequest.App = videoRequest.App
		bidRequest.App.Content = &videoRequest.Content
	}

	bidRequest.Device = &videoRequest.Device
	bidRequest.User = videoRequest.User

	if len(videoRequest.BCat) != 0 {
		bidRequest.BCat = videoRequest.BCat
	}

	if len(videoRequest.BAdv) != 0 {
		bidRequest.BAdv = videoRequest.BAdv
	}

	bidExt, err := createBidExtension(videoRequest)
	if err != nil {
		return err
	}
	if len(bidExt) > 0 {
		bidRequest.Ext = bidExt
	}

	bidRequest.Test = videoRequest.Test

	if videoRequest.TMax == 0 {
		bidRequest.TMax = defaultRequestTimeout
	} else {
		bidRequest.TMax = videoRequest.TMax
	}

	if videoRequest.Regs != nil {
		bidRequest.Regs = videoRequest.Regs
	}

	return nil
}

func createBidExtension(videoRequest *openrtb_ext.BidRequestVideo) ([]byte, error) {
	var inclBrandCat *openrtb_ext.ExtIncludeBrandCategory
	if videoRequest.IncludeBrandCategory != nil {
		inclBrandCat = &openrtb_ext.ExtIncludeBrandCategory{
			PrimaryAdServer:     videoRequest.IncludeBrandCategory.PrimaryAdserver,
			Publisher:           videoRequest.IncludeBrandCategory.Publisher,
			WithCategory:        true,
			TranslateCategories: videoRequest.IncludeBrandCategory.TranslateCategories,
		}
	} else {
		inclBrandCat = &openrtb_ext.ExtIncludeBrandCategory{
			WithCategory: false,
		}
	}

	var durationRangeSec []int
	if !videoRequest.PodConfig.RequireExactDuration {
		durationRangeSec = videoRequest.PodConfig.DurationRangeSec
	}

	targeting := openrtb_ext.ExtRequestTargeting{
		PriceGranularity:     videoRequest.PriceGranularity,
		IncludeBrandCategory: inclBrandCat,
		DurationRangeSec:     durationRangeSec,
		IncludeBidderKeys:    ptrutil.ToPtr(true),
		AppendBidderNames:    videoRequest.AppendBidderNames,
	}

	vastXml := openrtb_ext.ExtRequestPrebidCacheVAST{}
	cache := openrtb_ext.ExtRequestPrebidCache{
		VastXML: &vastXml,
	}

	prebid := openrtb_ext.ExtRequestPrebid{
		Cache:        &cache,
		Targeting:    &targeting,
		SupportDeals: videoRequest.SupportDeals,
	}
	extReq := openrtb_ext.ExtRequest{Prebid: prebid}

	return jsonutil.Marshal(extReq)
}

func (deps *endpointDeps) parseVideoRequest(request []byte, headers http.Header) (req *openrtb_ext.BidRequestVideo, errs []error, podErrors []PodError) {
	req = &openrtb_ext.BidRequestVideo{}

	if err := jsonutil.UnmarshalValid(request, &req); err != nil {
		errs = []error{err}
		return
	}

	//if Device.UA is not present in request body, init it with user-agent from request header if it's present
	if req.Device.UA == "" {
		ua := headers.Get("User-Agent")

		//Check UA is encoded. Without it the `+` character would get changed to a space if not actually encoded
		if strings.ContainsAny(ua, "%") {
			var err error
			req.Device.UA, err = url.QueryUnescape(ua)
			if err != nil {
				req.Device.UA = ua
			}
		} else {
			req.Device.UA = ua
		}
	}

	errL, podErrors := deps.validateVideoRequest(req)
	if len(errL) > 0 {
		errs = append(errs, errL...)
	}
	return
}

type PodError struct {
	PodId    int
	PodIndex int
	ErrMsgs  []string
}

func (deps *endpointDeps) validateVideoRequest(req *openrtb_ext.BidRequestVideo) ([]error, []PodError) {
	errL := []error{}

	if deps.cfg.VideoStoredRequestRequired && req.StoredRequestId == "" {
		err := errors.New("request missing required field: storedrequestid")
		errL = append(errL, err)
	}
	if len(req.PodConfig.DurationRangeSec) == 0 {
		err := errors.New("request missing required field: PodConfig.DurationRangeSec")
		errL = append(errL, err)
	}
	if isZeroOrNegativeDuration(req.PodConfig.DurationRangeSec) {
		err := errors.New("duration array cannot contain negative or zero values")
		errL = append(errL, err)
	}
	if len(req.PodConfig.Pods) == 0 {
		err := errors.New("request missing required field: PodConfig.Pods")
		errL = append(errL, err)
	}
	podErrors := make([]PodError, 0)
	podIdsSet := make(map[int]bool)
	for ind, pod := range req.PodConfig.Pods {
		podErr := PodError{}

		if podIdsSet[pod.PodId] {
			err := fmt.Sprintf("request duplicated required field: PodConfig.Pods.PodId, Pod id: %d", pod.PodId)
			podErr.ErrMsgs = append(podErr.ErrMsgs, err)
		} else {
			podIdsSet[pod.PodId] = true
		}
		if pod.PodId <= 0 {
			err := fmt.Sprintf("request missing required field: PodConfig.Pods.PodId, Pod index: %d", ind)
			podErr.ErrMsgs = append(podErr.ErrMsgs, err)
		}
		if pod.AdPodDurationSec == 0 {
			err := fmt.Sprintf("request missing or incorrect required field: PodConfig.Pods.AdPodDurationSec, Pod index: %d", ind)
			podErr.ErrMsgs = append(podErr.ErrMsgs, err)
		}
		if pod.AdPodDurationSec < 0 {
			err := fmt.Sprintf("request incorrect required field: PodConfig.Pods.AdPodDurationSec is negative, Pod index: %d", ind)
			podErr.ErrMsgs = append(podErr.ErrMsgs, err)
		}
		if pod.ConfigId == "" {
			err := fmt.Sprintf("request missing or incorrect required field: PodConfig.Pods.ConfigId, Pod index: %d", ind)
			podErr.ErrMsgs = append(podErr.ErrMsgs, err)
		}
		if len(podErr.ErrMsgs) > 0 {
			podErr.PodId = pod.PodId
			podErr.PodIndex = ind
			podErrors = append(podErrors, podErr)
		}
	}
	if req.App == nil && req.Site == nil {
		err := errors.New("request missing required field: site or app")
		errL = append(errL, err)
	} else if req.App != nil && req.Site != nil {
		err := errors.New("request.site or request.app must be defined, but not both")
		errL = append(errL, err)
	} else if req.Site != nil && req.Site.ID == "" && req.Site.Page == "" {
		err := errors.New("request.site missing required field: id or page")
		errL = append(errL, err)
	} else if req.App != nil {
		if req.App.ID != "" {
			if _, found := deps.cfg.BlockedAppsLookup[req.App.ID]; found {
				err := &errortypes.BlockedApp{Message: fmt.Sprintf("Prebid-server does not process requests from App ID: %s", req.App.ID)}
				errL = append(errL, err)
				return errL, podErrors
			}
		} else {
			if req.App.Bundle == "" {
				err := errors.New("request.app missing required field: id or bundle")
				errL = append(errL, err)
			}
		}
	}

	if req.Video != nil {
		if len(req.Video.MIMEs) == 0 {
			err := errors.New("request missing required field: Video.Mimes")
			errL = append(errL, err)
		} else {
			mimes := make([]string, 0, len(req.Video.MIMEs))
			for _, mime := range req.Video.MIMEs {
				if mime != "" {
					mimes = append(mimes, mime)
				}
			}
			if len(mimes) == 0 {
				err := errors.New("request missing required field: Video.Mimes, mime types contains empty strings only")
				errL = append(errL, err)
			}
			req.Video.MIMEs = mimes
		}

		if len(req.Video.Protocols) == 0 {
			err := errors.New("request missing required field: Video.Protocols")
			errL = append(errL, err)
		}
	} else {
		err := errors.New("request missing required field: Video")
		errL = append(errL, err)
	}

	return errL, podErrors
}

func isZeroOrNegativeDuration(duration []int) bool {
	for _, value := range duration {
		if value <= 0 {
			return true
		}
	}
	return false
}
