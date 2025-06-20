package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v3/analytics"
	analyticsBuild "github.com/prebid/prebid-server/v3/analytics/build"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/metrics"
	metricsConfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoEndpointImpressionsNumber(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	respBytes := recorder.Body.Bytes()
	resp := &openrtb_ext.BidResponseVideo{}
	if err := jsonutil.UnmarshalValid(respBytes, resp); err != nil {
		t.Fatalf("Unable to unmarshal response.")
	}

	assert.Len(t, ex.lastRequest.Imp, 11, "Incorrect number of impressions in request")
	assert.Equal(t, "prebid.com", string(ex.lastRequest.Site.Page), "Incorrect site page in request")
	assert.Equal(t, "TvName", ex.lastRequest.Site.Content.Series, "Incorrect site content series in request")

	assert.Len(t, resp.AdPods, 5, "Incorrect number of Ad Pods in response")
	assert.Len(t, resp.AdPods[0].Targeting, 4, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[1].Targeting, 3, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[2].Targeting, 5, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[3].Targeting, 1, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[4].Targeting, 3, "Incorrect Targeting data in response")

	assert.Equal(t, "20.00_395_30s", resp.AdPods[4].Targeting[0].HbPbCatDur, "Incorrect number of Ad Pods in response")
	assert.Equal(t, "ABC_123", resp.AdPods[0].Targeting[0].HbDeal, "If DealID exists in bid response, hb_deal targeting needs to be added to resp")
}

func TestVideoEndpointImpressionsDuration(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_different_durations.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	var extData openrtb_ext.ExtRequest
	jsonutil.UnmarshalValid(ex.lastRequest.Ext, &extData)
	assert.NotNil(t, extData.Prebid.Targeting.IncludeBidderKeys, "Request ext incorrect: IncludeBidderKeys should be true ")
	assert.True(t, *extData.Prebid.Targeting.IncludeBidderKeys, "Request ext incorrect: IncludeBidderKeys should be true ")

	assert.Len(t, ex.lastRequest.Imp, 22, "Incorrect number of impressions in request")
	assert.Equal(t, "1_0", ex.lastRequest.Imp[0].ID, "Incorrect impression id in request")
	assert.Equal(t, int64(15), ex.lastRequest.Imp[0].Video.MaxDuration, "Incorrect impression max duration in request")
	assert.Equal(t, int64(15), ex.lastRequest.Imp[0].Video.MinDuration, "Incorrect impression min duration in request")

	assert.Equal(t, "1_6", ex.lastRequest.Imp[6].ID, "Incorrect impression id in request")
	assert.Equal(t, int64(30), ex.lastRequest.Imp[6].Video.MaxDuration, "Incorrect impression max duration in request")
	assert.Equal(t, int64(30), ex.lastRequest.Imp[6].Video.MinDuration, "Incorrect impression min duration in request")

	assert.Equal(t, "2_0", ex.lastRequest.Imp[12].ID, "Incorrect impression id in request")
	assert.Equal(t, int64(15), ex.lastRequest.Imp[12].Video.MaxDuration, "Incorrect impression max duration in request")
	assert.Equal(t, int64(15), ex.lastRequest.Imp[12].Video.MinDuration, "Incorrect impression min duration in request")

	assert.Equal(t, "2_5", ex.lastRequest.Imp[17].ID, "Incorrect impression id in request")
	assert.Equal(t, int64(30), ex.lastRequest.Imp[17].Video.MaxDuration, "Incorrect impression max duration in request")
	assert.Equal(t, int64(30), ex.lastRequest.Imp[17].Video.MinDuration, "Incorrect impression min duration in request")
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

	translateCategories := true
	videoRequest := openrtb_ext.BidRequestVideo{
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver:     1,
			Publisher:           "",
			TranslateCategories: &translateCategories,
		},
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: false,
		},
		PriceGranularity: &openrtb_ext.PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges:    priceGranRanges,
		},
	}
	res, err := createBidExtension(&videoRequest)
	assert.NoError(t, err, "Error should be nil")

	resExt := &openrtb_ext.ExtRequest{}

	if err := jsonutil.UnmarshalValid(res, &resExt); err != nil {
		assert.Fail(t, "Unable to unmarshal bid extension")
	}
	assert.Equal(t, durationRange, resExt.Prebid.Targeting.DurationRangeSec, "Duration range seconds is incorrect")
	assert.Equal(t, priceGranRanges, resExt.Prebid.Targeting.PriceGranularity.Ranges, "Price granularity is incorrect")
}

func TestCreateBidExtensionTargeting(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	require.NotNil(t, ex.lastRequest, "The request never made it into the Exchange.")

	// assert targeting set to default
	expectedRequestExt := `{"prebid":{"cache":{"vastxml":{}},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":20,"increment":0.1}]},"includebidderkeys":true,"includewinners":true,"includebrandcategory":{"primaryadserver":1,"withcategory":true}}}}`
	assert.JSONEq(t, expectedRequestExt, string(ex.lastRequest.Ext))
}

func TestVideoEndpointDebugQueryTrue(t *testing.T) {
	ex := &mockExchangeVideo{
		cache: &mockCacheClient{},
	}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video?debug=true", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}
	if !ex.cache.called {
		t.Fatalf("Cache was not called when it should have been")
	}

	respBytes := recorder.Body.Bytes()
	resp := &openrtb_ext.BidResponseVideo{}
	if err := jsonutil.UnmarshalValid(respBytes, resp); err != nil {
		t.Fatalf("Unable to unmarshal response.")
	}

	assert.Len(t, ex.lastRequest.Imp, 11, "Incorrect number of impressions in request")
	assert.Equal(t, "prebid.com", string(ex.lastRequest.Site.Page), "Incorrect site page in request")
	assert.Equal(t, "TvName", ex.lastRequest.Site.Content.Series, "Incorrect site content series in request")

	assert.Len(t, resp.AdPods, 5, "Incorrect number of Ad Pods in response")
	assert.Len(t, resp.AdPods[0].Targeting, 4, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[1].Targeting, 3, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[2].Targeting, 5, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[3].Targeting, 1, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[4].Targeting, 3, "Incorrect Targeting data in response")

	assert.Equal(t, "20.00_395_30s", resp.AdPods[4].Targeting[0].HbPbCatDur, "Incorrect number of Ad Pods in response")
}

func TestVideoEndpointDebugQueryFalse(t *testing.T) {
	ex := &mockExchangeVideo{
		cache: &mockCacheClient{},
	}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video?debug=123", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}
	if ex.cache.called {
		t.Fatalf("Cache was called when it shouldn't have been")
	}

	respBytes := recorder.Body.Bytes()
	resp := &openrtb_ext.BidResponseVideo{}
	if err := jsonutil.UnmarshalValid(respBytes, resp); err != nil {
		t.Fatalf("Unable to unmarshal response.")
	}

	assert.Len(t, ex.lastRequest.Imp, 11, "Incorrect number of impressions in request")
	assert.Equal(t, "prebid.com", string(ex.lastRequest.Site.Page), "Incorrect site page in request")
	assert.Equal(t, "TvName", ex.lastRequest.Site.Content.Series, "Incorrect site content series in request")

	assert.Len(t, resp.AdPods, 5, "Incorrect number of Ad Pods in response")
	assert.Len(t, resp.AdPods[0].Targeting, 4, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[1].Targeting, 3, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[2].Targeting, 5, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[3].Targeting, 1, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[4].Targeting, 3, "Incorrect Targeting data in response")

	assert.Equal(t, "20.00_395_30s", resp.AdPods[4].Targeting[0].HbPbCatDur, "Incorrect number of Ad Pods in response")
}

func TestVideoEndpointDebugError(t *testing.T) {
	ex := &mockExchangeVideo{
		cache: &mockCacheClient{},
	}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_invalid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video?debug=true", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if !ex.cache.called {
		t.Fatalf("Cache was not called when it should have been")
	}

	assert.Equal(t, 500, recorder.Code, "Should catch error in request")
}

func TestVideoEndpointDebugNoAdPods(t *testing.T) {
	ex := &mockExchangeVideoNoBids{
		cache: &mockCacheClient{},
	}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video?debug=true", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDepsNoBids(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}
	if !ex.cache.called {
		t.Fatalf("Cache was not called when it should have been")
	}

	respBytes := recorder.Body.Bytes()
	resp := &openrtb_ext.BidResponseVideo{}
	if err := jsonutil.UnmarshalValid(respBytes, resp); err != nil {
		t.Fatalf("Unable to unmarshal response.")
	}

	assert.Len(t, resp.AdPods, 1, "Debug AdPod should be added to response")
	assert.Empty(t, resp.AdPods[0].Errors, "AdPod Errors should be empty")
	assert.Empty(t, resp.AdPods[0].Targeting[0].HbPb, "Hb_pb should be empty")
	assert.Empty(t, resp.AdPods[0].Targeting[0].HbPbCatDur, "Hb_pb_cat_dur should be empty")
	assert.NotEmpty(t, resp.AdPods[0].Targeting[0].HbCacheID, "Hb_cache_id should not be empty")
	assert.Equal(t, int64(0), resp.AdPods[0].PodId, "Pod ID should be 0")
}

func TestVideoEndpointNoPods(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_invalid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	errorMessage := recorder.Body.String()

	assert.Equal(t, 500, recorder.Code, "Should catch error in request")
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

	videoProtocols := []adcom1.MediaCreativeSubtype{15, 30}

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		App: &openrtb2.App{
			Bundle: "pbs.com",
		},
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
		Video: &openrtb2.Video{
			MIMEs:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Len(t, errors, 0, "Errors should be empty")
	assert.Len(t, podErrors, 0, "Pod errors should be empty")
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

	videoProtocols := []adcom1.MediaCreativeSubtype{}

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 0,
		},
		Video: &openrtb2.Video{
			MIMEs:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Len(t, podErrors, 0, "Pod errors should be empty")
	assert.Len(t, errors, 6, "Errors array should contain 6 error messages")

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

	videoProtocols := []adcom1.MediaCreativeSubtype{15, 30}

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		App: &openrtb2.App{
			Bundle: "pbs.com",
		},
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
		Video: &openrtb2.Video{
			MIMEs:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Len(t, errors, 0, "Errors should be empty")

	assert.Len(t, podErrors, 2, "Pod errors should contain 2 elements")

	assert.Equal(t, 2, podErrors[0].PodId, "Pod error ind 0, incorrect id should be 2")
	assert.Equal(t, 2, podErrors[0].PodIndex, "Pod error ind 0, incorrect index should be 2")
	assert.Len(t, podErrors[0].ErrMsgs, 3, "Pod error ind 0 should contain 3 errors")
	assert.Equal(t, "request duplicated required field: PodConfig.Pods.PodId, Pod id: 2", podErrors[0].ErrMsgs[0], "Pod error ind 0 should have duplicated pod id")
	assert.Equal(t, "request missing or incorrect required field: PodConfig.Pods.AdPodDurationSec, Pod index: 2", podErrors[0].ErrMsgs[1], "Pod error ind 0 should have missing AdPodDuration")
	assert.Equal(t, "request missing or incorrect required field: PodConfig.Pods.ConfigId, Pod index: 2", podErrors[0].ErrMsgs[2], "Pod error ind 0 should have missing config id")

	assert.Equal(t, 0, podErrors[1].PodId, "Pod error ind 1, incorrect id should be 0")
	assert.Equal(t, 3, podErrors[1].PodIndex, "Pod error ind 1, incorrect index should be 3")
	assert.Len(t, podErrors[1].ErrMsgs, 3, "Pod error ind 1 should contain 3 errors")
	assert.Equal(t, "request missing required field: PodConfig.Pods.PodId, Pod index: 3", podErrors[1].ErrMsgs[0], "Pod error ind 1 should have missed pod id")
	assert.Equal(t, "request incorrect required field: PodConfig.Pods.AdPodDurationSec is negative, Pod index: 3", podErrors[1].ErrMsgs[1], "Pod error ind 1 should have negative AdPodDurationSec")
	assert.Equal(t, "request missing or incorrect required field: PodConfig.Pods.ConfigId, Pod index: 3", podErrors[1].ErrMsgs[2], "Pod error ind 1 should have missing config id")
}

func TestVideoEndpointValidationsSiteAndApp(t *testing.T) {
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

	videoProtocols := []adcom1.MediaCreativeSubtype{15, 30}

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		App: &openrtb2.App{
			Bundle: "pbs.com",
		},
		Site: &openrtb2.Site{
			ID: "pbs.com",
		},
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
		Video: &openrtb2.Video{
			MIMEs:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Equal(t, "request.site or request.app must be defined, but not both", errors[0].Error(), "Site and App error should be present")
	assert.Len(t, podErrors, 0, "Pod errors should be empty")
}

func TestVideoEndpointValidationsSiteMissingRequiredField(t *testing.T) {
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

	videoProtocols := []adcom1.MediaCreativeSubtype{15, 30}

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     durationRange,
			RequireExactDuration: true,
			Pods:                 pods,
		},
		Site: &openrtb2.Site{
			Domain: "pbs.com",
		},
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
		Video: &openrtb2.Video{
			MIMEs:     mimes,
			Protocols: videoProtocols,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Equal(t, "request.site missing required field: id or page", errors[0].Error(), "Site required fields error should be present")
	assert.Len(t, podErrors, 0, "Pod errors should be empty")
}

func TestVideoEndpointValidationsMissingVideo(t *testing.T) {
	ex := &mockExchangeVideo{}
	deps := mockDeps(t, ex)
	deps.cfg.VideoStoredRequestRequired = true

	req := openrtb_ext.BidRequestVideo{
		StoredRequestId: "123",
		PodConfig: openrtb_ext.PodConfig{
			DurationRangeSec:     []int{15, 30},
			RequireExactDuration: true,
			Pods: []openrtb_ext.Pod{
				{
					PodId:            1,
					AdPodDurationSec: 30,
					ConfigId:         "qwerty",
				},
				{
					PodId:            2,
					AdPodDurationSec: 30,
					ConfigId:         "qwerty",
				},
			},
		},
		App: &openrtb2.App{
			Bundle: "pbs.com",
		},
		IncludeBrandCategory: &openrtb_ext.IncludeBrandCategory{
			PrimaryAdserver: 1,
		},
	}

	errors, podErrors := deps.validateVideoRequest(&req)
	assert.Len(t, podErrors, 0, "Pod errors should be empty")
	assert.Len(t, errors, 1, "Errors array should contain 1 error message")
	assert.Equal(t, "request missing required field: Video", errors[0].Error(), "Errors array should contain message regarding missing Video field")
}

func TestVideoBuildVideoResponseMissedCacheForOneBid(t *testing.T) {
	openRtbBidResp := openrtb2.BidResponse{}
	podErrors := make([]PodError, 0)

	seatBids := make([]openrtb2.SeatBid, 0)
	seatBid := openrtb2.SeatBid{}

	bids := make([]openrtb2.Bid, 0)
	bid1 := openrtb2.Bid{}
	bid2 := openrtb2.Bid{}
	bid3 := openrtb2.Bid{}

	extBid1 := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"17.00","hb_pb_cat_dur_appnex":"17.00_123_30s","hb_size":"1x1","hb_uuid_appnexus":"837ea3b7-5598-4958-8c45-8e9ef2bf7cc1"}}}`)
	extBid2 := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"17.00","hb_pb_cat_dur_appnex":"17.00_456_30s","hb_size":"1x1","hb_uuid_appnexus":"837ea3b7-5598-4958-8c45-8e9ef2bf7cc1"}}}`)
	extBid3 := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"17.00","hb_pb_cat_dur_appnex":"17.00_406_30s","hb_size":"1x1"}}}`)

	bid1.Ext = extBid1
	bids = append(bids, bid1)

	bid2.Ext = extBid2
	bids = append(bids, bid2)

	bid3.Ext = extBid3
	bids = append(bids, bid3)

	seatBid.Bid = bids
	seatBid.Seat = "appnexus"
	seatBids = append(seatBids, seatBid)
	openRtbBidResp.SeatBid = seatBids

	bidRespVideo, err := buildVideoResponse(&openRtbBidResp, podErrors)
	assert.NoError(t, err, "Should be no error")
	assert.Len(t, bidRespVideo.AdPods, 1, "AdPods length should be 1")
	assert.Len(t, bidRespVideo.AdPods[0].Targeting, 2, "AdPod Targeting length should be 2")
	assert.Equal(t, "17.00_123_30s", bidRespVideo.AdPods[0].Targeting[0].HbPbCatDur, "AdPod Targeting first element hb_pb_cat_dur should be 17.00_123_30s")
	assert.Equal(t, "17.00_456_30s", bidRespVideo.AdPods[0].Targeting[1].HbPbCatDur, "AdPod Targeting first element hb_pb_cat_dur should be 17.00_456_30s")
}

func TestVideoBuildVideoResponseMissedCacheForAllBids(t *testing.T) {
	openRtbBidResp := openrtb2.BidResponse{}
	podErrors := make([]PodError, 0)

	seatBids := make([]openrtb2.SeatBid, 0)
	seatBid := openrtb2.SeatBid{}

	bids := make([]openrtb2.Bid, 0)
	bid1 := openrtb2.Bid{}
	bid2 := openrtb2.Bid{}
	bid3 := openrtb2.Bid{}

	extBid1 := []byte(`{"prebid":{"targeting":{"hb_bidder":"appnexus","hb_pb":"17.00","hb_pb_cat_dur":"17.00_123_30s","hb_size":"1x1"}}}`)
	extBid2 := []byte(`{"prebid":{"targeting":{"hb_bidder":"appnexus","hb_pb":"17.00","hb_pb_cat_dur":"17.00_456_30s","hb_size":"1x1"}}}`)
	extBid3 := []byte(`{"prebid":{"targeting":{"hb_bidder":"appnexus","hb_pb":"17.00","hb_pb_cat_dur":"17.00_406_30s","hb_size":"1x1"}}}`)

	bid1.Ext = extBid1
	bids = append(bids, bid1)

	bid2.Ext = extBid2
	bids = append(bids, bid2)

	bid3.Ext = extBid3
	bids = append(bids, bid3)

	seatBid.Bid = bids
	seatBids = append(seatBids, seatBid)
	openRtbBidResp.SeatBid = seatBids

	bidRespVideo, err := buildVideoResponse(&openRtbBidResp, podErrors)
	assert.Nil(t, bidRespVideo, "bid response should be nil")
	assert.Equal(t, "caching failed for all bids", err.Error(), "error should be caching failed for all bids")
}

func TestVideoBuildVideoResponsePodErrors(t *testing.T) {
	openRtbBidResp := openrtb2.BidResponse{}
	podErrors := make([]PodError, 0, 2)

	seatBids := make([]openrtb2.SeatBid, 0)
	seatBid := openrtb2.SeatBid{}

	bids := make([]openrtb2.Bid, 0)
	bid1 := openrtb2.Bid{}
	bid2 := openrtb2.Bid{}

	extBid1 := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"17.00","hb_pb_cat_dur_appnex":"17.00_123_30s","hb_size":"1x1","hb_uuid_appnexus":"837ea3b7-5598-4958-8c45-8e9ef2bf7cc1"}}}`)
	extBid2 := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"17.00","hb_pb_cat_dur_appnex":"17.00_456_30s","hb_size":"1x1","hb_uuid_appnexus":"837ea3b7-5598-4958-8c45-8e9ef2bf7cc1"}}}`)

	bid1.Ext = extBid1
	bids = append(bids, bid1)

	bid2.Ext = extBid2
	bids = append(bids, bid2)

	seatBid.Bid = bids
	seatBid.Seat = "appnexus"
	seatBids = append(seatBids, seatBid)
	openRtbBidResp.SeatBid = seatBids

	podErr1 := PodError{}
	podErr1.PodId = 222
	podErr1.PodIndex = 1
	podErrors = append(podErrors, podErr1)

	podErr2 := PodError{}
	podErr2.PodId = 333
	podErr2.PodIndex = 2
	podErrors = append(podErrors, podErr2)

	bidRespVideo, err := buildVideoResponse(&openRtbBidResp, podErrors)
	assert.NoError(t, err, "Error should be nil")
	assert.Len(t, bidRespVideo.AdPods, 3, "AdPods length should be 3")
	assert.Len(t, bidRespVideo.AdPods[0].Targeting, 2, "First ad pod should be correct and contain 2 targeting elements")
	assert.Equal(t, int64(222), bidRespVideo.AdPods[1].PodId, "AdPods should contain error element at index 1")
	assert.Equal(t, int64(333), bidRespVideo.AdPods[2].PodId, "AdPods should contain error element at index 2")
}

func TestVideoBuildVideoResponseNoBids(t *testing.T) {
	openRtbBidResp := openrtb2.BidResponse{}
	podErrors := make([]PodError, 0)
	openRtbBidResp.SeatBid = make([]openrtb2.SeatBid, 0)
	bidRespVideo, err := buildVideoResponse(&openRtbBidResp, podErrors)
	assert.NoError(t, err, "Error should be nil")
	assert.Len(t, bidRespVideo.AdPods, 0, "AdPods length should be 0")
}

func TestMergeOpenRTBToVideoRequest(t *testing.T) {
	var bidReq = &openrtb2.BidRequest{}
	var videoReq = &openrtb_ext.BidRequestVideo{}

	videoReq.App = &openrtb2.App{
		Domain: "test.com",
		Bundle: "test.bundle",
	}

	videoReq.Site = &openrtb2.Site{
		Page: "site.com/index",
	}

	var dnt int8 = 4
	var lmt int8 = 5
	videoReq.Device = openrtb2.Device{
		DNT: &dnt,
		Lmt: &lmt,
	}

	videoReq.BCat = []string{"test1", "test2"}
	videoReq.BAdv = []string{"test3", "test4"}

	videoReq.Regs = &openrtb2.Regs{
		Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"1NYY","existing":"any","consent":"anyConsent"}`),
	}

	videoReq.User = &openrtb2.User{
		BuyerUID: "test UID",
		Yob:      1980,
		Keywords: "test keywords",
		Ext:      json.RawMessage(`{"consent":"test string"}`),
	}

	mergeData(videoReq, bidReq)

	assert.Equal(t, videoReq.BCat, bidReq.BCat, "BCat is incorrect")
	assert.Equal(t, videoReq.BAdv, bidReq.BAdv, "BAdv is incorrect")

	assert.Equal(t, videoReq.App.Domain, bidReq.App.Domain, "App.Domain is incorrect")
	assert.Equal(t, videoReq.App.Bundle, bidReq.App.Bundle, "App.Bundle is incorrect")

	assert.Equal(t, videoReq.Device.Lmt, bidReq.Device.Lmt, "Device.Lmt is incorrect")
	assert.Equal(t, videoReq.Device.DNT, bidReq.Device.DNT, "Device.DNT is incorrect")

	assert.Equal(t, videoReq.Site.Page, bidReq.Site.Page, "Device.Site.Page is incorrect")

	assert.Equal(t, videoReq.Regs, bidReq.Regs, "Regs is incorrect")

	assert.Equal(t, videoReq.User, bidReq.User, "User is incorrect")
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		description       string
		giveErrors        []error
		wantCode          int
		wantMetricsStatus metrics.RequestStatus
	}{
		{
			description: "Blocked account - return 503 with blocked metrics status",
			giveErrors: []error{
				&errortypes.AccountDisabled{},
			},
			wantCode:          503,
			wantMetricsStatus: metrics.RequestStatusBlockedApp,
		},
		{
			description: "Blocked app - return 503 with blocked metrics status",
			giveErrors: []error{
				&errortypes.BlockedApp{},
			},
			wantCode:          503,
			wantMetricsStatus: metrics.RequestStatusBlockedApp,
		},
		{
			description: "Account required error - return 400 with bad input metrics status",
			giveErrors: []error{
				&errortypes.AcctRequired{},
			},
			wantCode:          400,
			wantMetricsStatus: metrics.RequestStatusBadInput,
		},
		{
			description: "Malformed account config error - return 500 with account config error metrics status",
			giveErrors: []error{
				&errortypes.MalformedAcct{},
			},
			wantCode:          500,
			wantMetricsStatus: metrics.RequestStatusAccountConfigErr,
		},
		{
			description: "Multiple generic errors - return 500 with generic error metrics status",
			giveErrors: []error{
				errors.New("Error for testing handleError 1"),
				errors.New("Error for testing handleError 2"),
			},
			wantCode:          500,
			wantMetricsStatus: metrics.RequestStatusErr,
		},
	}

	for _, tt := range tests {
		vo := analytics.VideoObject{
			Status: 200,
			Errors: make([]error, 0),
		}

		labels := metrics.Labels{
			Source:        metrics.DemandUnknown,
			RType:         metrics.ReqTypeVideo,
			PubID:         metrics.PublisherUnknown,
			CookieFlag:    metrics.CookieFlagUnknown,
			RequestStatus: metrics.RequestStatusOK,
		}

		recorder := httptest.NewRecorder()
		handleError(&labels, recorder, tt.giveErrors, &vo, nil)

		assert.Equal(t, tt.wantMetricsStatus, labels.RequestStatus, tt.description)
		assert.Equal(t, tt.wantCode, recorder.Code, tt.description)
		assert.Equal(t, tt.wantCode, vo.Status, tt.description)
		assert.ElementsMatch(t, tt.giveErrors, vo.Errors, tt.description)
	}
}

func TestHandleErrorMetrics(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_invalid_sample.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps, met, mod := mockDepsWithMetrics(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	assert.Equal(t, int64(0), met.RequestStatuses[metrics.ReqTypeVideo][metrics.RequestStatusOK].Count(), "OK requests count should be 0")
	assert.Equal(t, int64(1), met.RequestStatuses[metrics.ReqTypeVideo][metrics.RequestStatusErr].Count(), "Error requests count should be 1")
	assert.Equal(t, 1, len(mod.videoObjects), "Mock AnalyticsModule should have 1 AuctionObject")
	assert.Equal(t, 500, mod.videoObjects[0].Status, "AnalyticsObject should have 500 status")
	assert.Equal(t, 2, len(mod.videoObjects[0].Errors), "AnalyticsObject should have Errors length of 2")
	assert.Equal(t, "request missing required field: PodConfig.DurationRangeSec", mod.videoObjects[0].Errors[0].Error(), "First error in AnalyticsObject should have message regarding DurationRangeSec")
	assert.Equal(t, "request missing required field: PodConfig.Pods", mod.videoObjects[0].Errors[1].Error(), "Second error in AnalyticsObject should have message regarding Pods")
}

func TestParseVideoRequestWithUserAgentAndHeader(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_with_device_user_agent.json")
	headers := http.Header{}
	headers.Add("User-Agent", "TestHeader")

	deps := mockDeps(t, ex)
	req, valErr, podErr := deps.parseVideoRequest([]byte(reqBody), headers)

	assert.Equal(t, "TestHeaderSample", req.Device.UA, "Header should be taken from original request")
	assert.Equal(t, []error(nil), valErr, "No validation errors should be returned")
	assert.Equal(t, make([]PodError, 0), podErr, "No pod errors should be returned")
}

func TestParseVideoRequestWithUserAgentAndEmptyHeader(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_with_device_user_agent.json")

	headers := http.Header{}

	deps := mockDeps(t, ex)
	req, valErr, podErr := deps.parseVideoRequest([]byte(reqBody), headers)

	assert.Equal(t, "TestHeaderSample", req.Device.UA, "Header should be taken from original request")
	assert.Equal(t, []error(nil), valErr, "No validation errors should be returned")
	assert.Equal(t, make([]PodError, 0), podErr, "No pod errors should be returned")
}

func TestParseVideoRequestWithoutUserAgentWithHeader(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_without_device_user_agent.json")
	headers := http.Header{}
	headers.Add("User-Agent", "TestHeader")

	deps := mockDeps(t, ex)
	req, valErr, podErr := deps.parseVideoRequest([]byte(reqBody), headers)

	assert.Equal(t, "TestHeader", req.Device.UA, "Device.ua should be taken from request header")
	assert.Equal(t, []error(nil), valErr, "No validation errors should be returned")
	assert.Equal(t, make([]PodError, 0), podErr, "No pod errors should be returned")
}

func TestParseVideoRequestWithoutUserAgentAndEmptyHeader(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_without_device_user_agent.json")

	headers := http.Header{}

	deps := mockDeps(t, ex)

	req, valErr, podErr := deps.parseVideoRequest([]byte(reqBody), headers)

	assert.Equal(t, "", req.Device.UA, "Device.ua should be empty")
	assert.Equal(t, []error(nil), valErr, "No validation errors should be returned")
	assert.Equal(t, make([]PodError, 0), podErr, "No pod errors should be returned")
}

func TestParseVideoRequestWithEncodedUserAgentInHeader(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_without_device_user_agent.json")

	uaEncoded := "Mozilla%2F5.0%20%28Macintosh%3B%20Intel%20Mac%20OS%20X%2010_14_6%29%20AppleWebKit%2F537.36%20%28KHTML%2C%20like%20Gecko%29%20Chrome%2F78.0.3904.87%20Safari%2F537.36"
	uaDecoded := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.87 Safari/537.36"

	headers := http.Header{}
	headers.Add("User-Agent", uaEncoded)

	deps := mockDeps(t, ex)
	req, valErr, podErr := deps.parseVideoRequest([]byte(reqBody), headers)

	assert.Equal(t, uaDecoded, req.Device.UA, "Device.ua should be taken from request header")
	assert.Equal(t, []error(nil), valErr, "No validation errors should be returned")
	assert.Equal(t, make([]PodError, 0), podErr, "No pod errors should be returned")
}

func TestParseVideoRequestWithDecodedUserAgentInHeader(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_without_device_user_agent.json")

	uaDecoded := "Mozilla/5.0+(Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.87 Safari/537.36"

	headers := http.Header{}
	headers.Add("User-Agent", uaDecoded)

	deps := mockDeps(t, ex)
	req, valErr, podErr := deps.parseVideoRequest([]byte(reqBody), headers)

	assert.Equal(t, uaDecoded, req.Device.UA, "Device.ua should be taken from request header")
	assert.Equal(t, []error(nil), valErr, "No validation errors should be returned")
	assert.Equal(t, make([]PodError, 0), podErr, "No pod errors should be returned")
}

func TestHandleErrorDebugLog(t *testing.T) {
	vo := analytics.VideoObject{
		Status: 200,
		Errors: make([]error, 0),
	}

	labels := metrics.Labels{
		Source:        metrics.DemandUnknown,
		RType:         metrics.ReqTypeVideo,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
	}

	recorder := httptest.NewRecorder()
	err1 := errors.New("Error for testing handleError 1")
	err2 := errors.New("Error for testing handleError 2")
	debugLog := exchange.DebugLog{
		Enabled:   true,
		CacheType: prebid_cache_client.TypeXML,
		Data: exchange.DebugData{
			Request:  "test request string",
			Headers:  "test headers string",
			Response: "test response string",
		},
		TTL:                      int64(3600),
		Regexp:                   regexp.MustCompile(`[<>]`),
		DebugOverride:            false,
		DebugEnabledOrOverridden: true,
	}
	handleError(&labels, recorder, []error{err1, err2}, &vo, &debugLog)

	assert.Equal(t, metrics.RequestStatusErr, labels.RequestStatus, "labels.RequestStatus should indicate an error")
	assert.Equal(t, 500, recorder.Code, "Error status should be written to writer")
	assert.Equal(t, 500, vo.Status, "Analytics object should have error status")
	assert.Equal(t, 3, len(vo.Errors), "New errors including debug cache ID should be appended to Analytics object Errors")
	assert.Equal(t, "Error for testing handleError 1", vo.Errors[0].Error(), "Error in Analytics object should have test error message for first error")
	assert.Equal(t, "Error for testing handleError 2", vo.Errors[1].Error(), "Error in Analytics object should have test error message for second error")
	assert.NotEmpty(t, debugLog.CacheKey, "DebugLog CacheKey value should have been set")
}

func TestCreateImpressionTemplate(t *testing.T) {
	imp := openrtb2.Imp{}
	imp.Video = &openrtb2.Video{}
	imp.Video.Protocols = []adcom1.MediaCreativeSubtype{1, 2}
	imp.Video.MIMEs = []string{"video/mp4"}
	imp.Video.H = ptrutil.ToPtr[int64](200)
	imp.Video.W = ptrutil.ToPtr[int64](400)
	imp.Video.PlaybackMethod = []adcom1.PlaybackMethod{5, 6}

	video := openrtb2.Video{}
	video.Protocols = []adcom1.MediaCreativeSubtype{3, 4}
	video.MIMEs = []string{"video/flv"}
	video.H = ptrutil.ToPtr[int64](300)
	video.W = ptrutil.ToPtr[int64](0)
	video.PlaybackMethod = []adcom1.PlaybackMethod{7, 8}

	res := createImpressionTemplate(imp, &video)
	assert.Equal(t, []adcom1.MediaCreativeSubtype{3, 4}, res.Video.Protocols, "Incorrect video protocols")
	assert.Equal(t, []string{"video/flv"}, res.Video.MIMEs, "Incorrect video MIMEs")
	assert.Equal(t, ptrutil.ToPtr[int64](300), res.Video.H, "Incorrect video height")
	assert.Equal(t, ptrutil.ToPtr[int64](0), res.Video.W, "Incorrect video width")
	assert.Equal(t, []adcom1.PlaybackMethod{7, 8}, res.Video.PlaybackMethod, "Incorrect video playback method")
}

func TestCCPA(t *testing.T) {
	testCases := []struct {
		description         string
		testFilePath        string
		expectConsentString bool
		expectEmptyConsent  bool
	}{
		{
			description:         "Missing Consent",
			testFilePath:        "sample-requests/video/video_valid_sample.json",
			expectConsentString: false,
			expectEmptyConsent:  true,
		},
		{
			description:         "Valid Consent",
			testFilePath:        "sample-requests/video/video_valid_sample_ccpa_valid.json",
			expectConsentString: true,
		},
		{
			description:         "Malformed Consent",
			testFilePath:        "sample-requests/video/video_valid_sample_ccpa_malformed.json",
			expectConsentString: false,
		},
	}

	for _, test := range testCases {
		reqBody := readVideoTestFile(t, test.testFilePath)

		// Create HTTP Request + Response Recorder
		httpRequest := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
		httpResponseRecorder := httptest.NewRecorder()

		// Run Test
		ex := &mockExchangeVideo{}
		mockDeps(t, ex).VideoAuctionEndpoint(httpResponseRecorder, httpRequest, nil)

		// Validate Request To Exchange
		// - An error should never be generated for CCPA problems.
		if ex.lastRequest == nil {
			t.Fatalf("%s: The request never made it into the exchange.", test.description)
		}

		if test.expectConsentString {
			assert.Len(t, ex.lastRequest.Regs.USPrivacy, 4, test.description+":consent")
		} else if test.expectEmptyConsent {
			assert.Empty(t, ex.lastRequest.Regs.USPrivacy, test.description+":consent")
		}

		// Validate HTTP Response
		responseBytes := httpResponseRecorder.Body.Bytes()
		response := &openrtb_ext.BidResponseVideo{}
		if err := jsonutil.UnmarshalValid(responseBytes, response); err != nil {
			t.Fatalf("%s: Unable to unmarshal response.", test.description)
		}
		assert.Len(t, ex.lastRequest.Imp, 11, test.description+":imps")
		assert.Len(t, response.AdPods, 5, test.description+":adpods")
	}
}

func TestVideoEndpointAppendBidderNames(t *testing.T) {
	ex := &mockExchangeAppendBidderNames{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_valid_sample_appendbiddernames.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDepsAppendBidderNames(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	var extData openrtb_ext.ExtRequest
	jsonutil.UnmarshalValid(ex.lastRequest.Ext, &extData)
	assert.True(t, extData.Prebid.Targeting.AppendBidderNames, "Request ext incorrect: AppendBidderNames should be true ")

	respBytes := recorder.Body.Bytes()
	resp := &openrtb_ext.BidResponseVideo{}
	if err := jsonutil.UnmarshalValid(respBytes, resp); err != nil {
		t.Fatalf("Unable to unmarshal response.")
	}

	assert.Len(t, ex.lastRequest.Imp, 11, "Incorrect number of impressions in request")
	assert.Equal(t, "prebid.com", string(ex.lastRequest.Site.Page), "Incorrect site page in request")
	assert.Equal(t, "TvName", ex.lastRequest.Site.Content.Series, "Incorrect site content series in request")

	assert.Len(t, resp.AdPods, 5, "Incorrect number of Ad Pods in response")
	assert.Len(t, resp.AdPods[0].Targeting, 4, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[1].Targeting, 3, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[2].Targeting, 5, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[3].Targeting, 1, "Incorrect Targeting data in response")
	assert.Len(t, resp.AdPods[4].Targeting, 3, "Incorrect Targeting data in response")

	assert.Equal(t, "20.00_395_30s_appnexus", resp.AdPods[4].Targeting[0].HbPbCatDur, "Incorrect number of Ad Pods in response")
}

func TestFormatTargetingKey(t *testing.T) {
	res := formatTargetingKey(openrtb_ext.CategoryDurationKey, "appnexus")
	assert.Equal(t, "_pb_cat_dur_appnexus", res, "Tergeting key constructed incorrectly")
}

func TestFormatTargetingKeyLongKey(t *testing.T) {
	res := formatTargetingKey(openrtb_ext.PbKey, "20.00")
	assert.Equal(t, "_pb_20.00", res, "Tergeting key constructed incorrectly")
}

func TestFindTargetingByKey(t *testing.T) {
	tests := []struct {
		name             string
		targetingMap     map[string]string
		keyWithoutPrefix string
		expectedResult   string
	}{
		{
			name: "Correct match",
			targetingMap: map[string]string{
				"hb_key": "hb_key12345454",
			},
			keyWithoutPrefix: "_key12345454",
			expectedResult:   "hb_key12345454",
		},
		{
			name: "Dismatching",
			targetingMap: map[string]string{
				"hb_key": "hb_key12345454",
			},
			keyWithoutPrefix: "12345454",
			expectedResult:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTargetingByKey(tt.targetingMap, tt.keyWithoutPrefix)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestVideoAuctionResponseHeaders(t *testing.T) {
	testCases := []struct {
		description     string
		givenTestFile   string
		givenHeader     map[string]string
		expectedStatus  int
		expectedHeaders func(http.Header)
	}{
		{
			description:    "Success Response",
			givenTestFile:  "sample-requests/video/video_valid_sample.json",
			expectedStatus: 200,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Content-Type", "application/json")
			},
		},
		{
			description:    "Failure Response",
			givenTestFile:  "sample-requests/video/video_invalid_sample.json",
			expectedStatus: 500,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
			},
		},
		{
			description:    "Success Response with header Observe-Browsing-Topics",
			givenTestFile:  "sample-requests/video/video_valid_sample.json",
			givenHeader:    map[string]string{secBrowsingTopics: "anyValue"},
			expectedStatus: 200,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Content-Type", "application/json")
				h.Set("Observe-Browsing-Topics", "?1")
			},
		},
		{
			description:    "Failure Response with header Observe-Browsing-Topics",
			givenTestFile:  "sample-requests/video/video_invalid_sample.json",
			givenHeader:    map[string]string{secBrowsingTopics: "anyValue"},
			expectedStatus: 500,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Observe-Browsing-Topics", "?1")
			},
		},
	}

	exchange := &mockExchangeVideo{}
	endpoint := mockDeps(t, exchange)

	for _, test := range testCases {
		requestBody := readVideoTestFile(t, test.givenTestFile)

		httpReq := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(requestBody))
		for k, v := range test.givenHeader {
			httpReq.Header.Add(k, v)
		}
		recorder := httptest.NewRecorder()

		endpoint.VideoAuctionEndpoint(recorder, httpReq, nil)

		expectedHeaders := http.Header{}
		test.expectedHeaders(expectedHeaders)

		assert.Equal(t, test.expectedStatus, recorder.Result().StatusCode, test.description+":statuscode")
		assert.Equal(t, expectedHeaders, recorder.Result().Header, test.description+":statuscode")
	}
}

func mockDepsWithMetrics(t *testing.T, ex *mockExchangeVideo) (*endpointDeps, *metrics.Metrics, *mockAnalyticsModule) {
	mockModule := &mockAnalyticsModule{}

	metrics := metrics.NewMetrics(gometrics.NewRegistry(), openrtb_ext.CoreBidderNames(), config.DisabledMetrics{}, nil, nil)

	deps := &endpointDeps{
		fakeUUIDGenerator{},
		ex,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockVideoStoredReqFetcher{},
		&mockVideoStoredReqFetcher{},
		&mockAccountFetcher{data: mockVideoAccountData},
		&config.Configuration{MaxRequestSize: maxSize},
		metrics,
		mockModule,
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}
	return deps, metrics, mockModule
}

type mockAnalyticsModule struct {
	auctionObjects []*analytics.AuctionObject
	videoObjects   []*analytics.VideoObject
}

func (m *mockAnalyticsModule) LogAuctionObject(ao *analytics.AuctionObject, _ privacy.ActivityControl) {
	m.auctionObjects = append(m.auctionObjects, ao)
}

func (m *mockAnalyticsModule) LogVideoObject(vo *analytics.VideoObject, _ privacy.ActivityControl) {
	m.videoObjects = append(m.videoObjects, vo)
}

func (m *mockAnalyticsModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {}

func (m *mockAnalyticsModule) LogSetUIDObject(so *analytics.SetUIDObject) {}

func (m *mockAnalyticsModule) LogAmpObject(ao *analytics.AmpObject, _ privacy.ActivityControl) {
}

func (m *mockAnalyticsModule) LogNotificationEventObject(ne *analytics.NotificationEvent, _ privacy.ActivityControl) {
}

func (m *mockAnalyticsModule) Shutdown() {}

func mockDeps(t *testing.T, ex *mockExchangeVideo) *endpointDeps {
	return &endpointDeps{
		fakeUUIDGenerator{},
		ex,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockVideoStoredReqFetcher{},
		&mockVideoStoredReqFetcher{},
		&mockAccountFetcher{data: mockVideoAccountData},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		ex.cache,
		regexp.MustCompile(`[<>]`),
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}
}

func mockDepsAppendBidderNames(t *testing.T, ex *mockExchangeAppendBidderNames) *endpointDeps {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		ex,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockVideoStoredReqFetcher{},
		&mockVideoStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		ex.cache,
		regexp.MustCompile(`[<>]`),
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	return deps
}

func mockDepsNoBids(t *testing.T, ex *mockExchangeVideoNoBids) *endpointDeps {
	edep := &endpointDeps{
		fakeUUIDGenerator{},
		ex,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockVideoStoredReqFetcher{},
		&mockVideoStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		ex.cache,
		regexp.MustCompile(`[<>]`),
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	return edep
}

type mockCacheClient struct {
	called bool
}

func (m *mockCacheClient) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	if !m.called {
		m.called = true
	}
	return []string{}, []error{}
}

func (m *mockCacheClient) GetExtCacheData() (scheme string, host string, path string) {
	return "", "", ""
}

type mockVideoStoredReqFetcher struct {
}

func (cf mockVideoStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return testVideoStoredRequestData, testVideoStoredImpData, nil
}

func (cf mockVideoStoredReqFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return nil, nil
}

type mockExchangeVideo struct {
	lastRequest *openrtb2.BidRequest
	cache       *mockCacheClient
}

func (m *mockExchangeVideo) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*exchange.AuctionResponse, error) {
	if err := r.BidRequestWrapper.RebuildRequest(); err != nil {
		return nil, err
	}

	m.lastRequest = r.BidRequestWrapper.BidRequest
	if debugLog != nil && debugLog.Enabled {
		m.cache.called = true
	}
	ext := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"20.00","hb_pb_cat_dur_appnex":"20.00_395_30s","hb_size":"1x1", "hb_uuid_appnexus":"837ea3b7-5598-4958-8c45-8e9ef2bf7cc1", "hb_deal_appnexus": "ABC_123"},"type":"video","dealpriority":0,"dealtiersatisfied":false},"bidder":{"appnexus":{"brand_id":1,"auction_id":7840037870526938650,"bidder_id":2,"bid_ad_type":1,"creative_info":{"video":{"duration":30,"mimes":["video\/mp4"]}}}}}`)
	return &exchange.AuctionResponse{BidResponse: &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Seat: "appnexus",
			Bid: []openrtb2.Bid{
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
	}}, nil
}

type mockExchangeAppendBidderNames struct {
	lastRequest *openrtb2.BidRequest
	cache       *mockCacheClient
}

func (m *mockExchangeAppendBidderNames) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*exchange.AuctionResponse, error) {
	m.lastRequest = r.BidRequestWrapper.BidRequest
	if debugLog != nil && debugLog.Enabled {
		m.cache.called = true
	}
	ext := []byte(`{"prebid":{"targeting":{"hb_bidder_appnexus":"appnexus","hb_pb_appnexus":"20.00","hb_pb_cat_dur_appnex":"20.00_395_30s_appnexus","hb_size":"1x1", "hb_uuid_appnexus":"837ea3b7-5598-4958-8c45-8e9ef2bf7cc1"},"type":"video"},"bidder":{"appnexus":{"brand_id":1,"auction_id":7840037870526938650,"bidder_id":2,"bid_ad_type":1,"creative_info":{"video":{"duration":30,"mimes":["video\/mp4"]}}}}}`)
	return &exchange.AuctionResponse{BidResponse: &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Seat: "appnexus",
			Bid: []openrtb2.Bid{
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
		}}},
	}, nil
}

type mockExchangeVideoNoBids struct {
	lastRequest *openrtb2.BidRequest
	cache       *mockCacheClient
}

func (m *mockExchangeVideoNoBids) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*exchange.AuctionResponse, error) {
	m.lastRequest = r.BidRequestWrapper.BidRequest
	return &exchange.AuctionResponse{BidResponse: &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{}},
	}}, nil
}

var mockVideoAccountData = map[string]json.RawMessage{
	"valid_acct":     json.RawMessage(`{"disabled":false}`),
	"disabled_acct":  json.RawMessage(`{"disabled":true}`),
	"malformed_acct": json.RawMessage(`{"disabled":"invalid type"}`),
}

var testVideoStoredImpData = map[string]json.RawMessage{
	"fba10607-0c12-43d1-ad07-b8a513bc75d6": json.RawMessage(`{"ext": {"appnexus": {"placementId": 14997137}}}`),
	"8b452b41-2681-4a20-9086-6f16ffad7773": json.RawMessage(`{"ext": {"appnexus": {"placementId": 15016213}}}`),
	"87d82a45-35c3-46cc-9315-2e3eeb91d0f2": json.RawMessage(`{"ext": {"appnexus": {"placementId": 15062775}}}`),
}

var testVideoStoredRequestData = map[string]json.RawMessage{
	"80ce30c53c16e6ede735f123ef6e32361bfc7b22": json.RawMessage(`{"accountid": "11223344", "site": {"page": "mygame.foo.com"}}`),
}

func readVideoTestFile(t *testing.T, filename string) string {
	requestData, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}

	return string(getRequestPayload(t, requestData))
}

func TestVideoRequestValidationFailed(t *testing.T) {
	ex := &mockExchangeVideo{}
	reqBody := readVideoTestFile(t, "sample-requests/video/video_invalid_sample_negative_tmax.json")
	req := httptest.NewRequest("POST", "/openrtb2/video", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps := mockDeps(t, ex)
	deps.VideoAuctionEndpoint(recorder, req, nil)

	errorMessage := recorder.Body.String()

	assert.Equal(t, 500, recorder.Code, "Should catch error in request")
	assert.Equal(t, "Critical error while running the video endpoint:  request.tmax must be nonnegative. Got -2", errorMessage, "Incorrect request validation message")
}
