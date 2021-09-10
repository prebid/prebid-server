package exchange

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	nativeRequests "github.com/mxmCherry/openrtb/v15/native1/request"
	nativeResponse "github.com/mxmCherry/openrtb/v15/native1/response"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
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

	bidAdjustment := 2.0
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
	ctx = context.WithValue(ctx, DebugContextKey, true)

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

		bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, test.debugInfo)
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		seatBid, errs := bidder.requestBid(ctx, &openrtb2.BidRequest{}, "test", bidAdjustment, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, true, false)

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
		if len(seatBid.bids) != len(mockBidderResponse.Bids) {
			t.Fatalf("Expected %d bids. Got %d", len(mockBidderResponse.Bids), len(seatBid.bids))
		}
		for index, typedBid := range mockBidderResponse.Bids {
			if typedBid.Bid != seatBid.bids[index].bid {
				t.Errorf("Bid %d did not point to the same bid returned by the Bidder.", index)
			}
			if typedBid.BidType != seatBid.bids[index].bidType {
				t.Errorf("Bid %d did not have the right type. Expected %s, got %s", index, typedBid.BidType, seatBid.bids[index].bidType)
			}
			if typedBid.DealPriority != seatBid.bids[index].dealPriority {
				t.Errorf("Bid %d did not have the right deal priority. Expected %s, got %s", index, typedBid.BidType, seatBid.bids[index].bidType)
			}
		}
		if mockBidderResponse.Bids[0].Bid.Price != bidAdjustment*firstInitialPrice {
			t.Errorf("Bid[0].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*firstInitialPrice, mockBidderResponse.Bids[0].Bid.Price)
		}
		if mockBidderResponse.Bids[1].Bid.Price != bidAdjustment*secondInitialPrice {
			t.Errorf("Bid[1].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*secondInitialPrice, mockBidderResponse.Bids[1].Bid.Price)
		}
		if len(seatBid.httpCalls) != test.httpCallsLen {
			t.Errorf("The bidder shouldn't log HttpCalls when request.test == 0. Found %d", len(seatBid.httpCalls))
		}
	}
}

func TestRequestBidRemovesSensitiveHeaders(t *testing.T) {
	server := httptest.NewServer(mockHandler(200, "getBody", "responseJson"))
	defer server.Close()

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
	ctx = context.WithValue(ctx, DebugContextKey, true)

	bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, debugInfo)
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	seatBid, errs := bidder.requestBid(ctx, &openrtb2.BidRequest{}, "test", 1, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, true, false)

	expectedHttpCalls := []*openrtb_ext.ExtHttpCall{
		{
			Uri:            server.URL,
			RequestBody:    "requestJson",
			RequestHeaders: map[string][]string{"Content-Type": {"application/json"}},
			ResponseBody:   "responseJson",
			Status:         200,
		},
	}

	assert.Empty(t, errs)
	assert.ElementsMatch(t, seatBid.httpCalls, expectedHttpCalls)
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
	ctx = context.WithValue(ctx, DebugContextKey, true)

	bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, debugInfo)
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	seatBid, errs := bidder.requestBid(ctx, &openrtb2.BidRequest{}, "test", 1, currencyConverter.Rates(), &adapters.ExtraRequestInfo{GlobalPrivacyControlHeader: "1"}, true, false)

	expectedHttpCall := []*openrtb_ext.ExtHttpCall{
		{
			Uri:            server.URL,
			RequestBody:    "requestJson",
			RequestHeaders: map[string][]string{"Content-Type": {"application/json"}, "Sec-Gpc": {"1"}},
			ResponseBody:   "responseJson",
			Status:         200,
		},
	}

	assert.Empty(t, errs)
	assert.ElementsMatch(t, seatBid.httpCalls, expectedHttpCall)
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
	ctx = context.WithValue(ctx, DebugContextKey, true)

	bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, debugInfo)
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	seatBid, errs := bidder.requestBid(ctx, &openrtb2.BidRequest{}, "test", 1, currencyConverter.Rates(), &adapters.ExtraRequestInfo{GlobalPrivacyControlHeader: "1"}, true, false)

	expectedHttpCall := []*openrtb_ext.ExtHttpCall{
		{
			Uri:            server.URL,
			RequestBody:    "requestJson",
			RequestHeaders: map[string][]string{"Sec-Gpc": {"1"}},
			ResponseBody:   "responseJson",
			Status:         200,
		},
	}

	assert.Empty(t, errs)
	assert.ElementsMatch(t, seatBid.httpCalls, expectedHttpCall)
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
	bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	seatBid, errs := bidder.requestBid(context.Background(), &openrtb2.BidRequest{}, "test", 1.0, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, true, true)

	if seatBid == nil {
		t.Fatalf("SeatBid should exist, because bids exist.")
	}

	if len(errs) != 1+len(bidderImpl.httpRequests) {
		t.Errorf("Expected %d errors. Got %d", 1+len(bidderImpl.httpRequests), len(errs))
	}
	if len(seatBid.bids) != len(bidderImpl.httpResponses)*len(mockBidderResponse.Bids) {
		t.Errorf("Expected %d bids. Got %d", len(bidderImpl.httpResponses)*len(mockBidderResponse.Bids), len(seatBid.bids))
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
		me:         &metricsConfig.DummyMetricsEngine{},
	}

	callInfo := bidder.doRequest(ctx, &adapters.RequestData{
		Method: "POST",
		Uri:    server.URL,
	})
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
	})
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
		me:         &metricsConfig.DummyMetricsEngine{},
	}

	callInfo := bidder.doRequest(context.Background(), &adapters.RequestData{
		Method: "POST",
		Uri:    server.URL,
	})
	if callInfo.err == nil {
		t.Errorf("bidderAdapter.doRequest should return an error if the connection closes unexpectedly.")
	}
}

type bid struct {
	currency string
	price    float64
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
				{currency: "USD", price: 1.1},
				{currency: "USD", price: 1.2},
				{currency: "USD", price: 1.3},
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
				{currency: "USD", price: 1.1},
				{currency: "USD", price: 1.2},
				{currency: "USD", price: 1.3},
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
				{currency: "USD", price: 1.1 * 1.1435678764},
				{currency: "USD", price: 1.2 * 1.1435678764},
				{currency: "USD", price: 1.3 * 1.1435678764},
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
				{currency: "USD", price: 1.1},
				{currency: "USD", price: 1.2 * 1.1435678764},
				{currency: "USD", price: 1.3 * 1.3050530256},
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
				{currency: "USD", price: 1.1},
				{currency: "USD", price: 1.2 * 1.1435678764},
				{currency: "USD", price: 1.3 * 1.3050530256},
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
				{currency: "USD", price: 1.2 * 1.1435678764},
				{currency: "USD", price: 1.3 * 1.3050530256},
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
		bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
		currencyConverter := currency.NewRateConverter(
			&http.Client{},
			mockedHTTPServer.URL,
			time.Duration(24)*time.Hour,
		)
		time.Sleep(time.Duration(500) * time.Millisecond)
		currencyConverter.Run()

		seatBid, errs := bidder.requestBid(
			context.Background(),
			&openrtb2.BidRequest{},
			"test",
			1,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			true,
			true,
		)

		// Verify:
		resultLightBids := make([]bid, len(seatBid.bids))
		for i, b := range seatBid.bids {
			resultLightBids[i] = bid{
				price:    b.bid.Price,
				currency: seatBid.currency,
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
		bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
		seatBid, errs := bidder.requestBid(
			context.Background(),
			&openrtb2.BidRequest{},
			"test",
			1,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			true,
			true,
		)

		// Verify:
		assert.Equal(t, false, (seatBid == nil && tc.expectedBidsCount != 0), tc.description)
		assert.Equal(t, tc.expectedBidsCount, uint(len(seatBid.bids)), tc.description)
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
		bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
		currencyConverter := currency.NewRateConverter(
			&http.Client{},
			mockedHTTPServer.URL,
			time.Duration(24)*time.Hour,
		)
		seatBid, errs := bidder.requestBid(
			context.Background(),
			&openrtb2.BidRequest{
				Cur: tc.bidRequestCurrencies,
			},
			"test",
			1,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			true,
			false,
		)

		// Verify:
		if tc.expectedError {
			assert.NotNil(t, errs, tc.description)
		} else {
			assert.Nil(t, errs, tc.description)
			assert.Equal(t, tc.expectedPickedCurrency, seatBid.currency, tc.description)
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
		bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
		currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

		seatBids, _ := bidder.requestBid(
			context.Background(),
			tc.mockBidderRequest,
			"test",
			1.0,
			currencyConverter.Rates(),
			&adapters.ExtraRequestInfo{},
			true,
			true,
		)

		var actualValue string
		for _, bid := range seatBids.bids {
			actualValue = bid.bid.AdM
			diffJson(t, tc.description, []byte(actualValue), []byte(tc.expectedValue))
		}
	}
}

func TestErrorReporting(t *testing.T) {
	bidder := adaptBidder(&bidRejector{}, nil, &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	bids, errs := bidder.requestBid(context.Background(), &openrtb2.BidRequest{}, "test", 1.0, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, true, false)
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
	bidAdjustment := 2.0

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{},
	}

	// setup a mock metrics engine and its expectation
	metrics := &metrics.MetricsEngineMock{}
	expectedAdapterName := openrtb_ext.BidderAppnexus
	compareConnWaitTime := func(dur time.Duration) bool { return dur.Nanoseconds() > 0 }

	metrics.On("RecordAdapterConnections", expectedAdapterName, false, mock.MatchedBy(compareConnWaitTime)).Once()

	// Run requestBid using an http.Client with a mock handler
	bidder := adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, metrics, openrtb_ext.BidderAppnexus, nil)
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	_, errs := bidder.requestBid(context.Background(), &openrtb2.BidRequest{}, "test", bidAdjustment, currencyConverter.Rates(), &adapters.ExtraRequestInfo{}, true, true)

	// Assert no errors
	assert.Equal(t, 0, len(errs), "bidder.requestBid returned errors %v \n", errs)

	// Assert RecordAdapterConnections() was called with the parameters we expected
	metrics.AssertExpectations(t)
}

type DNSDoneTripper struct{}

func (DNSDoneTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Access the httptrace.ClientTrace
	trace := httptrace.ContextClientTrace(req.Context())
	// Call the DNSDone method on the client trace
	trace.DNSDone(httptrace.DNSDoneInfo{})

	resp := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader("postBody")),
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
		Body:       ioutil.NopCloser(strings.NewReader("postBody")),
	}

	return resp, nil
}

func TestCallRecordDNSTime(t *testing.T) {
	// setup a mock metrics engine and its expectation
	metricsMock := &metrics.MetricsEngineMock{}
	metricsMock.Mock.On("RecordDNSTime", mock.Anything).Return()

	// Instantiate the bidder that will send the request. We'll make sure to use an
	// http.Client that runs our mock RoundTripper so DNSDone(httptrace.DNSDoneInfo{})
	// gets called
	bidder := &bidderAdapter{
		Bidder: &mixedMultiBidder{},
		Client: &http.Client{Transport: DNSDoneTripper{}},
		me:     metricsMock,
	}

	// Run test
	bidder.doRequest(context.Background(), &adapters.RequestData{Method: "POST", Uri: "http://www.example.com/"})

	// Tried one or another, none seem to work without panicking
	metricsMock.AssertExpectations(t)
}

func TestCallRecordTLSHandshakeTime(t *testing.T) {
	// setup a mock metrics engine and its expectation
	metricsMock := &metrics.MetricsEngineMock{}
	metricsMock.Mock.On("RecordTLSHandshakeTime", mock.Anything).Return()

	// Instantiate the bidder that will send the request. We'll make sure to use an
	// http.Client that runs our mock RoundTripper so DNSDone(httptrace.DNSDoneInfo{})
	// gets called
	bidder := &bidderAdapter{
		Bidder: &mixedMultiBidder{},
		Client: &http.Client{Transport: TLSHandshakeTripper{}},
		me:     metricsMock,
	}

	// Run test
	bidder.doRequest(context.Background(), &adapters.RequestData{Method: "POST", Uri: "http://www.example.com/"})

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
		me:     &metricsConfig.DummyMetricsEngine{},
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
		me: &metricsConfig.DummyMetricsEngine{},
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

	bidderAdapter.doRequestImpl(ctx, &bidRequest, logger)

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

func wrapWithBidderInfo(bidder adapters.Bidder) adapters.Bidder {
	bidderInfo := config.BidderInfo{
		Enabled: true,
		Capabilities: &config.CapabilitiesInfo{
			App: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	return adapters.BuildInfoAwareBidder(bidder, bidderInfo)
}

type goodSingleBidder struct {
	bidRequest   *openrtb2.BidRequest
	httpRequest  *adapters.RequestData
	httpResponse *adapters.ResponseData
	bidResponse  *adapters.BidderResponse
}

func (bidder *goodSingleBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	return []*adapters.RequestData{bidder.httpRequest}, nil
}

func (bidder *goodSingleBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponse = response
	return bidder.bidResponse, nil
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
	httpRequest  *adapters.RequestData
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
