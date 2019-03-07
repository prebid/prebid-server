package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/evanphx/json-patch"
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
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewVideoEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsById stored_requests.Fetcher, cfg *config.Configuration, met pbsmetrics.MetricsEngine, pbsAnalytics analytics.PBSAnalyticsModule, disabledBidders map[string]string, defReqJSON []byte, bidderMap map[string]openrtb_ext.BidderName, categories stored_requests.CategoryFetcher) (httprouter.Handle, error) {

	if ex == nil || validator == nil || requestsById == nil || cfg == nil || met == nil {
		return nil, errors.New("NewSimplifiedEndpoint requires non-nil arguments.")
	}
	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	return httprouter.Handle((&endpointDeps{ex, validator, requestsById, cfg, met, pbsAnalytics, disabledBidders, defRequest, defReqJSON, bidderMap, categories}).VideoAuctionEndpoint), nil
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
		RType:         pbsmetrics.ReqTypeORTB2Web,
		PubID:         "",
		Browser:       pbsmetrics.BrowserOther,
		CookieFlag:    pbsmetrics.CookieFlagUnknown,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}
	numImps := 0
	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordImps(labels, numImps)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogAuctionObject(&ao)
	}()

	isSafari := checkSafari(r)
	if isSafari {
		labels.Browser = pbsmetrics.BrowserSafari
	}

	lr := &io.LimitedReader{
		R: r.Body,
		N: deps.cfg.MaxRequestSize,
	}
	requestJson, err := ioutil.ReadAll(lr)
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return
	}

	//load additional data - stored simplified req
	storedRequestId, err := getVideoStoredRequestId(requestJson)
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return
	}
	storedRequest, err := loadStoredVideoRequest(storedRequestId)
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return
	}

	//merge incoming req with stored video req
	resolvedRequest, err := jsonpatch.MergePatch(storedRequest, requestJson)

	//unmarshal and validate combined result
	videoBidReq, errL := deps.parseVideoRequest(resolvedRequest)
	if len(errL) > 0 {
		handleError(labels, w, errL, ao)
		return
	}

	var bidReq = &openrtb.BidRequest{}

	//create full open rtb req from full video request
	mergeData(videoBidReq, bidReq)

	//create impressions array
	imps, err := createImpressions(videoBidReq)
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return
	}
	bidReq.Imp = imps
	bidReq.ID = "bid_id" //TODO: look at auction

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(r, bidReq) // move after merge

	errL = deps.validateRequest(bidReq)
	if len(errL) > 0 {
		handleError(labels, w, errL, ao)
		return
	}

	ctx := context.Background()
	cancel := func() {}
	timeout := deps.cfg.AuctionTimeouts.LimitAuctionTimeout(time.Duration(bidReq.TMax) * time.Millisecond)
	if timeout > 0 {
		ctx, cancel = context.WithDeadline(ctx, start.Add(timeout))
	}
	defer cancel()

	usersyncs := usersync.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie))
	if bidReq.App != nil {
		labels.Source = pbsmetrics.DemandApp
	} else {
		labels.Source = pbsmetrics.DemandWeb
		if usersyncs.LiveSyncCount() == 0 {
			labels.CookieFlag = pbsmetrics.CookieFlagNo
		} else {
			labels.CookieFlag = pbsmetrics.CookieFlagYes
		}
	}

	numImps = len(bidReq.Imp)

	//execute auction logic
	response, err := deps.ex.HoldAuction(ctx, bidReq, usersyncs, labels, &deps.categories)
	ao.Request = bidReq
	ao.Response = response
	if err != nil {
		errL := []error{err}
		handleError(labels, w, errL, ao)
		return
	}

	//build simplified response
	bidResp, err := buildVideoResponse(response)
	if err != nil {
		errL := []error{err}
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

func handleError(labels pbsmetrics.Labels, w http.ResponseWriter, errL []error, ao analytics.AuctionObject) {
	labels.RequestStatus = pbsmetrics.RequestStatusErr
	w.WriteHeader(http.StatusInternalServerError)
	var errors string
	for _, er := range errL {
		errors = fmt.Sprintf("%s %s", errors, er.Error())
	}
	fmt.Fprintf(w, "Critical error while running the video endpoint: %v", errors)
	glog.Errorf("/openrtb2/video Critical error: %v", errors)
	ao.Status = http.StatusInternalServerError
	ao.Errors = append(ao.Errors, errL...)
}

func createImpressions(videoReq *openrtb_ext.BidRequestVideo) (imps []openrtb.Imp, err error) {
	videoDur := videoReq.PodConfig.DurationRangeSec
	minDuration, maxDuration := minMax(videoDur)
	reqExactDur := videoReq.PodConfig.RequireExactDuration
	videoData := videoReq.Video

	finalImpsArray := make([]openrtb.Imp, 0)
	for _, pod := range videoReq.PodConfig.Pods {

		//load stored impression
		storedImpressionId := string(pod.ConfigId)
		storedImp := loadStoredImp(storedImpressionId)

		numImps := pod.AdPodDurationSec / minDuration

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
				//fmt.Println(podIndex, "  ", impInd, "duration ", videoDur[durationIndex])
			} else {
				impsArray[impInd].Video.MaxDuration = int64(maxDuration)
			}

			impsArray[impInd].ID = fmt.Sprintf("%d_%d", pod.PodId, impInd)
		}
		finalImpsArray = append(finalImpsArray, impsArray...)

	}
	return finalImpsArray, nil
}

func createImpressionTemplate(imp openrtb.Imp, video openrtb_ext.SimplifiedVideo) openrtb.Imp {
	imp.Video = &openrtb.Video{}
	imp.Video.W = video.W
	imp.Video.H = video.H
	imp.Video.Protocols = video.Protocols
	imp.Video.MIMEs = video.Mime
	return imp
}

func loadStoredImp(storedImpId string) openrtb.Imp {
	//temporary stub to test endpoint: placement id in place of pod config id
	ext := fmt.Sprintf(`{"appnexus":{"placementId": %s}}`, storedImpId) //12971250

	return openrtb.Imp{
		ID:  "stored_imp_id",
		Ext: []byte(ext)}

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

func buildVideoResponse(bidresponse *openrtb.BidResponse) (*openrtb_ext.BidResponseVideo, error) { //should be video response

	adPods := make([]*openrtb_ext.AdPod, 0)
	for _, seatBid := range bidresponse.SeatBid {
		for _, bid := range seatBid.Bid {

			var tempRespBidExt openrtb_ext.ExtBid
			if err := json.Unmarshal(bid.Ext, &tempRespBidExt); err != nil {
				return nil, err
			}

			impId := bid.ImpID
			podNum := strings.Split(impId, "_")[0]
			podInd, _ := strconv.ParseInt(podNum, 0, 64)

			videoTargeting := openrtb_ext.VideoTargeting{
				tempRespBidExt.Prebid.Targeting["hb_pb"],
				tempRespBidExt.Prebid.Targeting["hb_pb_cat_dur"],
				"",
			}

			adPod := findAdPod(podInd, adPods)
			if adPod == nil {
				adPod = &openrtb_ext.AdPod{
					podInd,
					make([]openrtb_ext.VideoTargeting, 0, 0),
					openrtb_ext.VideoErrors{},
				}
				adPods = append(adPods, adPod)
			}
			adPod.Targeting = append(adPod.Targeting, videoTargeting)

		}
	}
	videoResponse := openrtb_ext.BidResponseVideo{}
	videoResponse.AdPods = adPods

	return &videoResponse, nil
}

func findAdPod(podInd int64, pods []*openrtb_ext.AdPod) *openrtb_ext.AdPod {
	for _, pod := range pods {
		if pod.PodId == podInd {
			return pod
		}
	}
	return nil
}

func loadStoredVideoRequest(storedRequestId string) ([]byte, error) {
	jsonString := []byte(`{"accountid": "11223344", "site": {"page": "mygame.foo.com"}}`)
	return jsonString, nil
}

func getVideoStoredRequestId(request []byte) (string, error) {
	req := openrtb_ext.StoredRequestId{}

	if err := json.Unmarshal(request, &req); err != nil {
		return "", err
	}
	return req.StoredRequestId, nil

}

func mergeData(videoRequest *openrtb_ext.BidRequestVideo, bidRequest *openrtb.BidRequest) error {

	if videoRequest.Site.Page != "" {
		bidRequest.Site = &videoRequest.Site
		if &videoRequest.Content != nil {
			bidRequest.Site.Content = &videoRequest.Content
		}
	}

	if videoRequest.App.Domain != "" {
		bidRequest.App = &videoRequest.App
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

	bidExt, err := createBidExtension(videoRequest)
	if err != nil {
		return err
	}
	if len(bidExt) > 0 {
		bidRequest.Ext = bidExt
	}

	bidRequest.TMax = 5000
	return nil
}

func createBidExtension(videoRequest *openrtb_ext.BidRequestVideo) ([]byte, error) {

	inclBrandCat := openrtb_ext.ExtIncludeBrandCategory{
		videoRequest.IncludeBrandCategory.PrimaryAdserver,
		videoRequest.IncludeBrandCategory.Publisher,
	}
	targeting := openrtb_ext.ExtRequestTargeting{
		openrtb_ext.PriceGranularityFromString("med"),
		true,
		false,
		inclBrandCat}

	prebid := openrtb_ext.ExtRequestPrebid{
		Targeting: &targeting,
	}
	extReq := openrtb_ext.ExtRequest{prebid}

	reqJSON, err := json.Marshal(extReq)
	if err != nil {
		return nil, err
	}
	return reqJSON, nil
}

func (deps *endpointDeps) parseVideoRequest(request []byte) (req *openrtb_ext.BidRequestVideo, errs []error) {
	req = &openrtb_ext.BidRequestVideo{}

	if err := json.Unmarshal(request, &req); err != nil {
		errs = []error{err}
		return
	}

	errL := deps.validateVideoRequest(req)
	if len(errL) > 0 {
		errs = append(errs, errL...)
	}
	return
}

func (deps *endpointDeps) validateVideoRequest(req *openrtb_ext.BidRequestVideo) []error {
	errL := []error{}
	if req.AccountId == "" {
		err := errors.New("request missing required field: accountid")
		errL = append(errL, err)
	}
	if req.StoredRequestId == "" {
		err := errors.New("request missing required field: storedrequestid")
		errL = append(errL, err)
	}
	if len(req.PodConfig.DurationRangeSec) == 0 {
		err := errors.New("request missing required field: PodConfig.DurationRangeSec")
		errL = append(errL, err)
	}
	if len(req.PodConfig.Pods) == 0 {
		err := errors.New("request missing required field: PodConfig.Pods")
		errL = append(errL, err)
	}
	for ind, pod := range req.PodConfig.Pods {
		if pod.PodId <= 0 {
			err := fmt.Errorf("request missing required field: PodConfig.Pods.PodId, Pod index: %d", ind)
			errL = append(errL, err)
		}
		if pod.AdPodDurationSec == 0 {
			err := fmt.Errorf("request missing required field: PodConfig.Pods.AdPodDurationSec, Pod index: %d", ind)
			errL = append(errL, err)
		}
		if pod.ConfigId == "" {
			err := fmt.Errorf("request missing required field: PodConfig.Pods.ConfigId, Pod index: %d", ind)
			errL = append(errL, err)
		}
	}
	if req.App.Domain == "" && req.Site.Page == "" {
		err := errors.New("request missing required field: site or app")
		errL = append(errL, err)
	}
	if len(req.Video.Mime) == 0 {
		err := errors.New("request missing required field: Video.Mime")
		errL = append(errL, err)
	}
	if len(req.Video.Protocols) == 0 {
		err := errors.New("request missing required field: Video.Protocols")
		errL = append(errL, err)
	}

	return errL
}
