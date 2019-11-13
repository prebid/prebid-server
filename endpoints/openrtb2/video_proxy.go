package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests"
	"net/http"
	"time"
)

func NewVideoProxyEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsById stored_requests.Fetcher, videoFetcher stored_requests.Fetcher, categories stored_requests.CategoryFetcher, cfg *config.Configuration, met pbsmetrics.MetricsEngine, pbsAnalytics analytics.PBSAnalyticsModule, disabledBidders map[string]string, defReqJSON []byte, bidderMap map[string]openrtb_ext.BidderName) (httprouter.Handle, error) {

	if ex == nil || validator == nil || requestsById == nil || cfg == nil || met == nil {
		return nil, errors.New("NewVideoEndpoint requires non-nil arguments.")
	}
	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	return httprouter.Handle((&endpointDeps{ex, validator, requestsById, videoFetcher, categories, cfg, met, pbsAnalytics, disabledBidders, defRequest, defReqJSON, bidderMap}).VideoProxyEndpoint), nil
}

/**

Video proxy endpoint is SSAI server integration proxy intended to execute video endpoint logic,
append key value pairs in SSAI server format and redirect to SSAI server. By having this SSAI servers don't need to
integrate with Prebid Server.

1. Parse URL params and extract stored video request config id and SSAI server id
2. Load video request
3. Load SSAI server metadata
4. Execute video logic
5. Extract key-values
6. Append/Replace key-value pairs and build SSAI server URL and return redirect

URL params:
url - SSAI server url with all initial parameters
configid - stored video request config id

--------------To remove-----------------
localhost:8090/videoproxy?configid=81262407-734f-4ce0-9d39-94e4f7b0dbed&url=

to encode URL: encodeURIComponent("http://google.com/test?q=query&n=10")
will produce http%3A%2F%2Fgoogle.com%2Ftest%3Fq%3Dquery%26n%3D10
----------------------------------------

**/
func (deps *endpointDeps) VideoProxyEndpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

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

	configIds, ok := r.URL.Query()["configid"]

	if !ok || len(configIds) == 0 {
		fmt.Println("Video Proxy, confid is missing")
	}
	urlParam, ok := r.URL.Query()["url"]

	if !ok || len(urlParam) == 0 {
		fmt.Println("Video Proxy, url id is missing")
	}

	confId := string(configIds[0])
	url := string(urlParam[0])

	fmt.Println("Video Proxy, confid=", confId, " url=", url)

	storedRequest, errs := deps.loadStoredVideoRequest(context.Background(), confId)
	if len(errs) > 0 {
		handleError(labels, w, errs, ao)
		return
	}

	//load ssai serv er metadata
	//validate metadata

	storedVideoRequest := &openrtb_ext.BidRequestVideo{}

	if err := json.Unmarshal(storedRequest, &storedVideoRequest); err != nil {
		errs = []error{err}
		return
	}

	videoResp, _ := deps.executeVideoRequest(storedVideoRequest, make([]PodError, 0), labels, w, r, ao, start)

	if len(videoResp.AdPods) > 0 {
		hbCatDur := make([]string, 0)
		bhCahce := videoResp.AdPods[0].Targeting[0].HbCacheID
		for _, pod := range videoResp.AdPods {
			for _, ad := range pod.Targeting {
				hbCatDur = append(hbCatDur, ad.HbPbCatDur)
			}
		}
		fmt.Println("hbCache " + string(bhCahce))
		fmt.Println("hbCat ", hbCatDur)
	} else {
		err := errors.New("empty response")
		errs = []error{err}
		return
	}

	//r.Method = "POST"
	//http.Redirect(w, r, "http://www.google.com", 302)

}
