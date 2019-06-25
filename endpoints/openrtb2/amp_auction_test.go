package openrtb2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"

	"github.com/mxmCherry/openrtb"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

// From auction_test.go
// const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted stored bids correctly.
func TestGoodAmpRequests(t *testing.T) {
	goodRequests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "aliased-buyeruids.json")),
		"2": json.RawMessage(validRequest(t, "aliases.json")),
		"4": json.RawMessage(validRequest(t, "digitrust.json")),
		"5": json.RawMessage(validRequest(t, "gdpr-no-consentstring.json")),
		"6": json.RawMessage(validRequest(t, "gdpr.json")),
		"7": json.RawMessage(validRequest(t, "site.json")),
		"9": json.RawMessage(validRequest(t, "user.json")),
	}

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{goodRequests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)

	for requestID := range goodRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", string(goodRequests[requestID]))
		}

		var response AmpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.Targeting == nil || len(response.Targeting) == 0 {
			t.Errorf("Bad response, no targeting data.\n Response was: %v", recorder.Body)
		}
		if len(response.Targeting) != 3 {
			t.Errorf("Bad targeting data. Expected 3 keys, got %d.", len(response.Targeting))
		}

		if response.Debug != nil {
			t.Errorf("Debug present but not requested")
		}
		if _, ok := response.Errors[openrtb_ext.BidderOpenx]; !ok {
			t.Errorf("OpenX error message is not present. (%v)", response.Errors)
		}
	}
}

// Prevents #683
func TestAMPPageInfo(t *testing.T) {
	const page = "http://test.somepage.co.uk:1234?myquery=1&other=2"
	stored := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	exchange := &mockAmpExchange{}

	endpoint, _ := NewAmpEndpoint(
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)
	request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&curl=%s", url.QueryEscape(page)), nil)
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !assert.NotNil(t, exchange.lastRequest, "Endpoint responded with %d: %s", recorder.Code, recorder.Body.String()) {
		return
	}
	if !assert.NotNil(t, exchange.lastRequest.Site) {
		return
	}
	assert.Equal(t, page, exchange.lastRequest.Site.Page)
	assert.Equal(t, "test.somepage.co.uk", exchange.lastRequest.Site.Domain)
}

func TestAMPSiteExt(t *testing.T) {
	stored := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	exchange := &mockAmpExchange{}
	endpoint, _ := NewAmpEndpoint(
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		nil,
		nil,
		openrtb_ext.BidderMap,
	)
	request, err := http.NewRequest("GET", "/openrtb2/auction/amp?tag_id=1", nil)
	if !assert.NoError(t, err) {
		return
	}
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !assert.NotNil(t, exchange.lastRequest, "Endpoint responded with %d: %s", recorder.Code, recorder.Body.String()) {
		return
	}
	if !assert.NotNil(t, exchange.lastRequest.Site) {
		return
	}
	assert.JSONEq(t, `{"amp":1}`, string(exchange.lastRequest.Site.Ext))
}

// TestBadRequests makes sure we return 400's on bad requests.
func TestAmpBadRequests(t *testing.T) {
	files := fetchFiles(t, "sample-requests/invalid-whole")
	badRequests := make(map[string]json.RawMessage, len(files))
	for index, file := range files {
		badRequests[strconv.Itoa(100+index)] = readFile(t, "sample-requests/invalid-whole/"+file.Name())
	}

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())

	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{badRequests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)
	for requestID := range badRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()

		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, fmt.Sprintf("/openrtb2/auction/amp?config=%s", requestID))
		}
	}
}

// TestAmpDebug makes sure we get debug information back when requested
func TestAmpDebug(t *testing.T) {
	requests := map[string]json.RawMessage{
		"2": json.RawMessage(validRequest(t, "site.json")),
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)

	for requestID := range requests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s&debug=1", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", string(requests[requestID]))
		}

		var response AmpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.Targeting == nil || len(response.Targeting) == 0 {
			t.Errorf("Bad response, no targeting data.\n Response was: %v", recorder.Body)
		}
		if len(response.Targeting) != 3 {
			t.Errorf("Bad targeting data. Expected 3 keys, got %d.", len(response.Targeting))
		}

		if response.Debug == nil {
			t.Errorf("Debug requested but not present")
		}
	}
}

// Prevents #452
func TestAmpTargetingDefaults(t *testing.T) {
	req := &openrtb.BidRequest{}
	if errs := defaultRequestExt(req); len(errs) != 0 {
		t.Fatalf("Unexpected error defaulting request.ext for AMP: %v", errs)
	}

	var extRequest openrtb_ext.ExtRequest
	if err := json.Unmarshal(req.Ext, &extRequest); err != nil {
		t.Fatalf("Unexpected error unmarshalling defaulted request.ext for AMP: %v", err)
	}
	if extRequest.Prebid.Targeting == nil {
		t.Fatal("AMP defaults should set request.ext.targeting")
	}
	if !extRequest.Prebid.Targeting.IncludeWinners {
		t.Error("AMP defaults should set request.ext.targeting.includewinners to true")
	}
	if !extRequest.Prebid.Targeting.IncludeBidderKeys {
		t.Error("AMP defaults should set request.ext.targeting.includebidderkeys to true")
	}
	if !reflect.DeepEqual(extRequest.Prebid.Targeting.PriceGranularity, openrtb_ext.PriceGranularityFromString("med")) {
		t.Error("AMP defaults should set request.ext.targeting.pricegranularity to medium")
	}
}

func TestQueryParamOverrides(t *testing.T) {
	requests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())

	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)

	requestID := "1"
	curl := "http://example.com"
	slot := "1234"
	timeout := int64(500)

	request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s&debug=1&curl=%s&slot=%s&timeout=%d", requestID, curl, slot, timeout), nil)
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
		t.Errorf("Response body was: %s", recorder.Body)
		t.Errorf("Request was: %s", string(requests[requestID]))
	}

	var response AmpResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Error unmarshalling response: %s", err.Error())
	}

	resolvedRequest := response.Debug.ResolvedRequest
	if resolvedRequest.TMax != timeout {
		t.Errorf("Expected TMax to equal timeout (%d), got: %d", timeout, resolvedRequest.TMax)
	}

	resolvedImp := resolvedRequest.Imp[0]
	if resolvedImp.TagID != slot {
		t.Errorf("Expected Imp.TagId to equal slot (%s), got: %s", slot, resolvedImp.TagID)
	}

	if resolvedRequest.Site == nil || resolvedRequest.Site.Page != curl {
		t.Errorf("Expected Site.Page to equal curl (%s), got: %s", curl, resolvedRequest.Site.Page)
	}
}

func TestOverrideDimensions(t *testing.T) {
	formatOverrideSpec{
		overrideWidth:  20,
		overrideHeight: 40,
		expect: []openrtb.Format{{
			W: 20,
			H: 40,
		}},
	}.execute(t)
}

func TestOverrideHeightNormalWidth(t *testing.T) {
	formatOverrideSpec{
		width:          20,
		overrideHeight: 40,
		expect: []openrtb.Format{{
			W: 20,
			H: 40,
		}},
	}.execute(t)
}

func TestOverrideWidthNormalHeight(t *testing.T) {
	formatOverrideSpec{
		overrideWidth: 20,
		height:        40,
		expect: []openrtb.Format{{
			W: 20,
			H: 40,
		}},
	}.execute(t)
}

func TestMultisize(t *testing.T) {
	formatOverrideSpec{
		multisize: "200x50,100x60",
		expect: []openrtb.Format{{
			W: 200,
			H: 50,
		}, {
			W: 100,
			H: 60,
		}},
	}.execute(t)
}

func TestHeightOnly(t *testing.T) {
	formatOverrideSpec{
		height: 200,
		expect: []openrtb.Format{{
			W: 300,
			H: 200,
		}},
	}.execute(t)
}

func TestWidthOnly(t *testing.T) {
	formatOverrideSpec{
		width: 150,
		expect: []openrtb.Format{{
			W: 150,
			H: 600,
		}},
	}.execute(t)
}

type formatOverrideSpec struct {
	width          uint64
	height         uint64
	overrideWidth  uint64
	overrideHeight uint64
	multisize      string
	expect         []openrtb.Format
}

func (s formatOverrideSpec) execute(t *testing.T) {
	requests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		theMetrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)

	url := fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&debug=1&w=%d&h=%d&ow=%d&oh=%d&ms=%s", s.width, s.height, s.overrideWidth, s.overrideHeight, s.multisize)
	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d. Got %d. Request config ID was 1", http.StatusOK, recorder.Code)
		t.Errorf("Response body was: %s", recorder.Body)
		t.Errorf("Request was: %s", string(requests["1"]))
	}
	var response AmpResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Error unmarshalling response: %s", err.Error())
	}

	formats := response.Debug.ResolvedRequest.Imp[0].Banner.Format
	if len(formats) != len(s.expect) {
		t.Fatalf("Bad formats length. Expected %v, got %v", s.expect, formats)
	}
	for i := 0; i < len(formats); i++ {
		if formats[i].W != s.expect[i].W {
			t.Errorf("format[%d].W were not equal. Expected %d, got %d", i, s.expect[i].W, formats[i].W)
		}
		if formats[i].H != s.expect[i].H {
			t.Errorf("format[%d].H were not equal. Expected %d, got %d", i, s.expect[i].H, formats[i].H)
		}
	}
}

type mockAmpStoredReqFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockAmpStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return cf.data, nil, nil
}

type mockAmpExchange struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockAmpExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest

	response := &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
				Ext: json.RawMessage(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
			}},
		}},
		Ext: json.RawMessage(`{ "errors": {"openx":[ { "code": 1, "message": "The request exceeded the timeout allocated" } ] } }`),
	}

	if bidRequest.Test == 1 {
		resolvedRequest, err := json.Marshal(bidRequest)
		if err != nil {
			resolvedRequest = json.RawMessage("{}")
		}
		response.Ext = json.RawMessage(fmt.Sprintf(`{"debug": {"httpcalls": {}, "resolvedrequest": %s}}`, resolvedRequest))
	}

	return response, nil
}
