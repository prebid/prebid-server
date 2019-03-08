package openrtb2

import (
	"context"
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
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

func mockDeps(t *testing.T, ex *mockExchangeVideo) *endpointDeps {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	edep := &endpointDeps{
		ex,
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil}

	return edep
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
