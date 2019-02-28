package openrtb2

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests"
	"io"
	"io/ioutil"
	"net/http"
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
6. Call setFieldsImplicitly from auction.go to get basic data from the HTTP request into an OpenRTB bid request to start building the OpenRTB bid request.
7. Add fields from merged JSON data that correspond to an OpenRTB request into the OpenRTB bid request we are building.
	a. Unmarshal certain OpenRTB defined structs directly into the OpenRTB bid request.
	b. In cases where customized logic is needed just copy/fill the fields in directly.
8. Loop through ad pods to build array of Imps into OpenRTB request, for each pod:
	a. Load the stored impression to use as the basis for impressions generated for this pod from the configid field.
	b. NumImps = adpoddurationsec / MIN_VALUE(allowedDurations)
	c. Build impression array for this pod:
		i. Create array of NumImps entries initialized to the base impression loaded from the configid.
			1. If requireexactdurations = true, iterate over allowdDurations and for (NumImps / len(allowedDurations)) number of Imps set minduration = maxduration = allowedDurations[i]
			2. If requireexactdurations = false, set maxduration = MAX_VALUE(allowedDurations)
		ii. Set Imp.id field to "podX_Y" where X is the pod index and Y is the impression index within this pod.
	d. Append impressions for this pod to the overall list of impressions in the OpenRTB bid request.
9. Call validateRequest() function from auction.go to validate the generated request.
10. Build response in proper format
*/
func (deps *endpointDeps) VideoAuctionEndpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	lr := &io.LimitedReader{
		R: r.Body,
		N: deps.cfg.MaxRequestSize,
	}
	requestJson, err := ioutil.ReadAll(lr)
	if err != nil {
		return
	}

	//load additional data - stored simplified req
	storedRequestId, err := getVideoStoredRequestId(requestJson)
	if err != nil {
		return
	}
	storedRequest, err := loadStoredVideoRequest(storedRequestId)
	if err != nil {
		return
	}

	//merge incoming req with stored video req
	resolvedRequest, err := jsonpatch.MergePatch(requestJson, storedRequest)

	//unmarshal and validate combined result
	videoBidReq, errl := deps.parseVideoRequest(resolvedRequest)
	if len(errl) > 0 {
		return
	}

	var bidReq = openrtb.BidRequest{}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(r, &bidReq)

	//create full open rtb req from full video request
	mergeData(&videoBidReq, &bidReq)

	//create impressions array
	imps, errl := createImpressions(videoBidReq)
	bidReq.Imp = imps

	errL := deps.validateRequest(&bidReq)
	if len(errL) > 0 {
		//handle errors
		return
	}

	labels := pbsmetrics.Labels{}
	//execute auction logic
	response, err := deps.ex.HoldAuction(nil, &bidReq, nil, labels, &deps.categories)
	if err != nil {
		//handle error
	}

	//build simplified response
	var bidResp = openrtb_ext.BidResponseVideo{}
	buildVideoResponse(response, &bidResp)

}

func createImpressions(videoReq openrtb_ext.BidRequestVideo) (imps []openrtb.Imp, errs []error) {
	videoDur := videoReq.PodConfig.DurationRangeSec
	minDuration, maxDuration := minMax(videoDur)
	reqExactDur := videoReq.PodConfig.RequireExactDuration
	videoData := videoReq.Video

	finalImpsArray := make([]openrtb.Imp, 0)
	for podIndex, pod := range videoReq.PodConfig.Pods {

		//load stored impression
		storedImpressionId := string(pod.ConfigId)
		storedImp := loadStoredImp(storedImpressionId)

		impTemplate := createImpressionTemplate(storedImp, videoData)
		numImps := pod.AdPodDurationSec / minDuration

		impDivNumber := numImps / len(videoDur)

		impsArray := make([]openrtb.Imp, numImps)
		for impInd := range impsArray {
			impsArray[impInd] = impTemplate
			if reqExactDur {
				//floor := int(math.Floor(ind/impDivNumber))
				floor := impInd / impDivNumber
				impsArray[impInd].Video.MaxDuration = int64(videoDur[floor])
				impsArray[impInd].Video.MinDuration = int64(videoDur[floor])
				fmt.Println(podIndex, "  ", impInd, "duration ", videoDur[floor])
			} else {
				impsArray[impInd].Video.MaxDuration = int64(maxDuration)
			}

			impsArray[impInd].ID = fmt.Sprintf("pod%d_%d", podIndex, impInd)
		}
		finalImpsArray = append(finalImpsArray, impsArray...)

	}
	return finalImpsArray, nil
}

func createImpressionTemplate(imp openrtb.Imp, video openrtb_ext.SimplifiedVideo) openrtb.Imp {
	imp.Video = &openrtb.Video{
		nil,
		0,
		0,
		nil,
		0,
		0,
		0,
		nil,
		0,
		0,
		nil,
		0,
		0,
		0,
		nil,
		0,
		0,
		0,
		0,
		nil,
		0,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	}
	imp.Video.W = video.W
	imp.Video.H = video.H
	imp.Video.Protocols = video.Protocols
	imp.Video.MIMEs = video.Mime
	return imp
}

func loadStoredImp(storedImpId string) openrtb.Imp {
	return openrtb.Imp{
		ID: "stored_imp_id",
		Ext: []byte(`{  
            "appnexus":{  
               "placementId":12971250
            }
         }`)}

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

func buildVideoResponse(bidresponse *openrtb.BidResponse, videoResponse *openrtb_ext.BidResponseVideo) { //should be video response
	return
}

func loadStoredVideoRequest(storedRequestId string) ([]byte, error) {
	jsonString := []byte(`{"accountid": "11223344", "app": {"domain": "mygame.foo.com"}}`)
	return jsonString, nil
}

func getVideoStoredRequestId(request []byte) (string, error) {
	req := openrtb_ext.StoredRequestId{}

	if err := json.Unmarshal(request, &req); err != nil {
		return "", err
	}
	return req.StoredRequestId, nil

}

func mergeData(videoRequest *openrtb_ext.BidRequestVideo, bidRequest *openrtb.BidRequest) {

}

func (deps *endpointDeps) parseVideoRequest(request []byte) (req openrtb_ext.BidRequestVideo, errs []error) {
	req = openrtb_ext.BidRequestVideo{}

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

func (deps *endpointDeps) validateVideoRequest(req openrtb_ext.BidRequestVideo) []error {
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
	if req.App.Domain == "" || req.Site.Page == "" {
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
