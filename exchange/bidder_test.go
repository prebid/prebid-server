package exchange

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	nativeRequests "github.com/prebid/openrtb/v19/native1/request"
	nativeResponse "github.com/prebid/openrtb/v19/native1/response"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/experiment/adscert"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestSingleBidder makes sure that the following things work if the Bidder needs only one request.
//
// 1. The Bidder implementation is called with the arguments we expect.
// 2. The returned values are correct for a non-test bid.
func TestSingleBidder(t *testing.T) {
	type aTest struct {
		debugInfo    *config.DebugInfo
		httpCallsLen int
	}

	testCases := []*aTest{
		{&config.DebugInfo{Allow: false}, 0},
		{&config.DebugInfo{Allow: true}, 1},
	}

	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidAdjustments := map[string]float64{"test": 2.0}
	firstInitialPrice := 3.0
	secondInitialPrice := 4.0

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: nil,
	}

	ctx := context.Background()

	for _, test := range testCases {
		mockBidderResponse := &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						Price: firstInitialPrice,
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
				},
				{
					Bid: &openrtb2.Bid{
						Price: secondInitialPrice,
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
				},
			},
		}
		bidderImpl.bidResponse = mockBidderResponse

		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, test.debugInfo, "")
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		bidderReq := BidderRequest{
			BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
			BidderName: "test",
		}
		bidReqOptions := bidRequestOptions{
			accountDebugAllowed: true,
			headerDebugAllowed:  false,
			addCallSignHeader:   false,
			bidAdjustments:      bidAdjustments,
		}
		extraInfo := &adapters.ExtraRequestInfo{}
		seatBids, errs := bidder.requestBid(ctx, bidderReq, currencyConverter.Rates(), extraInfo, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})
		assert.NotEmpty(t, extraInfo.MakeBidsTimeInfo.Durations)
		assert.False(t, extraInfo.MakeBidsTimeInfo.AfterMakeBidsStartTime.IsZero())

		assert.Len(t, seatBids, 1)
		seatBid := seatBids[0]

		// Make sure the goodSingleBidder was called with the expected arguments.
		if bidderImpl.httpResponse == nil {
			t.Errorf("The Bidder should be called with the server's response.")
		}
		if bidderImpl.httpResponse.StatusCode != respStatus {
			t.Errorf("Bad response status. Expected %d, got %d", respStatus, bidderImpl.httpResponse.StatusCode)
		}
		if string(bidderImpl.httpResponse.Body) != respBody {
			t.Errorf("Bad response body. Expected %s, got %s", respBody, string(bidderImpl.httpResponse.Body))
		}

		// Make sure the returned values are what we expect
		if len(errortypes.FatalOnly(errs)) != 0 {
			t.Errorf("bidder.Bid returned %d errors. Expected 0", len(errs))
		}

		if !test.debugInfo.Allow && len(errortypes.WarningOnly(errs)) != 1 {
			t.Errorf("bidder.Bid returned %d warnings. Expected 1", len(errs))
		}
		if len(seatBid.Bids) != len(mockBidderResponse.Bids) {
			t.Fatalf("Expected %d bids. Got %d", len(mockBidderResponse.Bids), len(seatBid.Bids))
		}
		for index, typedBid := range mockBidderResponse.Bids {
			if typedBid.Bid != seatBid.Bids[index].Bid {
				t.Errorf("Bid %d did not point to the same bid returned by the Bidder.", index)
			}
			if typedBid.BidType != seatBid.Bids[index].BidType {
				t.Errorf("Bid %d did not have the right type. Expected %s, got %s", index, typedBid.BidType, seatBid.Bids[index].BidType)
			}
			if typedBid.DealPriority != seatBid.Bids[index].DealPriority {
				t.Errorf("Bid %d did not have the right deal priority. Expected %s, got %s", index, typedBid.BidType, seatBid.Bids[index].BidType)
			}
		}
		bidAdjustment := bidAdjustments["test"]
		if mockBidderResponse.Bids[0].Bid.Price != bidAdjustment*firstInitialPrice {
			t.Errorf("Bid[0].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*firstInitialPrice, mockBidderResponse.Bids[0].Bid.Price)
		}
		if mockBidderResponse.Bids[1].Bid.Price != bidAdjustment*secondInitialPrice {
			t.Errorf("Bid[1].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*secondInitialPrice, mockBidderResponse.Bids[1].Bid.Price)
		}
		if len(seatBid.HttpCalls) != test.httpCallsLen {
			t.Errorf("The bidder shouldn't log HttpCalls when request.test == 0. Found %d", len(seatBid.HttpCalls))
		}
		for index, bid := range seatBid.Bids {
			assert.NotEqual(t, mockBidderResponse.Bids[index].Bid.Price, bid.OriginalBidCPM, "The bid price was adjusted, so the originally bid CPM should be different")
		}
	}
}

func TestSingleBidderGzip(t *testing.T) {
	type aTest struct {
		debugInfo    *config.DebugInfo
		httpCallsLen int
	}

	testCases := []*aTest{
		{&config.DebugInfo{Allow: false}, 0},
		{&config.DebugInfo{Allow: true}, 1},
	}

	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidAdjustments := map[string]float64{"test": 2.0}
	firstInitialPrice := 3.0
	secondInitialPrice := 4.0

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte(`{"key":"val"}`),
			Headers: http.Header{},
		},
		bidResponse: nil,
	}

	ctx := context.Background()

	for _, test := range testCases {
		mockBidderResponse := &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						Price: firstInitialPrice,
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
				},
				{
					Bid: &openrtb2.Bid{
						Price: secondInitialPrice,
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
				},
			},
		}
		bidderImpl.bidResponse = mockBidderResponse

		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, test.debugInfo, "GZIP")
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		bidderReq := BidderRequest{
			BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
			BidderName: "test",
		}
		bidReqOptions := bidRequestOptions{
			accountDebugAllowed: true,
			headerDebugAllowed:  false,
			addCallSignHeader:   false,
			bidAdjustments:      bidAdjustments,
		}
		extraInfo := &adapters.ExtraRequestInfo{}
		seatBids, errs := bidder.requestBid(ctx, bidderReq, currencyConverter.Rates(), extraInfo, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})
		assert.NotEmpty(t, extraInfo.MakeBidsTimeInfo.Durations)
		assert.False(t, extraInfo.MakeBidsTimeInfo.AfterMakeBidsStartTime.IsZero())
		assert.Len(t, seatBids, 1)
		seatBid := seatBids[0]

		// Make sure the goodSingleBidder was called with the expected arguments.
		if bidderImpl.httpResponse == nil {
			t.Errorf("The Bidder should be called with the server's response.")
		}
		if bidderImpl.httpResponse.StatusCode != respStatus {
			t.Errorf("Bad response status. Expected %d, got %d", respStatus, bidderImpl.httpResponse.StatusCode)
		}
		if string(bidderImpl.httpResponse.Body) != respBody {
			t.Errorf("Bad response body. Expected %s, got %s", respBody, string(bidderImpl.httpResponse.Body))
		}

		// Make sure the returned values are what we expect
		if len(errortypes.FatalOnly(errs)) != 0 {
			t.Errorf("bidder.Bid returned %d errors. Expected 0", len(errs))
		}

		if !test.debugInfo.Allow && len(errortypes.WarningOnly(errs)) != 1 {
			t.Errorf("bidder.Bid returned %d warnings. Expected 1", len(errs))
		}
		if len(seatBid.Bids) != len(mockBidderResponse.Bids) {
			t.Fatalf("Expected %d bids. Got %d", len(mockBidderResponse.Bids), len(seatBid.Bids))
		}
		for index, typedBid := range mockBidderResponse.Bids {
			if typedBid.Bid != seatBid.Bids[index].Bid {
				t.Errorf("Bid %d did not point to the same bid returned by the Bidder.", index)
			}
			if typedBid.BidType != seatBid.Bids[index].BidType {
				t.Errorf("Bid %d did not have the right type. Expected %s, got %s", index, typedBid.BidType, seatBid.Bids[index].BidType)
			}
			if typedBid.DealPriority != seatBid.Bids[index].DealPriority {
				t.Errorf("Bid %d did not have the right deal priority. Expected %s, got %s", index, typedBid.BidType, seatBid.Bids[index].BidType)
			}
		}
		bidAdjustment := bidAdjustments["test"]
		if mockBidderResponse.Bids[0].Bid.Price != bidAdjustment*firstInitialPrice {
			t.Errorf("Bid[0].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*firstInitialPrice, mockBidderResponse.Bids[0].Bid.Price)
		}
		if mockBidderResponse.Bids[1].Bid.Price != bidAdjustment*secondInitialPrice {
			t.Errorf("Bid[1].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*secondInitialPrice, mockBidderResponse.Bids[1].Bid.Price)
		}
		if len(seatBid.HttpCalls) != test.httpCallsLen {
			t.Errorf("The bidder shouldn't log HttpCalls when request.test == 0. Found %d", len(seatBid.HttpCalls))
		}
		if test.debugInfo.Allow && len(seatBid.HttpCalls) > 0 {
			assert.Equalf(t, "gzip", seatBid.HttpCalls[0].RequestHeaders["Content-Encoding"][0], "Mismatched headers")
			assert.Equalf(t, "{\"key\":\"val\"}", seatBid.HttpCalls[0].RequestBody, "Mismatched request bodies")
		}
		for index, bid := range seatBid.Bids {
			assert.NotEqual(t, mockBidderResponse.Bids[index].Bid.Price, bid.OriginalBidCPM, "The bid price was adjusted, so the originally bid CPM should be different")
		}
	}
}

func TestRequestBidRemovesSensitiveHeaders(t *testing.T) {
	server := httptest.NewServer(mockHandler(200, "getBody", "responseJson"))
	defer server.Close()

	oldVer := version.Ver
	version.Ver = "test-version"
	defer func() {
		version.Ver = oldVer
	}()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")
	requestHeaders.Add("Authorization", "anySecret")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("requestJson"),
			Headers: requestHeaders,
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{},
		},
	}

	debugInfo := &config.DebugInfo{Allow: true}
	ctx := context.Background()

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, debugInfo, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: "test",
	}
	bidAdjustments := map[string]float64{"test": 1}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  false,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	extraInfo := &adapters.ExtraRequestInfo{}
	seatBids, errs := bidder.requestBid(ctx, bidderReq, currencyConverter.Rates(), extraInfo, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})
	expectedHttpCalls := []*openrtb_ext.ExtHttpCall{
		{
			Uri:            server.URL,
			RequestBody:    "requestJson",
			RequestHeaders: map[string][]string{"Content-Type": {"application/json"}, "X-Prebid": {"pbs-go/test-version"}},
			ResponseBody:   "responseJson",
			Status:         200,
		},
	}

	assert.Empty(t, errs)
	assert.Len(t, seatBids, 1)
	assert.NotEmpty(t, extraInfo.MakeBidsTimeInfo.Durations)
	assert.False(t, extraInfo.MakeBidsTimeInfo.AfterMakeBidsStartTime.IsZero())
	assert.ElementsMatch(t, seatBids[0].HttpCalls, expectedHttpCalls)
}

func TestSetGPCHeader(t *testing.T) {
	server := httptest.NewServer(mockHandler(200, "getBody", "responseJson"))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("requestJson"),
			Headers: requestHeaders,
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{},
		},
	}

	debugInfo := &config.DebugInfo{Allow: true}
	ctx := context.Background()

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, debugInfo, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: "test",
	}
	bidAdjustments := map[string]float64{"test": 1}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  false,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	extraInfo := &adapters.ExtraRequestInfo{GlobalPrivacyControlHeader: "1"}
	seatBids, errs := bidder.requestBid(ctx, bidderReq, currencyConverter.Rates(), extraInfo, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})

	expectedHttpCall := []*openrtb_ext.ExtHttpCall{
		{
			Uri:            server.URL,
			RequestBody:    "requestJson",
			RequestHeaders: map[string][]string{"Content-Type": {"application/json"}, "X-Prebid": {"pbs-go/unknown"}, "Sec-Gpc": {"1"}},
			ResponseBody:   "responseJson",
			Status:         200,
		},
	}

	assert.Empty(t, errs)
	assert.Len(t, seatBids, 1)
	assert.NotEmpty(t, extraInfo.MakeBidsTimeInfo.Durations)
	assert.False(t, extraInfo.MakeBidsTimeInfo.AfterMakeBidsStartTime.IsZero())
	assert.ElementsMatch(t, seatBids[0].HttpCalls, expectedHttpCall)
}

func TestSetGPCHeaderNil(t *testing.T) {
	server := httptest.NewServer(mockHandler(200, "getBody", "responseJson"))
	defer server.Close()

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("requestJson"),
			Headers: nil,
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{},
		},
	}

	debugInfo := &config.DebugInfo{Allow: true}
	ctx := context.Background()

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, debugInfo, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: "test",
	}
	bidAdjustments := map[string]float64{"test": 1}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  false,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	extraInfo := &adapters.ExtraRequestInfo{GlobalPrivacyControlHeader: "1"}
	seatBids, errs := bidder.requestBid(ctx, bidderReq, currencyConverter.Rates(), extraInfo, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})

	expectedHttpCall := []*openrtb_ext.ExtHttpCall{
		{
			Uri:            server.URL,
			RequestBody:    "requestJson",
			RequestHeaders: map[string][]string{"X-Prebid": {"pbs-go/unknown"}, "Sec-Gpc": {"1"}},
			ResponseBody:   "responseJson",
			Status:         200,
		},
	}

	assert.Empty(t, errs)
	assert.NotEmpty(t, extraInfo.MakeBidsTimeInfo.Durations)
	assert.False(t, extraInfo.MakeBidsTimeInfo.AfterMakeBidsStartTime.IsZero())
	assert.Len(t, seatBids, 1)
	assert.ElementsMatch(t, seatBids[0].HttpCalls, expectedHttpCall)
}

// TestMultiBidder makes sure all the requests get sent, and the responses processed.
// Because this is done in parallel, it should be run under the race detector.
func TestMultiBidder(t *testing.T) {
	respStatus := 200
	getRespBody := "{\"wasPost\":false}"
	postRespBody := "{\"wasPost\":true}"
	server := httptest.NewServer(mockHandler(respStatus, getRespBody, postRespBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	mockBidderResponse := &adapters.BidderResponse{
		Bids: []*adapters.TypedBid{
			{
				Bid:     &openrtb2.Bid{},
				BidType: openrtb_ext.BidTypeBanner,
			},
			{
				Bid:     &openrtb2.Bid{},
				BidType: openrtb_ext.BidTypeVideo,
			},
		},
	}

	bidderImpl := &mixedMultiBidder{
		httpRequests: []*adapters.RequestData{{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
			{
				Method:  "GET",
				Uri:     server.URL,
				Body:    []byte("{\"key\":\"val2\"}"),
				Headers: http.Header{},
			}},
		bidResponse: mockBidderResponse,
	}
	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: "test",
	}
	bidAdjustments := map[string]float64{"test": 1.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  true,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	seatBids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})

	if len(seatBids) != 1 {
		t.Fatalf("SeatBid should exist, because bids exist.")
	}

	if len(errs) != 1+len(bidderImpl.httpRequests) {
		t.Errorf("Expected %d errors. Got %d", 1+len(bidderImpl.httpRequests), len(errs))
	}
	if len(seatBids[0].Bids) != len(bidderImpl.httpResponses)*len(mockBidderResponse.Bids) {
		t.Errorf("Expected %d bids. Got %d", len(bidderImpl.httpResponses)*len(mockBidderResponse.Bids), len(seatBids[0].Bids))
	}

}

// TestBidderTimeout makes sure that things work smoothly if the context expires before the Bidder
// manages to complete its task.
func TestBidderTimeout(t *testing.T) {
	// Fixes #369 (hopefully): Define a context which has already expired
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(-7*time.Hour))
	cancelFunc()
	<-ctx.Done()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.Method == "GET" {
			w.Write([]byte("getBody"))
		} else {
			w.Write([]byte("postBody"))
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	bidder := &bidderAdapter{
		Bidder:     &mixedMultiBidder{},
		BidderName: openrtb_ext.BidderAppnexus,
		Client:     server.Client(),
		me:         &metricsConfig.NilMetricsEngine{},
	}

	callInfo := bidder.doRequest(ctx, &adapters.RequestData{
		Method: "POST",
		Uri:    server.URL,
	}, time.Now())
	if callInfo.err == nil {
		t.Errorf("The bidder should report an error if the context has expired already.")
	}
	if callInfo.response != nil {
		t.Errorf("There should be no response if the request never completed.")
	}
}

// TestInvalidRequest makes sure that bidderAdapter.doRequest returns errors on bad requests.
func TestInvalidRequest(t *testing.T) {
	server := httptest.NewServer(mockHandler(200, "getBody", "postBody"))
	bidder := &bidderAdapter{
		Bidder: &mixedMultiBidder{},
		Client: server.Client(),
	}

	callInfo := bidder.doRequest(context.Background(), &adapters.RequestData{
		Method: "\"", // force http.NewRequest() to fail
	}, time.Now())
	if callInfo.err == nil {
		t.Errorf("bidderAdapter.doRequest should return an error if the request data is malformed.")
	}
}

// TestConnectionClose makes sure that bidderAdapter.doRequest returns errors if the connection closes unexpectedly.
func TestConnectionClose(t *testing.T) {
	var server *httptest.Server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.CloseClientConnections()
	})
	server = httptest.NewServer(handler)

	bidder := &bidderAdapter{
		Bidder:     &mixedMultiBidder{},
		Client:     server.Client(),
		BidderName: openrtb_ext.BidderAppnexus,
		me:         &metricsConfig.NilMetricsEngine{},
	}

	callInfo := bidder.doRequest(context.Background(), &adapters.RequestData{
		Method: "POST",
		Uri:    server.URL,
	}, time.Now())
	if callInfo.err == nil {
		t.Errorf("bidderAdapter.doRequest should return an error if the connection closes unexpectedly.")
	}
}

type bid struct {
	currency       string
	price          float64
	originalBidCur string
}

// TestMultiCurrencies rate converter is set / active.
func TestMultiCurrencies(t *testing.T) {
	// Setup:
	respStatus := 200
	getRespBody := "{\"wasPost\":false}"
	postRespBody := "{\"wasPost\":true}"

	testCases := []struct {
		bids                      []bid
		rates                     currency.Rates
		expectedBids              []bid
		expectedBadCurrencyErrors []error
		description               string
	}{
		{
			bids: []bid{
				{currency: "USD", price: 1.1},
				{currency: "USD", price: 1.2},
				{currency: "USD", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{
				{currency: "USD", price: 1.1, originalBidCur: "USD"},
				{currency: "USD", price: 1.2, originalBidCur: "USD"},
				{currency: "USD", price: 1.3, originalBidCur: "USD"},
			},
			expectedBadCurrencyErrors: []error{},
			description:               "Case 1 - Bidder respond with the same currency (default one) on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "", price: 1.1},
				{currency: "", price: 1.2},
				{currency: "", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{
				{currency: "USD", price: 1.1, originalBidCur: "USD"},
				{currency: "USD", price: 1.2, originalBidCur: "USD"},
				{currency: "USD", price: 1.3, originalBidCur: "USD"},
			},
			expectedBadCurrencyErrors: []error{},
			description:               "Case 2 - Bidder respond with no currency on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "EUR", price: 1.1},
				{currency: "EUR", price: 1.2},
				{currency: "EUR", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{
				{currency: "USD", price: 1.1 * 1.1435678764, originalBidCur: "EUR"},
				{currency: "USD", price: 1.2 * 1.1435678764, originalBidCur: "EUR"},
				{currency: "USD", price: 1.3 * 1.1435678764, originalBidCur: "EUR"},
			},
			expectedBadCurrencyErrors: []error{},
			description:               "Case 3 - Bidder respond with the same non default currency on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "USD", price: 1.1},
				{currency: "EUR", price: 1.2},
				{currency: "GBP", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{
				{currency: "USD", price: 1.1, originalBidCur: "USD"},
				{currency: "USD", price: 1.2 * 1.1435678764, originalBidCur: "EUR"},
				{currency: "USD", price: 1.3 * 1.3050530256, originalBidCur: "GBP"},
			},
			expectedBadCurrencyErrors: []error{},
			description:               "Case 4 - Bidder respond with a mix of currencies on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "", price: 1.1},
				{currency: "EUR", price: 1.2},
				{currency: "GBP", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{
				{currency: "USD", price: 1.1, originalBidCur: "USD"},
				{currency: "USD", price: 1.2 * 1.1435678764, originalBidCur: "EUR"},
				{currency: "USD", price: 1.3 * 1.3050530256, originalBidCur: "GBP"},
			},
			expectedBadCurrencyErrors: []error{},
			description:               "Case 5 - Bidder respond with a mix of currencies and no currency on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "JPY", price: 1.1},
				{currency: "EUR", price: 1.2},
				{currency: "GBP", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{
				{currency: "USD", price: 1.2 * 1.1435678764, originalBidCur: "EUR"},
				{currency: "USD", price: 1.3 * 1.3050530256, originalBidCur: "GBP"},
			},
			expectedBadCurrencyErrors: []error{
				currency.ConversionNotFoundError{FromCur: "JPY", ToCur: "USD"},
			},
			description: "Case 6 - Bidder respond with a mix of currencies and one unknown on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "JPY", price: 1.1},
				{currency: "BZD", price: 1.2},
				{currency: "DKK", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{},
			expectedBadCurrencyErrors: []error{
				currency.ConversionNotFoundError{FromCur: "JPY", ToCur: "USD"},
				currency.ConversionNotFoundError{FromCur: "BZD", ToCur: "USD"},
				currency.ConversionNotFoundError{FromCur: "DKK", ToCur: "USD"},
			},
			description: "Case 7 - Bidder respond with currencies not having any rate on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "AAA", price: 1.1},
				{currency: "BBB", price: 1.2},
				{currency: "CCC", price: 1.3},
			},
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			expectedBids: []bid{},
			expectedBadCurrencyErrors: []error{
				errors.New("currency: tag is not a recognized currency"),
				errors.New("currency: tag is not a recognized currency"),
				errors.New("currency: tag is not a recognized currency"),
			},
			description: "Case 8 - Bidder respond with not existing currencies",
		},
	}

	server := httptest.NewServer(mockHandler(respStatus, getRespBody, postRespBody))
	defer server.Close()

	for _, tc := range testCases {
		mockBidderResponses := make([]*adapters.BidderResponse, len(tc.bids))
		bidderImpl := &goodMultiHTTPCallsBidder{
			bidResponses: mockBidderResponses,
		}
		bidderImpl.httpRequest = make([]*adapters.RequestData, len(tc.bids))

		for i, bid := range tc.bids {
			mockBidderResponses[i] = &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							Price: bid.price,
						},
						BidType: openrtb_ext.BidTypeBanner,
					},
				},
				Currency: bid.currency,
			}

			bidderImpl.httpRequest[i] = &adapters.RequestData{
				Method:  "POST",
				Uri:     server.URL,
				Body:    []byte("{\"key\":\"val\"}"),
				Headers: http.Header{},
			}
		}

		mockedHTTPServer := httptest.NewServer(http.HandlerFunc(
			func(rw http.ResponseWriter, req *http.Request) {
				b, err := json.Marshal(tc.rates)
				if err == nil {
					rw.WriteHeader(http.StatusOK)
					rw.Write(b)
				} else {
					rw.WriteHeader(http.StatusInternalServerError)
				}
			}),
		)

		// Execute:
		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
		currencyConverter := currency.NewRateConverter(
			&http.Client{},
			mockedHTTPServer.URL,
			time.Duration(24)*time.Hour,
		)
		time.Sleep(time.Duration(500) * time.Millisecond)
		currencyConverter.Run()

		bidderReq := BidderRequest{
			BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
			BidderName: openrtb_ext.BidderAppnexus,
		}
		bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 1}
		seatBids, errs := bidder.requestBid(
			context.Background(),
			bidderReq,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			&adscert.NilSigner{},
			bidRequestOptions{
				accountDebugAllowed: true,
				headerDebugAllowed:  true,
				addCallSignHeader:   false,
				bidAdjustments:      bidAdjustments,
			},
			openrtb_ext.ExtAlternateBidderCodes{},
			&hookexecution.EmptyHookExecutor{},
		)
		assert.Len(t, seatBids, 1)
		seatBid := seatBids[0]

		// Verify:
		resultLightBids := make([]bid, len(seatBid.Bids))
		for i, b := range seatBid.Bids {
			resultLightBids[i] = bid{
				price:          b.Bid.Price,
				currency:       seatBid.Currency,
				originalBidCur: b.OriginalBidCur,
			}
		}
		assert.ElementsMatch(t, tc.expectedBids, resultLightBids, tc.description)
		assert.ElementsMatch(t, tc.expectedBadCurrencyErrors, errs, tc.description)
	}
}

// TestMultiCurrencies_RateConverterNotSet no rate converter is set / active.
func TestMultiCurrencies_RateConverterNotSet(t *testing.T) {
	// Setup:
	respStatus := 200
	getRespBody := "{\"wasPost\":false}"
	postRespBody := "{\"wasPost\":true}"

	testCases := []struct {
		bidCurrency               []string
		expectedBidsCount         uint
		expectedBadCurrencyErrors []error
		description               string
	}{
		{
			bidCurrency:               []string{"USD", "USD", "USD"},
			expectedBidsCount:         3,
			expectedBadCurrencyErrors: []error{},
			description:               "Case 1 - Bidder respond with the same currency (default one) on all HTTP responses",
		},
		{
			bidCurrency:       []string{"EUR", "EUR", "EUR"},
			expectedBidsCount: 0,
			expectedBadCurrencyErrors: []error{
				currency.ConversionNotFoundError{FromCur: "EUR", ToCur: "USD"},
				currency.ConversionNotFoundError{FromCur: "EUR", ToCur: "USD"},
				currency.ConversionNotFoundError{FromCur: "EUR", ToCur: "USD"},
			},
			description: "Case 2 - Bidder respond with the same currency (not default one) on all HTTP responses",
		},
		{
			bidCurrency:               []string{"", "", ""},
			expectedBidsCount:         3,
			expectedBadCurrencyErrors: []error{},
			description:               "Case 3 - Bidder responds with currency not set on all HTTP responses",
		},
		{
			bidCurrency:               []string{"", "USD", ""},
			expectedBidsCount:         3,
			expectedBadCurrencyErrors: []error{},
			description:               "Case 4 - Bidder responds with a mix of not set and default currency in HTTP responses",
		},
		{
			bidCurrency:               []string{"USD", "USD", ""},
			expectedBidsCount:         3,
			expectedBadCurrencyErrors: []error{},
			description:               "Case 5 - Bidder responds with a mix of not set and default currency in HTTP responses",
		},
		{
			bidCurrency:               []string{"", "", "USD"},
			expectedBidsCount:         3,
			expectedBadCurrencyErrors: []error{},
			description:               "Case 6 - Bidder responds with a mix of not set and default currency in HTTP responses",
		},
		{
			bidCurrency:       []string{"EUR", "", "USD"},
			expectedBidsCount: 2,
			expectedBadCurrencyErrors: []error{
				currency.ConversionNotFoundError{FromCur: "EUR", ToCur: "USD"},
			},
			description: "Case 7 - Bidder responds with a mix of not set, non default currency and default currency in HTTP responses",
		},
		{
			bidCurrency:       []string{"GBP", "", "USD"},
			expectedBidsCount: 2,
			expectedBadCurrencyErrors: []error{
				currency.ConversionNotFoundError{FromCur: "GBP", ToCur: "USD"},
			},
			description: "Case 8 - Bidder responds with a mix of not set, non default currency and default currency in HTTP responses",
		},
		{
			bidCurrency:       []string{"GBP", "", ""},
			expectedBidsCount: 2,
			expectedBadCurrencyErrors: []error{
				currency.ConversionNotFoundError{FromCur: "GBP", ToCur: "USD"},
			},
			description: "Case 9 - Bidder responds with a mix of not set and empty currencies (default currency) in HTTP responses",
		},
		// Bidder respond with not existing currencies
		{
			bidCurrency:       []string{"AAA", "BBB", "CCC"},
			expectedBidsCount: 0,
			expectedBadCurrencyErrors: []error{
				errors.New("currency: tag is not a recognized currency"),
				errors.New("currency: tag is not a recognized currency"),
				errors.New("currency: tag is not a recognized currency"),
			},
			description: "Case 10 - Bidder respond with not existing currencies",
		},
	}

	server := httptest.NewServer(mockHandler(respStatus, getRespBody, postRespBody))
	defer server.Close()

	for _, tc := range testCases {
		mockBidderResponses := make([]*adapters.BidderResponse, len(tc.bidCurrency))
		bidderImpl := &goodMultiHTTPCallsBidder{
			bidResponses: mockBidderResponses,
		}
		bidderImpl.httpRequest = make([]*adapters.RequestData, len(tc.bidCurrency))

		for i, cur := range tc.bidCurrency {
			mockBidderResponses[i] = &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid:     &openrtb2.Bid{},
						BidType: openrtb_ext.BidTypeBanner,
					},
				},
				Currency: cur,
			}

			bidderImpl.httpRequest[i] = &adapters.RequestData{
				Method:  "POST",
				Uri:     server.URL,
				Body:    []byte("{\"key\":\"val\"}"),
				Headers: http.Header{},
			}
		}

		// Execute:
		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
		bidderReq := BidderRequest{
			BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
			BidderName: "test",
		}
		bidAdjustments := map[string]float64{"test": 1}
		seatBids, errs := bidder.requestBid(
			context.Background(),
			bidderReq,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			&adscert.NilSigner{},
			bidRequestOptions{
				accountDebugAllowed: true,
				headerDebugAllowed:  true,
				addCallSignHeader:   false,
				bidAdjustments:      bidAdjustments,
			},
			openrtb_ext.ExtAlternateBidderCodes{},
			&hookexecution.EmptyHookExecutor{},
		)
		assert.Len(t, seatBids, 1)
		seatBid := seatBids[0]

		// Verify:
		assert.Equal(t, false, (seatBid == nil && tc.expectedBidsCount != 0), tc.description)
		assert.Equal(t, tc.expectedBidsCount, uint(len(seatBid.Bids)), tc.description)
		assert.ElementsMatch(t, tc.expectedBadCurrencyErrors, errs, tc.description)
	}
}

// TestMultiCurrencies_RequestCurrencyPick tests request currencies pick.
func TestMultiCurrencies_RequestCurrencyPick(t *testing.T) {
	// Setup:
	respStatus := 200
	getRespBody := "{\"wasPost\":false}"
	postRespBody := "{\"wasPost\":true}"

	testCases := []struct {
		bidRequestCurrencies   []string
		bidResponsesCurrency   string
		expectedPickedCurrency string
		expectedError          bool
		rates                  currency.Rates
		description            string
	}{
		{
			bidRequestCurrencies:   []string{"EUR", "USD", "JPY"},
			bidResponsesCurrency:   "EUR",
			expectedPickedCurrency: "EUR",
			expectedError:          false,
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"JPY": {
						"USD": 0.0089,
					},
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			description: "Case 1 - Allowed currencies in bid request are known, first one is picked",
		},
		{
			bidRequestCurrencies:   []string{"JPY"},
			bidResponsesCurrency:   "JPY",
			expectedPickedCurrency: "JPY",
			expectedError:          false,
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"JPY": {
						"USD": 0.0089,
					},
				},
			},
			description: "Case 2 - There is only one allowed currencies in bid request, it's a known one, it's picked",
		},
		{
			bidRequestCurrencies:   []string{"CNY", "USD", "EUR", "JPY"},
			bidResponsesCurrency:   "USD",
			expectedPickedCurrency: "USD",
			expectedError:          false,
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{
					"JPY": {
						"USD": 0.0089,
					},
					"GBP": {
						"USD": 1.3050530256,
					},
					"EUR": {
						"USD": 1.1435678764,
					},
				},
			},
			description: "Case 3 - First allowed currencies in bid request is not known but the others are, second one is picked",
		},
		{
			bidRequestCurrencies:   []string{"CNY", "EUR", "JPY"},
			bidResponsesCurrency:   "",
			expectedPickedCurrency: "",
			expectedError:          true,
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{},
			},
			description: "Case 4 - None allowed currencies in bid request are known, an error is returned",
		},
		{
			bidRequestCurrencies:   []string{"CNY", "EUR", "JPY", "USD"},
			bidResponsesCurrency:   "USD",
			expectedPickedCurrency: "USD",
			expectedError:          false,
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{},
			},
			description: "Case 5 - None allowed currencies in bid request are known but the default one (`USD`), no rates are set but default currency will be picked",
		},
		{
			bidRequestCurrencies:   nil,
			bidResponsesCurrency:   "USD",
			expectedPickedCurrency: "USD",
			expectedError:          false,
			rates: currency.Rates{
				Conversions: map[string]map[string]float64{},
			},
			description: "Case 6 - No allowed currencies specified in bid request, default one is picked: `USD`",
		},
	}

	server := httptest.NewServer(mockHandler(respStatus, getRespBody, postRespBody))
	defer server.Close()

	for _, tc := range testCases {

		mockedHTTPServer := httptest.NewServer(http.HandlerFunc(
			func(rw http.ResponseWriter, req *http.Request) {
				b, err := json.Marshal(tc.rates)
				if err == nil {
					rw.WriteHeader(http.StatusOK)
					rw.Write(b)
				} else {
					rw.WriteHeader(http.StatusInternalServerError)
				}
			}),
		)

		mockBidderResponses := []*adapters.BidderResponse{
			{
				Bids: []*adapters.TypedBid{
					{
						Bid:     &openrtb2.Bid{},
						BidType: openrtb_ext.BidTypeBanner,
					},
				},
				Currency: tc.bidResponsesCurrency,
			},
		}
		bidderImpl := &goodMultiHTTPCallsBidder{
			bidResponses: mockBidderResponses,
		}
		bidderImpl.httpRequest = []*adapters.RequestData{
			{
				Method:  "POST",
				Uri:     server.URL,
				Body:    []byte("{\"key\":\"val\"}"),
				Headers: http.Header{},
			},
		}

		// Execute:
		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
		currencyConverter := currency.NewRateConverter(
			&http.Client{},
			mockedHTTPServer.URL,
			time.Duration(24)*time.Hour,
		)
		bidderReq := BidderRequest{
			BidRequest: &openrtb2.BidRequest{Cur: tc.bidRequestCurrencies, Imp: []openrtb2.Imp{{ID: "impId"}}},
			BidderName: "test",
		}
		bidAdjustments := map[string]float64{"test": 1}
		seatBids, errs := bidder.requestBid(
			context.Background(),
			bidderReq,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			&adscert.NilSigner{},
			bidRequestOptions{
				accountDebugAllowed: true,
				headerDebugAllowed:  false,
				addCallSignHeader:   false,
				bidAdjustments:      bidAdjustments,
			},
			openrtb_ext.ExtAlternateBidderCodes{},
			&hookexecution.EmptyHookExecutor{},
		)
		assert.Len(t, seatBids, 1)
		seatBid := seatBids[0]

		// Verify:
		if tc.expectedError {
			assert.NotNil(t, errs, tc.description)
		} else {
			assert.Nil(t, errs, tc.description)
			assert.Equal(t, tc.expectedPickedCurrency, seatBid.Currency, tc.description)
		}
	}
}

func TestMakeExt(t *testing.T) {
	testCases := []struct {
		description string
		given       *httpCallInfo
		expected    *openrtb_ext.ExtHttpCall
	}{
		{
			description: "Nil",
			given:       nil,
			expected:    &openrtb_ext.ExtHttpCall{},
		},
		{
			description: "Empty",
			given: &httpCallInfo{
				err:      nil,
				response: nil,
				request:  nil,
			},
			expected: &openrtb_ext.ExtHttpCall{},
		},
		{
			description: "Request & Response - No Error",
			given: &httpCallInfo{
				err: nil,
				request: &adapters.RequestData{
					Uri:     "requestUri",
					Body:    []byte("requestBody"),
					Headers: makeHeader(map[string][]string{"Key1": {"value1", "value2"}}),
				},
				response: &adapters.ResponseData{
					Body:       []byte("responseBody"),
					StatusCode: 999,
				},
			},
			expected: &openrtb_ext.ExtHttpCall{
				Uri:            "requestUri",
				RequestBody:    "requestBody",
				RequestHeaders: map[string][]string{"Key1": {"value1", "value2"}},
				ResponseBody:   "responseBody",
				Status:         999,
			},
		},
		{
			description: "Request & Response - No Error with Authorization removal",
			given: &httpCallInfo{
				err: nil,
				request: &adapters.RequestData{
					Uri:     "requestUri",
					Body:    []byte("requestBody"),
					Headers: makeHeader(map[string][]string{"Key1": {"value1", "value2"}, "Authorization": {"secret"}}),
				},
				response: &adapters.ResponseData{
					Body:       []byte("responseBody"),
					StatusCode: 999,
				},
			},
			expected: &openrtb_ext.ExtHttpCall{
				Uri:            "requestUri",
				RequestBody:    "requestBody",
				RequestHeaders: map[string][]string{"Key1": {"value1", "value2"}},
				ResponseBody:   "responseBody",
				Status:         999,
			},
		},
		{
			description: "Request & Response - No Error with nil header",
			given: &httpCallInfo{
				err: nil,
				request: &adapters.RequestData{
					Uri:     "requestUri",
					Body:    []byte("requestBody"),
					Headers: nil,
				},
				response: &adapters.ResponseData{
					Body:       []byte("responseBody"),
					StatusCode: 999,
				},
			},
			expected: &openrtb_ext.ExtHttpCall{
				Uri:            "requestUri",
				RequestBody:    "requestBody",
				RequestHeaders: nil,
				ResponseBody:   "responseBody",
				Status:         999,
			},
		},
		{
			description: "Request & Response - Error",
			given: &httpCallInfo{
				err: errors.New("error"),
				request: &adapters.RequestData{
					Uri:     "requestUri",
					Body:    []byte("requestBody"),
					Headers: makeHeader(map[string][]string{"Key1": {"value1", "value2"}}),
				},
				response: &adapters.ResponseData{
					Body:       []byte("responseBody"),
					StatusCode: 999,
				},
			},
			expected: &openrtb_ext.ExtHttpCall{
				Uri:            "requestUri",
				RequestBody:    "requestBody",
				RequestHeaders: map[string][]string{"Key1": {"value1", "value2"}},
			},
		},
		{
			description: "Request Only",
			given: &httpCallInfo{
				err: nil,
				request: &adapters.RequestData{
					Uri:     "requestUri",
					Body:    []byte("requestBody"),
					Headers: makeHeader(map[string][]string{"Key1": {"value1", "value2"}}),
				},
				response: nil,
			},
			expected: &openrtb_ext.ExtHttpCall{
				Uri:            "requestUri",
				RequestBody:    "requestBody",
				RequestHeaders: map[string][]string{"Key1": {"value1", "value2"}},
			},
		}, {
			description: "Response Only",
			given: &httpCallInfo{
				err: nil,
				response: &adapters.ResponseData{
					Body:       []byte("responseBody"),
					StatusCode: 999,
				},
			},
			expected: &openrtb_ext.ExtHttpCall{},
		},
	}

	for _, test := range testCases {
		result := makeExt(test.given)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestFilterHeader(t *testing.T) {
	testCases := []struct {
		description string
		given       http.Header
		expected    http.Header
	}{
		{
			description: "Nil",
			given:       nil,
			expected:    nil,
		},
		{
			description: "Empty",
			given:       http.Header{},
			expected:    http.Header{},
		},
		{
			description: "One",
			given:       makeHeader(map[string][]string{"Key1": {"value1"}}),
			expected:    makeHeader(map[string][]string{"Key1": {"value1"}}),
		},
		{
			description: "Many",
			given:       makeHeader(map[string][]string{"Key1": {"value1"}, "Key2": {"value2a", "value2b"}}),
			expected:    makeHeader(map[string][]string{"Key1": {"value1"}, "Key2": {"value2a", "value2b"}}),
		},
		{
			description: "Authorization Header Omitted",
			given:       makeHeader(map[string][]string{"authorization": {"secret"}}),
			expected:    http.Header{},
		},
		{
			description: "Authorization Header Omitted - Case Insensitive",
			given:       makeHeader(map[string][]string{"AuThOrIzAtIoN": {"secret"}}),
			expected:    http.Header{},
		},
		{
			description: "Authorization Header Omitted + Other Keys",
			given:       makeHeader(map[string][]string{"authorization": {"secret"}, "Key1": {"value1"}}),
			expected:    makeHeader(map[string][]string{"Key1": {"value1"}}),
		},
	}

	for _, test := range testCases {
		result := filterHeader(test.given)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func makeHeader(v map[string][]string) http.Header {
	h := http.Header{}
	for key, values := range v {
		for _, value := range values {
			h.Add(key, value)
		}
	}
	return h
}

func TestMobileNativeTypes(t *testing.T) {
	respBody := "{\"bid\":false}"
	respStatus := 200
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	reqBody := "{\"key\":\"val\"}"
	reqURL := server.URL

	testCases := []struct {
		mockBidderRequest  *openrtb2.BidRequest
		mockBidderResponse *adapters.BidderResponse
		expectedValue      string
		description        string
	}{
		{
			mockBidderRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "some-imp-id",
						Native: &openrtb2.Native{
							Request: "{\"ver\":\"1.1\",\"context\":1,\"contextsubtype\":11,\"plcmttype\":4,\"plcmtcnt\":1,\"assets\":[{\"id\":1,\"required\":1,\"title\":{\"len\":500}},{\"id\":2,\"required\":1,\"img\":{\"type\":3,\"wmin\":1,\"hmin\":1}},{\"id\":3,\"required\":0,\"data\":{\"type\":1,\"len\":200}},{\"id\":4,\"required\":0,\"data\":{\"type\":2,\"len\":15000}},{\"id\":5,\"required\":0,\"data\":{\"type\":6,\"len\":40}}]}",
						},
					},
				},
				App: &openrtb2.App{},
			},
			mockBidderResponse: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							ImpID: "some-imp-id",
							AdM:   "{\"assets\":[{\"id\":2,\"img\":{\"url\":\"http://vcdn.adnxs.com/p/creative-image/f8/7f/0f/13/f87f0f13-230c-4f05-8087-db9216e393de.jpg\",\"w\":989,\"h\":742,\"ext\":{\"appnexus\":{\"prevent_crop\":0}}}},{\"id\":1,\"title\":{\"text\":\"This is a Prebid Native Creative\"}},{\"id\":3,\"data\":{\"value\":\"Prebid.org\"}},{\"id\":4,\"data\":{\"value\":\"This is a Prebid Native Creative.  There are many like it, but this one is mine.\"}}],\"link\":{\"url\":\"http://some-url.com\"},\"imptrackers\":[\"http://someimptracker.com\"],\"jstracker\":\"some-js-tracker\"}",
							Price: 10,
						},
						BidType: openrtb_ext.BidTypeNative,
					},
				},
			},
			expectedValue: "{\"assets\":[{\"id\":2,\"img\":{\"type\":3,\"url\":\"http://vcdn.adnxs.com/p/creative-image/f8/7f/0f/13/f87f0f13-230c-4f05-8087-db9216e393de.jpg\",\"w\":989,\"h\":742,\"ext\":{\"appnexus\":{\"prevent_crop\":0}}}},{\"id\":1,\"title\":{\"text\":\"This is a Prebid Native Creative\"}},{\"id\":3,\"data\":{\"type\":1,\"value\":\"Prebid.org\"}},{\"id\":4,\"data\":{\"type\":2,\"value\":\"This is a Prebid Native Creative.  There are many like it, but this one is mine.\"}}],\"link\":{\"url\":\"http://some-url.com\"},\"imptrackers\":[\"http://someimptracker.com\"],\"jstracker\":\"some-js-tracker\"}",
			description:   "Checks types in response",
		},
		{
			mockBidderRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "some-imp-id",
						Native: &openrtb2.Native{
							Request: "{\"ver\":\"1.1\",\"context\":1,\"contextsubtype\":11,\"plcmttype\":4,\"plcmtcnt\":1,\"assets\":[{\"id\":1,\"required\":1,\"title\":{\"len\":500}},{\"id\":2,\"required\":1,\"img\":{\"type\":3,\"wmin\":1,\"hmin\":1}},{\"id\":3,\"required\":0,\"data\":{\"type\":1,\"len\":200}},{\"id\":4,\"required\":0,\"data\":{\"type\":2,\"len\":15000}},{\"id\":5,\"required\":0,\"data\":{\"type\":6,\"len\":40}}]}",
						},
					},
				},
				App: &openrtb2.App{},
			},
			mockBidderResponse: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							ImpID: "some-imp-id",
							AdM:   "{\"some-diff-markup\":\"creative\"}",
							Price: 10,
						},
						BidType: openrtb_ext.BidTypeNative,
					},
				},
			},
			expectedValue: "{\"some-diff-markup\":\"creative\"}",
			description:   "Non IAB compliant markup",
		},
	}

	for _, tc := range testCases {
		bidderImpl := &goodSingleBidder{
			httpRequest: &adapters.RequestData{
				Method:  "POST",
				Uri:     reqURL,
				Body:    []byte(reqBody),
				Headers: http.Header{},
			},
			bidResponse: tc.mockBidderResponse,
		}
		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		bidderReq := BidderRequest{
			BidRequest: tc.mockBidderRequest,
			BidderName: "test",
		}
		bidAdjustments := map[string]float64{"test": 1.0}
		seatBids, _ := bidder.requestBid(
			context.Background(),
			bidderReq,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			&adscert.NilSigner{},
			bidRequestOptions{
				accountDebugAllowed: true,
				headerDebugAllowed:  true,
				addCallSignHeader:   false,
				bidAdjustments:      bidAdjustments,
			},
			openrtb_ext.ExtAlternateBidderCodes{},
			&hookexecution.EmptyHookExecutor{},
		)
		assert.Len(t, seatBids, 1)

		var actualValue string
		for _, bid := range seatBids[0].Bids {
			actualValue = bid.Bid.AdM
			assert.JSONEq(t, tc.expectedValue, actualValue, tc.description)
		}
	}
}

func TestRequestBidsStoredBidResponses(t *testing.T) {
	respBody := "{\"bid\":false}"
	respStatus := 200
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	bidRespId1 := json.RawMessage(`{"id": "resp_id1", "seatbid": [{"bid": [{"id": "bid_id1"}], "seat": "testBidder1"}], "bidid": "123", "cur": "USD"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2", "seatbid": [{"bid": [{"id": "bid_id2_1", "impid": "bid1impid1"},{"id": "bid_id2_2", "impid": "bid2impid2"}], "seat": "testBidder2"}], "bidid": "124", "cur": "USD"}`)

	testCases := []struct {
		description           string
		mockBidderRequest     *openrtb2.BidRequest
		bidderStoredResponses map[string]json.RawMessage
		impReplaceImpId       map[string]bool
		expectedBidIds        []string
		expectedImpIds        []string
	}{
		{
			description: "Single imp with stored bid response, replace impid is true",
			mockBidderRequest: &openrtb2.BidRequest{
				Imp: nil,
				App: &openrtb2.App{},
			},
			bidderStoredResponses: map[string]json.RawMessage{
				"bidResponseId1": bidRespId1,
			},
			impReplaceImpId: map[string]bool{
				"bidResponseId1": true,
			},
			expectedBidIds: []string{"bid_id1"},
			expectedImpIds: []string{"bidResponseId1"},
		},
		{
			description: "Single imp with multiple stored bid responses, replace impid is true",
			mockBidderRequest: &openrtb2.BidRequest{
				Imp: nil,
				App: &openrtb2.App{},
			},
			bidderStoredResponses: map[string]json.RawMessage{
				"bidResponseId2": bidRespId2,
			},
			impReplaceImpId: map[string]bool{
				"bidResponseId2": true,
			},
			expectedBidIds: []string{"bid_id2_1", "bid_id2_2"},
			expectedImpIds: []string{"bidResponseId2", "bidResponseId2"},
		},
		{
			description: "Single imp with multiple stored bid responses, replace impid is false",
			mockBidderRequest: &openrtb2.BidRequest{
				Imp: nil,
				App: &openrtb2.App{},
			},
			bidderStoredResponses: map[string]json.RawMessage{
				"bidResponseId2": bidRespId2,
			},
			impReplaceImpId: map[string]bool{
				"bidResponseId2": false,
			},
			expectedBidIds: []string{"bid_id2_1", "bid_id2_2"},
			expectedImpIds: []string{"bid1impid1", "bid2impid2"},
		},
		{
			description: "Two imp with multiple stored bid responses, replace impid is true and false",
			mockBidderRequest: &openrtb2.BidRequest{
				Imp: nil,
				App: &openrtb2.App{},
			},
			bidderStoredResponses: map[string]json.RawMessage{
				"bidResponseId1": bidRespId1,
				"bidResponseId2": bidRespId2,
			},
			impReplaceImpId: map[string]bool{
				"bidResponseId1": true,
				"bidResponseId2": false,
			},
			expectedBidIds: []string{"bid_id2_1", "bid_id2_2", "bid_id1"},
			expectedImpIds: []string{"bid1impid1", "bid2impid2", "bidResponseId1"},
		},
	}

	for _, tc := range testCases {

		bidderImpl := &goodSingleBidderWithStoredBidResp{}
		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		bidderReq := BidderRequest{
			BidRequest:            tc.mockBidderRequest,
			BidderName:            openrtb_ext.BidderAppnexus,
			BidderStoredResponses: tc.bidderStoredResponses,
			ImpReplaceImpId:       tc.impReplaceImpId,
		}
		bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 1.0}
		seatBids, _ := bidder.requestBid(
			context.Background(),
			bidderReq,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			&adscert.NilSigner{},
			bidRequestOptions{
				accountDebugAllowed: true,
				headerDebugAllowed:  true,
				addCallSignHeader:   false,
				bidAdjustments:      bidAdjustments,
			},
			openrtb_ext.ExtAlternateBidderCodes{},
			&hookexecution.EmptyHookExecutor{},
		)
		assert.Len(t, seatBids, 1)

		assert.Len(t, seatBids[0].Bids, len(tc.expectedBidIds), "Incorrect bids number for test case ", tc.description)
		for _, bid := range seatBids[0].Bids {
			assert.Contains(t, tc.expectedBidIds, bid.Bid.ID, tc.description)
			assert.Contains(t, tc.expectedImpIds, bid.Bid.ImpID, tc.description)
		}
	}

}

// TestFledge verifies that fledge responses from bidders are collected.
func TestFledge(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	fledgeAuctionConfig1 := &openrtb_ext.FledgeAuctionConfig{
		ImpId:  "imp-id-1",
		Config: json.RawMessage("[1,2,3]"),
		Bidder: "openx",
	}
	fledgeAuctionConfig2 := &openrtb_ext.FledgeAuctionConfig{
		ImpId:  "imp-id-2",
		Config: json.RawMessage("[3,2,1]"),
		Bidder: "openx",
	}

	testCases := []struct {
		mockBidderResponse []*adapters.BidderResponse
		expectedFledge     []*openrtb_ext.FledgeAuctionConfig
		description        string
	}{
		{
			mockBidderResponse: []*adapters.BidderResponse{
				{
					Bids:                 []*adapters.TypedBid{},
					FledgeAuctionConfigs: []*openrtb_ext.FledgeAuctionConfig{fledgeAuctionConfig1, fledgeAuctionConfig2},
				},
				nil,
			},
			expectedFledge: []*openrtb_ext.FledgeAuctionConfig{fledgeAuctionConfig1, fledgeAuctionConfig2},
			description:    "Collects FLEDGE auction configs from single bidder response",
		},
		{
			mockBidderResponse: []*adapters.BidderResponse{
				{
					Bids:                 []*adapters.TypedBid{},
					FledgeAuctionConfigs: []*openrtb_ext.FledgeAuctionConfig{fledgeAuctionConfig2},
				},
				{
					Bids:                 []*adapters.TypedBid{},
					FledgeAuctionConfigs: []*openrtb_ext.FledgeAuctionConfig{fledgeAuctionConfig1},
				},
			},
			expectedFledge: []*openrtb_ext.FledgeAuctionConfig{fledgeAuctionConfig1, fledgeAuctionConfig2},
			description:    "Collects FLEDGE auction configs from multiple bidder response",
		},
	}

	for _, tc := range testCases {
		bidderImpl := &goodMultiHTTPCallsBidder{
			httpRequest: []*adapters.RequestData{
				{
					Method:  "POST",
					Uri:     server.URL,
					Body:    []byte("{\"key\":\"val1\"}"),
					Headers: http.Header{},
				},
				{
					Method:  "POST",
					Uri:     server.URL,
					Body:    []byte("{\"key\":\"val2\"}"),
					Headers: http.Header{},
				},
			},
			bidResponses: tc.mockBidderResponse,
		}
		bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderOpenx, nil, "")
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		bidderReq := BidderRequest{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-id-1",
						Banner: &openrtb2.Banner{},
						Ext:    json.RawMessage(`{"ae": 1}`),
					},
					{
						ID:     "imp-id-2",
						Banner: &openrtb2.Banner{},
						Ext:    json.RawMessage(`{"ae": 1}`),
					},
				},
			},
			BidderName: "openx",
		}
		seatBids, _ := bidder.requestBid(
			context.Background(),
			bidderReq,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			&adscert.NilSigner{},
			bidRequestOptions{
				accountDebugAllowed: true,
				headerDebugAllowed:  true,
				addCallSignHeader:   false,
				bidAdjustments:      map[string]float64{"test": 1.0},
			},
			openrtb_ext.ExtAlternateBidderCodes{},
			&hookexecution.EmptyHookExecutor{},
		)
		assert.Len(t, seatBids, 1)
		assert.NotNil(t, seatBids[0].FledgeAuctionConfigs)
		assert.Len(t, seatBids[0].FledgeAuctionConfigs, len(tc.expectedFledge))

		assert.ElementsMatch(t, seatBids[0].FledgeAuctionConfigs, tc.expectedFledge)
	}
}

func TestErrorReporting(t *testing.T) {
	bidder := AdaptBidder(&bidRejector{}, nil, &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: "test",
	}
	bidAdjustments := map[string]float64{"test": 1.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  false,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	bids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})
	if bids != nil {
		t.Errorf("There should be no seatbid if no http requests are returned.")
	}
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error. got %d", len(errs))
	}
	if errs[0].Error() != "Invalid params on BidRequest." {
		t.Errorf(`Error message was mutated. Expected "%s", Got "%s"`, "Invalid params on BidRequest.", errs[0].Error())
	}
}

func TestSetAssetTypes(t *testing.T) {
	testCases := []struct {
		respAsset   nativeResponse.Asset
		nativeReq   nativeRequests.Request
		expectedErr string
		desc        string
	}{
		{
			respAsset: nativeResponse.Asset{
				ID: openrtb2.Int64Ptr(1),
				Img: &nativeResponse.Image{
					URL: "http://some-url",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 1,
						Img: &nativeRequests.Image{
							Type: 2,
						},
					},
					{
						ID: 2,
						Data: &nativeRequests.Data{
							Type: 4,
						},
					},
				},
			},
			expectedErr: "",
			desc:        "Matching image asset exists in the request and asset type is set correctly",
		},
		{
			respAsset: nativeResponse.Asset{
				ID: openrtb2.Int64Ptr(2),
				Data: &nativeResponse.Data{
					Label: "some label",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 1,
						Img: &nativeRequests.Image{
							Type: 2,
						},
					},
					{
						ID: 2,
						Data: &nativeRequests.Data{
							Type: 4,
						},
					},
				},
			},
			expectedErr: "",
			desc:        "Matching data asset exists in the request and asset type is set correctly",
		},
		{
			respAsset: nativeResponse.Asset{
				ID: openrtb2.Int64Ptr(1),
				Img: &nativeResponse.Image{
					URL: "http://some-url",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 2,
						Img: &nativeRequests.Image{
							Type: 2,
						},
					},
				},
			},
			expectedErr: "Unable to find asset with ID:1 in the request",
			desc:        "Matching image asset with the same ID doesn't exist in the request",
		},
		{
			respAsset: nativeResponse.Asset{
				ID: openrtb2.Int64Ptr(2),
				Data: &nativeResponse.Data{
					Label: "some label",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 2,
						Img: &nativeRequests.Image{
							Type: 2,
						},
					},
				},
			},
			expectedErr: "Response has a Data asset with ID:2 present that doesn't exist in the request",
			desc:        "Assets with same ID in the req and resp are of different types",
		},
		{
			respAsset: nativeResponse.Asset{
				ID: openrtb2.Int64Ptr(1),
				Img: &nativeResponse.Image{
					URL: "http://some-url",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 1,
						Data: &nativeRequests.Data{
							Type: 2,
						},
					},
				},
			},
			expectedErr: "Response has an Image asset with ID:1 present that doesn't exist in the request",
			desc:        "Assets with same ID in the req and resp are of different types",
		},
		{
			respAsset: nativeResponse.Asset{
				Img: &nativeResponse.Image{
					URL: "http://some-url",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 1,
						Img: &nativeRequests.Image{
							Type: 2,
						},
					},
				},
			},
			expectedErr: "Response Image asset doesn't have an ID",
			desc:        "Response Image without an ID",
		},
		{
			respAsset: nativeResponse.Asset{
				Data: &nativeResponse.Data{
					Label: "some label",
				},
			},
			nativeReq: nativeRequests.Request{
				Assets: []nativeRequests.Asset{
					{
						ID: 1,
						Data: &nativeRequests.Data{
							Type: 2,
						},
					},
				},
			},
			expectedErr: "Response Data asset doesn't have an ID",
			desc:        "Response Data asset without an ID",
		},
	}

	for _, test := range testCases {
		err := setAssetTypes(test.respAsset, test.nativeReq)
		if len(test.expectedErr) != 0 {
			assert.EqualError(t, err, test.expectedErr, "Test Case: %s", test.desc)
			continue
		} else {
			assert.NoError(t, err, "Test Case: %s", test.desc)
		}

		for _, asset := range test.nativeReq.Assets {
			if asset.Img != nil && test.respAsset.Img != nil {
				assert.Equal(t, asset.Img.Type, test.respAsset.Img.Type, "Asset type not set correctly. Test Case: %s", test.desc)
			}
			if asset.Data != nil && test.respAsset.Data != nil {
				assert.Equal(t, asset.Data.Type, test.respAsset.Data.Type, "Asset type not set correctly. Test Case: %s", test.desc)
			}
		}
	}
}

func TestCallRecordAdapterConnections(t *testing.T) {
	// Setup mock server
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	// declare requestBid parameters
	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 2.0}

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{},
	}

	// setup a mock mockMetricEngine engine and its expectation
	mockMetricEngine := &metrics.MetricsEngineMock{}
	expectedAdapterName := openrtb_ext.BidderAppnexus
	compareConnWaitTime := func(dur time.Duration) bool { return dur.Nanoseconds() > 0 }

	mockMetricEngine.On("RecordAdapterConnections", expectedAdapterName, false, mock.MatchedBy(compareConnWaitTime)).Once()
	mockMetricEngine.On("RecordOverheadTime", metrics.PreBidder, mock.Anything).Once()

	// Run requestBid using an http.Client with a mock handler
	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, mockMetricEngine, openrtb_ext.BidderAppnexus, nil, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: openrtb_ext.BidderAppnexus,
	}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  true,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	_, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{PbsEntryPoint: metrics.ReqTypeORTB2Web}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})

	// Assert no errors
	assert.Equal(t, 0, len(errs), "bidder.requestBid returned errors %v \n", errs)

	// Assert RecordAdapterConnections() was called with the parameters we expected
	mockMetricEngine.AssertExpectations(t)
}

type DNSDoneTripper struct{}

func (DNSDoneTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Access the httptrace.ClientTrace
	trace := httptrace.ContextClientTrace(req.Context())
	// Call the DNSDone method on the client trace
	trace.DNSDone(httptrace.DNSDoneInfo{})

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("postBody")),
	}

	return resp, nil
}

type TLSHandshakeTripper struct{}

func (TLSHandshakeTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Access the httptrace.ClientTrace
	trace := httptrace.ContextClientTrace(req.Context())
	// Call the TLSHandshakeDone method on the client trace
	trace.TLSHandshakeDone(tls.ConnectionState{}, nil)

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("postBody")),
	}

	return resp, nil
}

func TestCallRecordDNSTime(t *testing.T) {
	// setup a mock metrics engine and its expectation
	metricsMock := &metrics.MetricsEngineMock{}
	metricsMock.Mock.On("RecordDNSTime", mock.Anything).Return()
	metricsMock.On("RecordOverheadTime", metrics.PreBidder, mock.Anything).Once()

	// Instantiate the bidder that will send the request. We'll make sure to use an
	// http.Client that runs our mock RoundTripper so DNSDone(httptrace.DNSDoneInfo{})
	// gets called
	bidder := &bidderAdapter{
		Bidder: &mixedMultiBidder{},
		Client: &http.Client{Transport: DNSDoneTripper{}},
		me:     metricsMock,
	}

	// Run test
	bidder.doRequest(context.Background(), &adapters.RequestData{Method: "POST", Uri: "http://www.example.com/"}, time.Now())

	// Tried one or another, none seem to work without panicking
	metricsMock.AssertExpectations(t)
}

func TestCallRecordTLSHandshakeTime(t *testing.T) {
	// setup a mock metrics engine and its expectation
	metricsMock := &metrics.MetricsEngineMock{}
	metricsMock.Mock.On("RecordTLSHandshakeTime", mock.Anything).Return()
	metricsMock.On("RecordOverheadTime", metrics.PreBidder, mock.Anything).Once()

	// Instantiate the bidder that will send the request. We'll make sure to use an
	// http.Client that runs our mock RoundTripper so DNSDone(httptrace.DNSDoneInfo{})
	// gets called
	bidder := &bidderAdapter{
		Bidder: &mixedMultiBidder{},
		Client: &http.Client{Transport: TLSHandshakeTripper{}},
		me:     metricsMock,
	}

	// Run test
	bidder.doRequest(context.Background(), &adapters.RequestData{Method: "POST", Uri: "http://www.example.com/"}, time.Now())

	// Tried one or another, none seem to work without panicking
	metricsMock.AssertExpectations(t)
}

func TestTimeoutNotificationOff(t *testing.T) {
	respBody := "{\"bid\":false}"
	respStatus := 200
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	bidderImpl := &notifyingBidder{
		notifyRequest: adapters.RequestData{
			Method:  "GET",
			Uri:     server.URL + "/notify/me",
			Body:    nil,
			Headers: http.Header{},
		},
	}
	bidder := &bidderAdapter{
		Bidder: bidderImpl,
		Client: server.Client(),
		config: bidderAdapterConfig{Debug: config.Debug{}},
		me:     &metricsConfig.NilMetricsEngine{},
	}
	if tb, ok := bidder.Bidder.(adapters.TimeoutBidder); !ok {
		t.Error("Failed to cast bidder to a TimeoutBidder")
	} else {
		bidder.doTimeoutNotification(tb, &adapters.RequestData{}, glog.Warningf)
	}
}

func TestTimeoutNotificationOn(t *testing.T) {
	// Expire context immediately to force timeout handler.
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now())
	cancelFunc()

	// Notification logic is hardcoded for 200ms. We need to wait for a little longer than that.
	server := httptest.NewServer(mockSlowHandler(205*time.Millisecond, 200, `{"bid":false}`))
	defer server.Close()

	bidder := &notifyingBidder{
		notifyRequest: adapters.RequestData{
			Method:  "GET",
			Uri:     server.URL + "/notify/me",
			Body:    nil,
			Headers: http.Header{},
		},
	}

	// Wrap with BidderInfo to mimic exchange.go flow.
	bidderWrappedWithInfo := wrapWithBidderInfo(bidder)

	bidderAdapter := &bidderAdapter{
		Bidder: bidderWrappedWithInfo,
		Client: server.Client(),
		config: bidderAdapterConfig{
			Debug: config.Debug{
				TimeoutNotification: config.TimeoutNotification{
					Log:          true,
					SamplingRate: 1.0,
				},
			},
		},
		me: &metricsConfig.NilMetricsEngine{},
	}

	// Unwrap To Mimic exchange.go Casting Code
	var coreBidder adapters.Bidder = bidderAdapter.Bidder
	if b, ok := coreBidder.(*adapters.InfoAwareBidder); ok {
		coreBidder = b.Bidder
	}
	if _, ok := coreBidder.(adapters.TimeoutBidder); !ok {
		t.Fatal("Failed to cast bidder to a TimeoutBidder")
	}

	bidRequest := adapters.RequestData{
		Method: "POST",
		Uri:    server.URL,
		Body:   []byte(`{"id":"this-id","app":{"publisher":{"id":"pub-id"}}}`),
	}

	var loggerBuffer bytes.Buffer
	logger := func(msg string, args ...interface{}) {
		loggerBuffer.WriteString(fmt.Sprintf(fmt.Sprintln(msg), args...))
	}

	bidderAdapter.doRequestImpl(ctx, &bidRequest, logger, time.Now())

	// Wait a little longer than the 205ms mock server sleep.
	time.Sleep(210 * time.Millisecond)

	logExpected := "TimeoutNotification: error:(context deadline exceeded) body:\n"
	logActual := loggerBuffer.String()
	assert.EqualValues(t, logExpected, logActual)
}

func TestParseDebugInfoTrue(t *testing.T) {
	debugInfo := &config.DebugInfo{Allow: true}
	resDebugInfo := parseDebugInfo(debugInfo)
	assert.True(t, resDebugInfo, "Debug Allow value should be true")
}

func TestParseDebugInfoFalse(t *testing.T) {
	debugInfo := &config.DebugInfo{Allow: false}
	resDebugInfo := parseDebugInfo(debugInfo)
	assert.False(t, resDebugInfo, "Debug Allow value should be false")
}

func TestParseDebugInfoIsNil(t *testing.T) {
	resDebugInfo := parseDebugInfo(nil)
	assert.True(t, resDebugInfo, "Debug Allow value should be true")
}

func TestPrepareStoredResponse(t *testing.T) {
	result := prepareStoredResponse("imp_id1", json.RawMessage(`{"id": "resp_id1"}`))
	assert.Equal(t, []byte(ImpIdReqBody+"imp_id1"), result.request.Body, "incorrect request body")
	assert.Equal(t, []byte(`{"id": "resp_id1"}`), result.response.Body, "incorrect response body")
}

func TestRequestBidsWithAdsCertsSigner(t *testing.T) {
	respStatus := 200
	respBody := `{"bid":false}`
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte(`{"key":"val"}`),
			Headers: http.Header{},
		},
		bidResponse: nil,
	}
	bidderImpl.bidResponse = &adapters.BidderResponse{
		Bids: []*adapters.TypedBid{
			{
				Bid: &openrtb2.Bid{
					ID: "bidId",
				},
				BidType:      openrtb_ext.BidTypeBanner,
				DealPriority: 4,
			},
		},
	}

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: false}, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: "test",
	}
	ctx := context.Background()
	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 2.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: false,
		headerDebugAllowed:  false,
		addCallSignHeader:   true,
		bidAdjustments:      bidAdjustments,
	}
	_, errs := bidder.requestBid(ctx, bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &MockSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{})

	assert.Empty(t, errs, "no errors should be returned")
}

func wrapWithBidderInfo(bidder adapters.Bidder) adapters.Bidder {
	bidderInfo := config.BidderInfo{
		Disabled: false,
		Capabilities: &config.CapabilitiesInfo{
			App: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	return adapters.BuildInfoAwareBidder(bidder, bidderInfo)
}

type goodSingleBidder struct {
	bidRequest            *openrtb2.BidRequest
	httpRequest           *adapters.RequestData
	httpResponse          *adapters.ResponseData
	bidResponse           *adapters.BidderResponse
	hasStoredBidResponses bool
}

func (bidder *goodSingleBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	return []*adapters.RequestData{bidder.httpRequest}, nil
}

func (bidder *goodSingleBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponse = response
	return bidder.bidResponse, nil
}

type goodSingleBidderWithStoredBidResp struct {
}

func (bidder *goodSingleBidderWithStoredBidResp) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return nil, nil
}

func (bidder *goodSingleBidderWithStoredBidResp) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: openrtb_ext.BidTypeVideo,
			})
		}
	}
	return bidResponse, nil
}

type goodMultiHTTPCallsBidder struct {
	bidRequest        *openrtb2.BidRequest
	httpRequest       []*adapters.RequestData
	httpResponses     []*adapters.ResponseData
	bidResponses      []*adapters.BidderResponse
	bidResponseNumber int
}

func (bidder *goodMultiHTTPCallsBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	response := make([]*adapters.RequestData, len(bidder.httpRequest))

	for i, r := range bidder.httpRequest {
		response[i] = r
	}
	return response, nil
}

func (bidder *goodMultiHTTPCallsBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	br := bidder.bidResponses[bidder.bidResponseNumber]
	bidder.bidResponseNumber++
	bidder.httpResponses = append(bidder.httpResponses, response)

	return br, nil
}

type mixedMultiBidder struct {
	bidRequest    *openrtb2.BidRequest
	httpRequests  []*adapters.RequestData
	httpResponses []*adapters.ResponseData
	bidResponse   *adapters.BidderResponse
}

func (bidder *mixedMultiBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	return bidder.httpRequests, []error{errors.New("The requests weren't ideal.")}
}

func (bidder *mixedMultiBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponses = append(bidder.httpResponses, response)
	return bidder.bidResponse, []error{errors.New("The bidResponse weren't ideal.")}
}

type bidRejector struct {
	httpResponse *adapters.ResponseData
}

func (bidder *bidRejector) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return nil, []error{errors.New("Invalid params on BidRequest.")}
}

func (bidder *bidRejector) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponse = response
	return nil, []error{errors.New("Can't make a response.")}
}

type notifyingBidder struct {
	requests      []*adapters.RequestData
	notifyRequest adapters.RequestData
}

func (bidder *notifyingBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return bidder.requests, nil
}

func (bidder *notifyingBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, nil
}

func (bidder *notifyingBidder) MakeTimeoutNotification(req *adapters.RequestData) (*adapters.RequestData, []error) {
	return &bidder.notifyRequest, nil
}

func TestExtraBid(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID: "pubmaticImp1",
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
					Seat:         "pubmatic",
				},
				{
					Bid: &openrtb2.Bid{
						ID: "groupmImp1",
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
					Seat:         "groupm",
				},
			},
		},
	}

	wantSeatBids := []*entities.PbsOrtbSeatBid{
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid:            &openrtb2.Bid{ID: "groupmImp1"},
				DealPriority:   5,
				BidType:        openrtb_ext.BidTypeVideo,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     "groupm",
			Currency: "USD",
		},
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid:            &openrtb2.Bid{ID: "pubmaticImp1"},
				DealPriority:   4,
				BidType:        openrtb_ext.BidTypeBanner,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     string(openrtb_ext.BidderPubmatic),
			Currency: "USD",
		},
	}

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{}, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: openrtb_ext.BidderPubmatic,
	}

	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 2.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: false,
		headerDebugAllowed:  false,
		addCallSignHeader:   true,
		bidAdjustments:      bidAdjustments,
	}

	seatBids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &MockSigner{}, bidReqOptions,
		openrtb_ext.ExtAlternateBidderCodes{
			Enabled: true,
			Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				string(openrtb_ext.BidderPubmatic): {
					Enabled:            true,
					AllowedBidderCodes: []string{"groupm"},
				},
			},
		},
		&hookexecution.EmptyHookExecutor{})
	assert.Nil(t, errs)
	assert.Len(t, seatBids, 2)
	sort.Slice(seatBids, func(i, j int) bool {
		return len(seatBids[i].Seat) < len(seatBids[j].Seat)
	})
	assert.Equal(t, wantSeatBids, seatBids)
}

func TestExtraBidWithAlternateBidderCodeDisabled(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID: "pubmaticImp1",
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
					Seat:         "pubmatic",
				},
				{
					Bid: &openrtb2.Bid{
						ID: "groupmImp1",
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
					Seat:         "groupm-rejected",
				},
				{
					Bid: &openrtb2.Bid{
						ID: "groupmImp2",
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
					Seat:         "groupm-allowed",
				},
			},
		},
	}

	wantSeatBids := []*entities.PbsOrtbSeatBid{
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid:            &openrtb2.Bid{ID: "groupmImp2"},
				DealPriority:   5,
				BidType:        openrtb_ext.BidTypeVideo,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     "groupm-allowed",
			Currency: "USD",
		},
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid:            &openrtb2.Bid{ID: "pubmaticImp1"},
				DealPriority:   4,
				BidType:        openrtb_ext.BidTypeBanner,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     string(openrtb_ext.BidderPubmatic),
			Currency: "USD",
		},
	}
	wantErrs := []error{
		&errortypes.Warning{
			WarningCode: errortypes.AlternateBidderCodeWarningCode,
			Message:     `invalid biddercode "groupm-rejected" sent by adapter "pubmatic"`,
		},
	}

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{}, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: openrtb_ext.BidderPubmatic,
	}
	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 2.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: false,
		headerDebugAllowed:  false,
		addCallSignHeader:   true,
		bidAdjustments:      bidAdjustments,
	}

	seatBids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &MockSigner{}, bidReqOptions,
		openrtb_ext.ExtAlternateBidderCodes{
			Enabled: true,
			Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				string(openrtb_ext.BidderPubmatic): {
					Enabled:            true,
					AllowedBidderCodes: []string{"groupm-allowed"},
				},
			},
		},
		&hookexecution.EmptyHookExecutor{})
	assert.Equal(t, wantErrs, errs)
	assert.Len(t, seatBids, 2)
	assert.ElementsMatch(t, wantSeatBids, seatBids)
}

func TestExtraBidWithBidAdjustments(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "pubmaticImp1",
						Price: 3,
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
					Seat:         "pubmatic",
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "groupmImp1",
						Price: 7,
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
					Seat:         "groupm",
				},
			},
		},
	}

	wantSeatBids := []*entities.PbsOrtbSeatBid{
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid: &openrtb2.Bid{
					ID:    "groupmImp1",
					Price: 21,
				},
				DealPriority:   5,
				BidType:        openrtb_ext.BidTypeVideo,
				OriginalBidCPM: 7,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     "groupm",
			Currency: "USD",
		},
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid: &openrtb2.Bid{
					ID:    "pubmaticImp1",
					Price: 6,
				},
				DealPriority:   4,
				BidType:        openrtb_ext.BidTypeBanner,
				OriginalBidCur: "USD",
				OriginalBidCPM: 3,
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     string(openrtb_ext.BidderPubmatic),
			Currency: "USD",
		},
	}

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{}, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: openrtb_ext.BidderPubmatic,
	}
	bidAdjustments := map[string]float64{
		string(openrtb_ext.BidderPubmatic): 2,
		string(openrtb_ext.BidderGroupm):   3,
	}

	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: false,
		headerDebugAllowed:  false,
		addCallSignHeader:   true,
		bidAdjustments:      bidAdjustments,
	}

	seatBids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &MockSigner{}, bidReqOptions,
		openrtb_ext.ExtAlternateBidderCodes{
			Enabled: true,
			Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				string(openrtb_ext.BidderPubmatic): {
					Enabled:            true,
					AllowedBidderCodes: []string{"groupm"},
				},
			},
		},
		&hookexecution.EmptyHookExecutor{})
	assert.Nil(t, errs)
	assert.Len(t, seatBids, 2)
	sort.Slice(seatBids, func(i, j int) bool {
		return len(seatBids[i].Seat) < len(seatBids[j].Seat)
	})
	assert.Equal(t, wantSeatBids, seatBids)
}

func TestExtraBidWithBidAdjustmentsUsingAdapterCode(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "pubmaticImp1",
						Price: 3,
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
					Seat:         "pubmatic",
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "groupmImp1",
						Price: 7,
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
					Seat:         "groupm",
				},
			},
		},
	}

	wantSeatBids := []*entities.PbsOrtbSeatBid{
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid: &openrtb2.Bid{
					ID:    "groupmImp1",
					Price: 14,
				},
				DealPriority:   5,
				BidType:        openrtb_ext.BidTypeVideo,
				OriginalBidCPM: 7,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     "groupm",
			Currency: "USD",
		},
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid: &openrtb2.Bid{
					ID:    "pubmaticImp1",
					Price: 6,
				},
				DealPriority:   4,
				BidType:        openrtb_ext.BidTypeBanner,
				OriginalBidCur: "USD",
				OriginalBidCPM: 3,
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     string(openrtb_ext.BidderPubmatic),
			Currency: "USD",
		},
	}

	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{}, "")
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}},
		BidderName: openrtb_ext.BidderPubmatic,
	}
	bidAdjustments := map[string]float64{
		string(openrtb_ext.BidderPubmatic): 2,
	}

	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: false,
		headerDebugAllowed:  false,
		addCallSignHeader:   true,
		bidAdjustments:      bidAdjustments,
	}

	seatBids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &MockSigner{}, bidReqOptions,
		openrtb_ext.ExtAlternateBidderCodes{
			Enabled: true,
			Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				string(openrtb_ext.BidderPubmatic): {
					Enabled:            true,
					AllowedBidderCodes: []string{"groupm"},
				},
			},
		},
		&hookexecution.EmptyHookExecutor{})
	assert.Nil(t, errs)
	assert.Len(t, seatBids, 2)
	sort.Slice(seatBids, func(i, j int) bool {
		return len(seatBids[i].Seat) < len(seatBids[j].Seat)
	})
	assert.Equal(t, wantSeatBids, seatBids)
}

func TestExtraBidWithMultiCurrencies(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "pubmaticImp1",
						Price: 3,
					},
					BidType:      openrtb_ext.BidTypeBanner,
					DealPriority: 4,
					Seat:         "pubmatic",
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "groupmImp1",
						Price: 7,
					},
					BidType:      openrtb_ext.BidTypeVideo,
					DealPriority: 5,
					Seat:         "groupm",
				},
			},
		},
	}

	wantSeatBids := []*entities.PbsOrtbSeatBid{
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid: &openrtb2.Bid{
					ID:    "groupmImp1",
					Price: 571.5994430039375,
				},
				DealPriority:   5,
				BidType:        openrtb_ext.BidTypeVideo,
				OriginalBidCPM: 7,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     "groupm",
			Currency: "INR",
		},
		{
			HttpCalls: []*openrtb_ext.ExtHttpCall{},
			Bids: []*entities.PbsOrtbBid{{
				Bid: &openrtb2.Bid{
					ID:    "pubmaticImp1",
					Price: 244.97118985883034,
				},
				DealPriority:   4,
				BidType:        openrtb_ext.BidTypeBanner,
				OriginalBidCPM: 3,
				OriginalBidCur: "USD",
				BidMeta:        &openrtb_ext.ExtBidPrebidMeta{AdapterCode: string(openrtb_ext.BidderPubmatic)},
			}},
			Seat:     string(openrtb_ext.BidderPubmatic),
			Currency: "INR",
		},
	}

	mockedHTTPServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.Write([]byte(`{"dataAsOf":"2022-11-24T00:00:00.000Z","generatedAt":"2022-11-24T15:00:46.363Z","conversions":{"USD":{"USD":1,"INR":81.65706328627678}}}`))
			rw.WriteHeader(http.StatusOK)
		}),
	)

	// Execute:
	bidder := AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
	currencyConverter := currency.NewRateConverter(
		&http.Client{},
		mockedHTTPServer.URL,
		time.Duration(24)*time.Hour,
	)
	time.Sleep(time.Duration(500) * time.Millisecond)
	currencyConverter.Run()

	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "impId"}}, Cur: []string{"INR"}},
		BidderName: openrtb_ext.BidderPubmatic,
	}

	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 2.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: false,
		headerDebugAllowed:  false,
		addCallSignHeader:   true,
		bidAdjustments:      bidAdjustments,
	}

	seatBids, errs := bidder.requestBid(context.Background(), bidderReq, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, &MockSigner{}, bidReqOptions,
		openrtb_ext.ExtAlternateBidderCodes{
			Enabled: true,
			Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				string(openrtb_ext.BidderPubmatic): {
					Enabled:            true,
					AllowedBidderCodes: []string{"groupm"},
				},
			},
		},
		&hookexecution.EmptyHookExecutor{})
	assert.Nil(t, errs)
	assert.Len(t, seatBids, 2)
	sort.Slice(seatBids, func(i, j int) bool {
		return len(seatBids[i].Seat) < len(seatBids[j].Seat)
	})
	assert.Equal(t, wantSeatBids, seatBids)
}
