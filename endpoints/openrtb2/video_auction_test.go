package openrtb2

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestVideoEndpointImpressionsNumber(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqData, err := ioutil.ReadFile("sample-requests/video/video_valid_sample.json")
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	reqBody := string(getRequestPayload(t, reqData))
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	respBytes := recorder.Body.Bytes()
	resp := &openrtb_ext.BidResponseVideo{}
	if err := json.Unmarshal(respBytes, resp); err != nil {
		t.Fatalf("Unable to umarshal response.")
	}

	assert.Equal(t, len(ex.lastRequest.Imp), 11, "Incorrect number of impressions in request")
	assert.Equal(t, string(ex.lastRequest.Site.Page), "prebid.com", "Incorrect site page in request")
	assert.Equal(t, ex.lastRequest.Site.Content.Series, "TvName", "Incorrect site content series in request")

	assert.Equal(t, len(resp.AdPods), 5, "Incorrect number of Ad Pods in response")
	assert.Equal(t, len(resp.AdPods[0].Targeting), 4, "Incorrect Targeting data in response")
	assert.Equal(t, len(resp.AdPods[1].Targeting), 3, "Incorrect Targeting data in response")
	assert.Equal(t, len(resp.AdPods[2].Targeting), 5, "Incorrect Targeting data in response")
	assert.Equal(t, len(resp.AdPods[3].Targeting), 1, "Incorrect Targeting data in response")
	assert.Equal(t, len(resp.AdPods[4].Targeting), 3, "Incorrect Targeting data in response")

	assert.Equal(t, resp.AdPods[4].Targeting[0].Hb_pb_cat_dur, "20.00_395_30s", "Incorrect number of Ad Pods in response")

}

func TestVideoEndpointImpressionsDuration(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqData, err := ioutil.ReadFile("sample-requests/video/video_valid_sample_different_durations.json")
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	reqBody := string(getRequestPayload(t, reqData))
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	assert.Equal(t, len(ex.lastRequest.Imp), 22, "Incorrect number of impressions in request")
	assert.Equal(t, ex.lastRequest.Imp[0].ID, "1_0", "Incorrect impression id in request")
	assert.Equal(t, ex.lastRequest.Imp[0].Video.MaxDuration, int64(15), "Incorrect impression max duration in request")
	assert.Equal(t, ex.lastRequest.Imp[0].Video.MinDuration, int64(15), "Incorrect impression min duration in request")

	assert.Equal(t, ex.lastRequest.Imp[6].ID, "1_6", "Incorrect impression id in request")
	assert.Equal(t, ex.lastRequest.Imp[6].Video.MaxDuration, int64(30), "Incorrect impression max duration in request")
	assert.Equal(t, ex.lastRequest.Imp[6].Video.MinDuration, int64(30), "Incorrect impression min duration in request")

	assert.Equal(t, ex.lastRequest.Imp[12].ID, "2_0", "Incorrect impression id in request")
	assert.Equal(t, ex.lastRequest.Imp[12].Video.MaxDuration, int64(15), "Incorrect impression max duration in request")
	assert.Equal(t, ex.lastRequest.Imp[12].Video.MinDuration, int64(15), "Incorrect impression min duration in request")

	assert.Equal(t, ex.lastRequest.Imp[17].ID, "2_5", "Incorrect impression id in request")
	assert.Equal(t, ex.lastRequest.Imp[17].Video.MaxDuration, int64(30), "Incorrect impression max duration in request")
	assert.Equal(t, ex.lastRequest.Imp[17].Video.MinDuration, int64(30), "Incorrect impression min duration in request")

}

func TestCreateBidExtension(t *testing.T) {
	durationRange := make([]int, 0)
	durationRange = append(durationRange, 15)
	durationRange = append(durationRange, 30)

	priceGranRanges := make([]openrtb_ext.GranularityRange, 0)
	priceGranRanges = append(priceGranRanges, openrtb_ext.GranularityRange{
		Max:       30,
		Min:       0,
		Increment: 0.1,
	})

	videoRequest := openrtb_ext.BidRequestVideo{
		IncludeBrandCategory: openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
			Publisher:       "",
		},
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: false,
		},
		PriceGranularity: openrtb_ext.PriceGranularity{
			Precision: 2,
			Ranges:    priceGranRanges,
		},
	}
	res, err := createBidExtension(&videoRequest)
	assert.Equal(t, err, nil, "Error should be nil")

	resExt := &openrtb_ext.ExtRequest{}

	if err := json.Unmarshal(res, &resExt); err != nil {
		assert.Fail(t, "Unable to unmarshal bid extension")
	}
	assert.Equal(t, resExt.Prebid.Targeting.DurationRangeSec, durationRange, "Duration range seconds is incorrect")
	assert.Equal(t, resExt.Prebid.Targeting.PriceGranularity.Ranges, priceGranRanges, "Price granularity is incorrect")

}

func TestCreateBidExtensionExactDurTrueNoPriceRange(t *testing.T) {
	durationRange := make([]int, 0)
	durationRange = append(durationRange, 15)
	durationRange = append(durationRange, 30)

	videoRequest := openrtb_ext.BidRequestVideo{
		IncludeBrandCategory: openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
			Publisher:       "",
		},
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
		},
		PriceGranularity: openrtb_ext.PriceGranularity{
			Precision: 0,
			Ranges:    nil,
		},
	}
	res, err := createBidExtension(&videoRequest)
	assert.Equal(t, err, nil, "Error should be nil")

	resExt := &openrtb_ext.ExtRequest{}

	if err := json.Unmarshal(res, &resExt); err != nil {
		assert.Fail(t, "Unable to unmarshal bid extension")
	}
	assert.Equal(t, resExt.Prebid.Targeting.DurationRangeSec, []int(nil), "Duration range seconds is incorrect")
	assert.Equal(t, resExt.Prebid.Targeting.PriceGranularity, openrtb_ext.PriceGranularityFromString("med"), "Price granularity is incorrect")
}

func TestVideoEndpointNoPods(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqData, err := ioutil.ReadFile("sample-requests/video/video_invalid_sample.json")
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	reqBody := string(getRequestPayload(t, reqData))
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	errorMessage := string(recorder.Body.Bytes())

	assert.Equal(t, recorder.Code, 500, "Should catch error in request")
	assert.Equal(t, "Critical error while running the video endpoint:  request missing required field: PodConfig.DurationRangeSec request missing required field: PodConfig.Pods", errorMessage, "Incorrect request validation message")
}

func TestVideoEndpointValidationsPositive(t *testing.T) {
	ex := &mockExchangeVideo{}
	deps := mockDeps(t, ex)
	deps.cfg.VideoStoredRequestRequired = true

	durationRange := make([]int, 0)
	durationRange = append(durationRange, 15)
	durationRange = append(durationRange, 30)

	pods := make([]openrtb_ext.Pod, 0)
	pod1 := openrtb_ext.Pod{
		PodId:            1,
		AdPodDurationSec: 30,
		ConfigId:         "qwerty",
	}
	pod2 := openrtb_ext.Pod{
		PodId:            2,
		AdPodDurationSec: 30,
		ConfigId:         "qwerty",
	}
	pods = append(pods, pod1)
	pods = append(pods, pod2)

	mimes := make([]string, 0)
	mimes = append(mimes, "mp4")
	mimes = append(mimes, "")

	videoProtocols := make([]openrtb.Protocol, 0)
	videoProtocols = append(videoProtocols, 15)
	videoProtocols = append(videoProtocols, 30)

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		App: openrtb.App{
			Domain: "pbs.com",
		},
		IncludeBrandCategory: openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
		Video: openrtb_ext.SimplifiedVideo{
			Mimes:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Equal(t, 0, len(errors), "Errors should be empty")
	assert.Equal(t, 0, len(podErrors), "Pod errors should be empty")
}

func TestVideoEndpointValidationsCritical(t *testing.T) {
	ex := &mockExchangeVideo{}
	deps := mockDeps(t, ex)
	deps.cfg.VideoStoredRequestRequired = true

	durationRange := make([]int, 0)
	durationRange = append(durationRange, 0)
	durationRange = append(durationRange, -30)

	pods := make([]openrtb_ext.Pod, 0)

	mimes := make([]string, 0)
	mimes = append(mimes, "")
	mimes = append(mimes, "")

	videoProtocols := make([]openrtb.Protocol, 0)

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		IncludeBrandCategory: openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 0,
		},
		Video: openrtb_ext.SimplifiedVideo{
			Mimes:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Equal(t, 0, len(podErrors), "Pod errors should be empty")
	assert.Equal(t, 6, len(errors), "Errors array should contain 6 error messages")

	assert.Equal(t, "request missing required field: storedrequestid", errors[0].Error(), "Errors array should contain 6 error messages")
	assert.Equal(t, "duration array cannot contain negative or zero values", errors[1].Error(), "Errors array should contain 6 error messages")
	assert.Equal(t, "request missing required field: PodConfig.Pods", errors[2].Error(), "Errors array should contain 6 error messages")
	assert.Equal(t, "request missing required field: site or app", errors[3].Error(), "Errors array should contain 6 error messages")
	assert.Equal(t, "request missing required field: Video.Mimes, mime types contains empty strings only", errors[4].Error(), "Errors array should contain 6 error messages")
	assert.Equal(t, "request missing required field: Video.Protocols", errors[5].Error(), "Errors array should contain 6 error messages")
}

func TestVideoEndpointValidationsPodErrors(t *testing.T) {
	ex := &mockExchangeVideo{}
	deps := mockDeps(t, ex)
	deps.cfg.VideoStoredRequestRequired = true

	durationRange := make([]int, 0)
	durationRange = append(durationRange, 15)
	durationRange = append(durationRange, 30)

	pods := make([]openrtb_ext.Pod, 0)
	pod1 := openrtb_ext.Pod{
		PodId:            1,
		AdPodDurationSec: 30,
		ConfigId:         "qwerty",
	}
	pod2 := openrtb_ext.Pod{
		PodId:            2,
		AdPodDurationSec: 30,
		ConfigId:         "qwerty",
	}
	pod3 := openrtb_ext.Pod{
		PodId:            2,
		AdPodDurationSec: 0,
		ConfigId:         "",
	}
	pod4 := openrtb_ext.Pod{
		PodId:            0,
		AdPodDurationSec: -30,
		ConfigId:         "",
	}
	pods = append(pods, pod1)
	pods = append(pods, pod2)
	pods = append(pods, pod3)
	pods = append(pods, pod4)

	mimes := make([]string, 0)
	mimes = append(mimes, "mp4")
	mimes = append(mimes, "")

	videoProtocols := make([]openrtb.Protocol, 0)
	videoProtocols = append(videoProtocols, 15)
	videoProtocols = append(videoProtocols, 30)

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		App: openrtb.App{
			Domain: "pbs.com",
		},
		IncludeBrandCategory: openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
		Video: openrtb_ext.SimplifiedVideo{
			Mimes:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Equal(t, 0, len(errors), "Errors should be empty")

	assert.Equal(t, 2, len(podErrors), "Pod errors should contain 2 elements")

	assert.Equal(t, 2, podErrors[0].PodId, "Pod error ind 0, incorrect id should be 2")
	assert.Equal(t, 2, podErrors[0].PodIndex, "Pod error ind 0, incorrect index should be 2")
	assert.Equal(t, 3, len(podErrors[0].ErrMsgs), "Pod error ind 0 should contain 3 errors")
	assert.Equal(t, "request duplicated required field: PodConfig.Pods.PodId, Pod id: 2", podErrors[0].ErrMsgs[0], "Pod error ind 0 should have duplicated pod id")
	assert.Equal(t, "request missing or incorrect required field: PodConfig.Pods.AdPodDurationSec, Pod index: 2", podErrors[0].ErrMsgs[1], "Pod error ind 0 should have missing AdPodDuration")
	assert.Equal(t, "request missing or incorrect required field: PodConfig.Pods.ConfigId, Pod index: 2", podErrors[0].ErrMsgs[2], "Pod error ind 0 should have missing config id")

	assert.Equal(t, 0, podErrors[1].PodId, "Pod error ind 1, incorrect id should be 0")
	assert.Equal(t, 3, podErrors[1].PodIndex, "Pod error ind 1, incorrect index should be 3")
	assert.Equal(t, 3, len(podErrors[1].ErrMsgs), "Pod error ind 1 should contain 3 errors")
	assert.Equal(t, "request missing required field: PodConfig.Pods.PodId, Pod index: 3", podErrors[1].ErrMsgs[0], "Pod error ind 1 should have missed pod id")
	assert.Equal(t, "request incorrect required field: PodConfig.Pods.AdPodDurationSec is negative, Pod index: 3", podErrors[1].ErrMsgs[1], "Pod error ind 1 should have negative AdPodDurationSec")
	assert.Equal(t, "request missing or incorrect required field: PodConfig.Pods.ConfigId, Pod index: 3", podErrors[1].ErrMsgs[2], "Pod error ind 1 should have missing config id")
}

func mockDeps(t *testing.T, ex *mockExchangeVideo) *endpointDeps {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	edep := &endpointDeps{
		ex,
		newParamsValidator(t),
		&mockVideoStoredReqFetcher{},
		&mockVideoStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
	}

	return edep
}

type mockVideoStoredReqFetcher struct {
}

func (cf mockVideoStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return testVideoStoredRequestData, testVideoStoredImpData, nil
}

type mockExchangeVideo struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockExchangeVideo) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest
	ext := []byte(`{"prebid":{"targeting":{"hb_bidder":"appnexus","hb_pb":"20.00","hb_pb_cat_dur":"20.00_395_30s","hb_size":"1x1"},"type":"video"},"bidder":{"appnexus":{"brand_id":1,"auction_id":7840037870526938650,"bidder_id":2,"bid_ad_type":1,"creative_info":{"video":{"duration":30,"mimes":["video\/mp4"]}}}}}`)
	return &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{
				{ID: "01", ImpID: "1_0", Ext: ext},
				{ID: "02", ImpID: "1_1", Ext: ext},
				{ID: "03", ImpID: "1_2", Ext: ext},
				{ID: "04", ImpID: "1_3", Ext: ext},
				{ID: "05", ImpID: "2_0", Ext: ext},
				{ID: "06", ImpID: "2_1", Ext: ext},
				{ID: "07", ImpID: "2_2", Ext: ext},
				{ID: "08", ImpID: "3_0", Ext: ext},
				{ID: "09", ImpID: "3_1", Ext: ext},
				{ID: "10", ImpID: "3_2", Ext: ext},
				{ID: "11", ImpID: "3_3", Ext: ext},
				{ID: "12", ImpID: "3_5", Ext: ext},
				{ID: "13", ImpID: "4_0", Ext: ext},
				{ID: "14", ImpID: "5_0", Ext: ext},
				{ID: "15", ImpID: "5_1", Ext: ext},
				{ID: "16", ImpID: "5_2", Ext: ext},
			},
		}},
	}, nil
}

var testVideoStoredImpData = map[string]json.RawMessage{
	"fba10607-0c12-43d1-ad07-b8a513bc75d6": json.RawMessage(`{"ext": {"appnexus": {"placementId": 14997137}}}`),
	"8b452b41-2681-4a20-9086-6f16ffad7773": json.RawMessage(`{"ext": {"appnexus": {"placementId": 15016213}}}`),
	"87d82a45-35c3-46cc-9315-2e3eeb91d0f2": json.RawMessage(`{"ext": {"appnexus": {"placementId": 15062775}}}`),
}

var testVideoStoredRequestData = map[string]json.RawMessage{
	"80ce30c53c16e6ede735f123ef6e32361bfc7b22": json.RawMessage(`{"accountid": "11223344", "site": {"page": "mygame.foo.com"}}`),
}
