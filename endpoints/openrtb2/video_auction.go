package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/prebid/prebid-server/errortypes"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
)

var defaultRequestTimeout int64 = 5000

func NewVideoEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsById stored_requests.Fetcher, videoFetcher stored_requests.Fetcher, categories stored_requests.CategoryFetcher, cfg *config.Configuration, met pbsmetrics.MetricsEngine, pbsAnalytics analytics.PBSAnalyticsModule, disabledBidders map[string]string, defReqJSON []byte, bidderMap map[string]openrtb_ext.BidderName) (httprouter.Handle, error) {

	if ex == nil || validator == nil || requestsById == nil || cfg == nil || met == nil {
		return nil, errors.New("NewVideoEndpoint requires non-nil arguments.")
	}
	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	return httprouter.Handle((&endpointDeps{ex, validator, requestsById, videoFetcher, categories, cfg, met, pbsAnalytics, disabledBidders, defRequest, defReqJSON, bidderMap}).VideoAuctionEndpoint), nil
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

	ao := analytics.AuctionObject{
		Status: http.StatusOK,
		Errors: make([]error, 0),
	}

	start := time.Now()
	labels := pbsmetrics.Labels{
		Source:        pbsmetrics.DemandUnknown,
		RType:         pbsmetrics.ReqTypeVideo,
		PubID:         pbsmetrics.PublisherUnknown,
		Browser:       getBrowserName(r),
		CookieFlag:    pbsmetrics.CookieFlagUnknown,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}
	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogAuctionObject(&ao)
	}()

	lr := &io.LimitedReader{
		R: r.Body,
		N: deps.cfg.MaxRequestSize,
	}
	requestJson, err := ioutil.ReadAll(lr)
	if err != nil {
		handleError(labels, w, []error{err}, ao)
		return
	}

	resolvedRequest := requestJson

	//load additional data - stored simplified req
	storedRequestId, err := getVideoStoredRequestId(requestJson)

	if err != nil {
		if deps.cfg.VideoStoredRequestRequired {
			handleError(labels, w, []error{err}, ao)
			return
		}
	} else {
		storedRequest, errs := deps.loadStoredVideoRequest(context.Background(), storedRequestId)
		if len(errs) > 0 {
			handleError(labels, w, errs, ao)
			return
		}

		//merge incoming req with stored video req
		resolvedRequest, err = jsonpatch.MergePatch(storedRequest, requestJson)
		if err != nil {
			handleError(labels, w, []error{err}, ao)
			return
		}
	}
	//unmarshal and validate combined result
	videoBidReq, errL, podErrors := deps.parseVideoRequest(resolvedRequest)
	if len(errL) > 0 {
		handleError(labels, w, errL, ao)
		return
	}

	bidResp, errL := deps.executeVideoRequest(videoBidReq, podErrors, labels, w, r, ao, start)
	if len(errL) > 0 {
		handleError(labels, w, errL, ao)
		return
	}

	resp, err := json.Marshal(bidResp)
	//resp, err := json.Marshal(response)
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)

}

func (deps *endpointDeps) executeVideoRequest(videoBidReq *openrtb_ext.BidRequestVideo,
	podErrors []PodError,
	labels pbsmetrics.Labels,
	w http.ResponseWriter,
	r *http.Request,
	ao analytics.AuctionObject,
	start time.Time) (videoBidResp *openrtb_ext.BidResponseVideo, errL []error) {
	var bidReq = &openrtb.BidRequest{}
	if deps.defaultRequest {
		if err := json.Unmarshal(deps.defReqJSON, bidReq); err != nil {
			err = fmt.Errorf("Invalid JSON in Default Request Settings: %s", err)
			handleError(labels, w, []error{err}, ao)
			return nil, []error{err}
		}
	}

	//create full open rtb req from full video request
	mergeData(videoBidReq, bidReq)

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
		err := errors.New(fmt.Sprintf("all pods are incorrect: %s", strings.Join(resPodErr, "; ")))
		errL = append(errL, err)
		handleError(labels, w, errL, ao)
		return
	}

	bidReq.Imp = imps
	bidReq.ID = "bid_id" //TODO: look at prebid.js

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(r, bidReq) // move after merge

	errL = deps.validateRequest(bidReq)
	if len(errL) > 0 {
		handleError(labels, w, errL, ao)
		return
	}

	ctx := context.Background()
	timeout := deps.cfg.AuctionTimeouts.LimitAuctionTimeout(time.Duration(bidReq.TMax) * time.Millisecond)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, start.Add(timeout))
		defer cancel()
	}

	usersyncs := usersync.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie))
	if bidReq.App != nil {
		labels.Source = pbsmetrics.DemandApp
		labels.PubID = effectivePubID(bidReq.App.Publisher)
	} else { // both bidReq.App == nil and bidReq.Site != nil are true
		labels.Source = pbsmetrics.DemandWeb
		if usersyncs.LiveSyncCount() == 0 {
			labels.CookieFlag = pbsmetrics.CookieFlagNo
		} else {
			labels.CookieFlag = pbsmetrics.CookieFlagYes
		}
		labels.PubID = effectivePubID(bidReq.Site.Publisher)
	}

	if acctIdErr := validateAccount(deps.cfg, labels.PubID); acctIdErr != nil {
		errL = append(errL, acctIdErr)
		handleError(labels, w, errL, ao)
		return
	}
	//execute auction logic
	response, err := deps.ex.HoldAuction(ctx, bidReq, usersyncs, labels, &deps.categories)
	ao.Request = bidReq
	ao.Response = response
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return nil, errL
	}

	//build simplified response
	videoBidResp, err = buildVideoResponse(response, podErrors)
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return nil, errL
	}
	if bidReq.Test == 1 {
		videoBidResp.Ext = response.Ext
	}
	return
}

func cleanupVideoBidRequest(videoReq *openrtb_ext.BidRequestVideo, podErrors []PodError) *openrtb_ext.BidRequestVideo {
	for i := len(podErrors) - 1; i >= 0; i-- {
		videoReq.PodConfig.Pods = append(videoReq.PodConfig.Pods[:podErrors[i].PodIndex], videoReq.PodConfig.Pods[podErrors[i].PodIndex+1:]...)
	}
	return videoReq
}

func handleError(labels pbsmetrics.Labels, w http.ResponseWriter, errL []error, ao analytics.AuctionObject) {
	labels.RequestStatus = pbsmetrics.RequestStatusErr
	var errors string
	var status int = http.StatusInternalServerError
	for _, er := range errL {
		erVal := errortypes.DecodeError(er)
		if erVal == errortypes.BlacklistedAppCode || erVal == errortypes.BlacklistedAcctCode {
			status = http.StatusServiceUnavailable
			labels.RequestStatus = pbsmetrics.RequestStatusBlacklisted
			break
		} else if erVal == errortypes.AcctRequiredCode {
			status = http.StatusBadRequest
			labels.RequestStatus = pbsmetrics.RequestStatusBadInput
			break
		}
		errors = fmt.Sprintf("%s %s", errors, er.Error())
	}
	w.WriteHeader(status)
	ao.Status = status
	fmt.Fprintf(w, "Critical error while running the video endpoint: %v", errors)
	glog.Errorf("/openrtb2/video Critical error: %v", errors)
	ao.Errors = append(ao.Errors, errL...)
}

func (deps *endpointDeps) createImpressions(videoReq *openrtb_ext.BidRequestVideo, podErrors []PodError) ([]openrtb.Imp, []PodError) {
	videoDur := videoReq.PodConfig.DurationRangeSec
	minDuration, maxDuration := minMax(videoDur)
	reqExactDur := videoReq.PodConfig.RequireExactDuration
	videoData := videoReq.Video

	finalImpsArray := make([]openrtb.Imp, 0)
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

		impsArray := make([]openrtb.Imp, numImps)
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

func createImpressionTemplate(imp openrtb.Imp, video openrtb_ext.SimplifiedVideo) openrtb.Imp {
	imp.Video = &openrtb.Video{}
	imp.Video.W = video.W
	imp.Video.H = video.H
	imp.Video.Protocols = video.Protocols
	imp.Video.MIMEs = video.Mimes
	return imp
}

func (deps *endpointDeps) loadStoredImp(storedImpId string) (openrtb.Imp, []error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(storedRequestTimeoutMillis)*time.Millisecond)
	defer cancel()

	impr := openrtb.Imp{}
	_, imp, err := deps.storedReqFetcher.FetchRequests(ctx, []string{}, []string{storedImpId})
	if err != nil {
		return impr, err
	}

	if err := json.Unmarshal(imp[storedImpId], &impr); err != nil {
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

func buildVideoResponse(bidresponse *openrtb.BidResponse, podErrors []PodError) (*openrtb_ext.BidResponseVideo, error) {

	adPods := make([]*openrtb_ext.AdPod, 0)
	anyBidsReturned := false
	for _, seatBid := range bidresponse.SeatBid {
		for _, bid := range seatBid.Bid {
			anyBidsReturned = true

			var tempRespBidExt openrtb_ext.ExtBid
			if err := json.Unmarshal(bid.Ext, &tempRespBidExt); err != nil {
				return nil, err
			}
			if tempRespBidExt.Prebid.Targeting[string(openrtb_ext.HbVastCacheKey)] == "" {
				continue
			}

			impId := bid.ImpID
			podNum := strings.Split(impId, "_")[0]
			podId, _ := strconv.ParseInt(podNum, 0, 64)

			videoTargeting := openrtb_ext.VideoTargeting{
				HbPb:       tempRespBidExt.Prebid.Targeting[string(openrtb_ext.HbpbConstantKey)],
				HbPbCatDur: tempRespBidExt.Prebid.Targeting[string(openrtb_ext.HbCategoryDurationKey)],
				HbCacheID:  tempRespBidExt.Prebid.Targeting[string(openrtb_ext.HbVastCacheKey)],
			}

			adPod := findAdPod(podId, adPods)
			if adPod == nil {
				adPod = &openrtb_ext.AdPod{
					PodId:     podId,
					Targeting: make([]openrtb_ext.VideoTargeting, 0, 0),
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

func mergeData(videoRequest *openrtb_ext.BidRequestVideo, bidRequest *openrtb.BidRequest) error {

	if videoRequest.Site != nil {
		bidRequest.Site = videoRequest.Site
		if &videoRequest.Content != nil {
			bidRequest.Site.Content = &videoRequest.Content
		}
	}

	if videoRequest.App != nil {
		bidRequest.App = videoRequest.App
		if &videoRequest.Content != nil {
			bidRequest.App.Content = &videoRequest.Content
		}
	}

	if &videoRequest.Device != nil {
		bidRequest.Device = &videoRequest.Device
	}

	if &videoRequest.User != nil {
		bidRequest.User = &openrtb.User{
			BuyerUID: videoRequest.User.Buyeruids["appnexus"], //TODO: map to string merging
			Yob:      videoRequest.User.Yob,
			Gender:   videoRequest.User.Gender,
			Keywords: videoRequest.User.Keywords,
		}
	}

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

	return nil
}

func createBidExtension(videoRequest *openrtb_ext.BidRequestVideo) ([]byte, error) {

	var inclBrandCat *openrtb_ext.ExtIncludeBrandCategory
	if videoRequest.IncludeBrandCategory != nil {
		inclBrandCat = &openrtb_ext.ExtIncludeBrandCategory{
			PrimaryAdServer: videoRequest.IncludeBrandCategory.PrimaryAdserver,
			Publisher:       videoRequest.IncludeBrandCategory.Publisher,
			WithCategory:    true,
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

	priceGranularity := openrtb_ext.PriceGranularityFromString("med")
	if videoRequest.PriceGranularity.Precision != 0 {
		priceGranularity = videoRequest.PriceGranularity
	}

	targeting := openrtb_ext.ExtRequestTargeting{
		PriceGranularity:     priceGranularity,
		IncludeWinners:       true,
		IncludeBrandCategory: inclBrandCat,
		DurationRangeSec:     durationRangeSec,
	}

	vastXml := openrtb_ext.ExtRequestPrebidCacheVAST{}
	cache := openrtb_ext.ExtRequestPrebidCache{
		VastXML: &vastXml,
	}

	prebid := openrtb_ext.ExtRequestPrebid{
		Cache:     &cache,
		Targeting: &targeting,
	}
	extReq := openrtb_ext.ExtRequest{Prebid: prebid}

	reqJSON, err := json.Marshal(extReq)
	if err != nil {
		return nil, err
	}
	return reqJSON, nil
}

func (deps *endpointDeps) parseVideoRequest(request []byte) (req *openrtb_ext.BidRequestVideo, errs []error, podErrors []PodError) {
	req = &openrtb_ext.BidRequestVideo{}

	if err := json.Unmarshal(request, &req); err != nil {
		errs = []error{err}
		return
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
	podErrors := make([]PodError, 0, 0)
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
			if _, found := deps.cfg.BlacklistedAppMap[req.App.ID]; found {
				err := &errortypes.BlacklistedApp{Message: fmt.Sprintf("Prebid-server does not process requests from App ID: %s", req.App.ID)}
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

	if len(req.Video.Mimes) == 0 {
		err := errors.New("request missing required field: Video.Mimes")
		errL = append(errL, err)
	} else {
		mimes := make([]string, 0, 0)
		for _, mime := range req.Video.Mimes {
			if mime != "" {
				mimes = append(mimes, mime)
			}
		}
		if len(mimes) == 0 {
			err := errors.New("request missing required field: Video.Mimes, mime types contains empty strings only")
			errL = append(errL, err)
		}
		if len(mimes) > 0 {
			req.Video.Mimes = mimes
		}
	}

	if len(req.Video.Protocols) == 0 {
		err := errors.New("request missing required field: Video.Protocols")
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
