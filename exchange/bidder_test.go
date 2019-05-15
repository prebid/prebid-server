package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// TestSingleBidder makes sure that the following things work if the Bidder needs only one request.
//
// 1. The Bidder implementation is called with the arguments we expect.
// 2. The returned values are correct for a non-test bid.
func TestSingleBidder(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	requestHeaders := http.Header{}
	requestHeaders.Add("Content-Type", "application/json")

	bidAdjustment := 2.0
	firstInitialPrice := 3.0
	secondInitialPrice := 4.0
	mockBidderResponse := &adapters.BidderResponse{
		Bids: []*adapters.TypedBid{
			{
				Bid: &openrtb.Bid{
					Price: firstInitialPrice,
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			{
				Bid: &openrtb.Bid{
					Price: secondInitialPrice,
				},
				BidType: openrtb_ext.BidTypeVideo,
			},
		},
	}

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: mockBidderResponse,
	}
	bidder := adaptBidder(bidderImpl, server.Client())
	currencyConverter := currencies.NewRateConverterDefault()
	seatBid, errs := bidder.requestBid(context.Background(), &openrtb.BidRequest{}, "test", bidAdjustment, currencyConverter.Rates())

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
	if len(errs) != 0 {
		t.Errorf("bidder.Bid returned %d errors. Expected 0", len(errs))
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
	}
	if mockBidderResponse.Bids[0].Bid.Price != bidAdjustment*firstInitialPrice {
		t.Errorf("Bid[0].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*firstInitialPrice, mockBidderResponse.Bids[0].Bid.Price)
	}
	if mockBidderResponse.Bids[1].Bid.Price != bidAdjustment*secondInitialPrice {
		t.Errorf("Bid[1].Price was not adjusted properly. Expected %f, got %f", bidAdjustment*secondInitialPrice, mockBidderResponse.Bids[1].Bid.Price)
	}
	if len(seatBid.httpCalls) != 0 {
		t.Errorf("The bidder shouldn't log HttpCalls when request.test == 0. Found %d", len(seatBid.httpCalls))
	}

	if len(seatBid.ext) != 0 {
		t.Errorf("The bidder shouldn't define any seatBid.ext. Got %s", string(seatBid.ext))
	}
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
				Bid:     &openrtb.Bid{},
				BidType: openrtb_ext.BidTypeBanner,
			},
			{
				Bid:     &openrtb.Bid{},
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
	bidder := adaptBidder(bidderImpl, server.Client())
	currencyConverter := currencies.NewRateConverterDefault()
	seatBid, errs := bidder.requestBid(context.Background(), &openrtb.BidRequest{}, "test", 1.0, currencyConverter.Rates())

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
		Bidder: &mixedMultiBidder{},
		Client: server.Client(),
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
		Bidder: &mixedMultiBidder{},
		Client: server.Client(),
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
		rates                     currencies.Rates
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
				errors.New("Currency conversion rate not found: 'JPY' => 'USD'"),
			},
			description: "Case 6 - Bidder respond with a mix of currencies and one unknown on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "JPY", price: 1.1},
				{currency: "BZD", price: 1.2},
				{currency: "DKK", price: 1.3},
			},
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
				errors.New("Currency conversion rate not found: 'JPY' => 'USD'"),
				errors.New("Currency conversion rate not found: 'BZD' => 'USD'"),
				errors.New("Currency conversion rate not found: 'DKK' => 'USD'"),
			},
			description: "Case 7 - Bidder respond with currencies not having any rate on all HTTP responses",
		},
		{
			bids: []bid{
				{currency: "AAA", price: 1.1},
				{currency: "BBB", price: 1.2},
				{currency: "CCC", price: 1.3},
			},
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
						Bid: &openrtb.Bid{
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
		bidder := adaptBidder(bidderImpl, server.Client())
		currencyConverter := currencies.NewRateConverter(
			&http.Client{},
			mockedHTTPServer.URL,
			time.Duration(10)*time.Second,
		)
		seatBid, errs := bidder.requestBid(
			context.Background(),
			&openrtb.BidRequest{},
			"test",
			1,
			currencyConverter.Rates(),
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
				fmt.Errorf("Constant rates doesn't proceed to any conversions, cannot convert 'EUR' => 'USD'"),
				fmt.Errorf("Constant rates doesn't proceed to any conversions, cannot convert 'EUR' => 'USD'"),
				fmt.Errorf("Constant rates doesn't proceed to any conversions, cannot convert 'EUR' => 'USD'"),
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
				fmt.Errorf("Constant rates doesn't proceed to any conversions, cannot convert 'EUR' => 'USD'"),
			},
			description: "Case 7 - Bidder responds with a mix of not set, non default currency and default currency in HTTP responses",
		},
		{
			bidCurrency:       []string{"GBP", "", "USD"},
			expectedBidsCount: 2,
			expectedBadCurrencyErrors: []error{
				fmt.Errorf("Constant rates doesn't proceed to any conversions, cannot convert 'GBP' => 'USD'"),
			},
			description: "Case 8 - Bidder responds with a mix of not set, non default currency and default currency in HTTP responses",
		},
		{
			bidCurrency:       []string{"GBP", "", ""},
			expectedBidsCount: 2,
			expectedBadCurrencyErrors: []error{
				fmt.Errorf("Constant rates doesn't proceed to any conversions, cannot convert 'GBP' => 'USD'"),
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
						Bid:     &openrtb.Bid{},
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
		bidder := adaptBidder(bidderImpl, server.Client())
		currencyConverter := currencies.NewRateConverterDefault()
		seatBid, errs := bidder.requestBid(
			context.Background(),
			&openrtb.BidRequest{},
			"test",
			1,
			currencyConverter.Rates(),
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
		rates                  currencies.Rates
		description            string
	}{
		{
			bidRequestCurrencies:   []string{"EUR", "USD", "JPY"},
			bidResponsesCurrency:   "EUR",
			expectedPickedCurrency: "EUR",
			expectedError:          false,
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf: time.Now(),
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
			rates: currencies.Rates{
				DataAsOf:    time.Now(),
				Conversions: map[string]map[string]float64{},
			},
			description: "Case 4 - None allowed currencies in bid request are known, an error is returned",
		},
		{
			bidRequestCurrencies:   []string{"CNY", "EUR", "JPY", "USD"},
			bidResponsesCurrency:   "USD",
			expectedPickedCurrency: "USD",
			expectedError:          false,
			rates: currencies.Rates{
				DataAsOf:    time.Now(),
				Conversions: map[string]map[string]float64{},
			},
			description: "Case 5 - None allowed currencies in bid request are known but the default one (`USD`), no rates are set but default currency will be picked",
		},
		{
			bidRequestCurrencies:   nil,
			bidResponsesCurrency:   "USD",
			expectedPickedCurrency: "USD",
			expectedError:          false,
			rates: currencies.Rates{
				DataAsOf:    time.Now(),
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
						Bid:     &openrtb.Bid{},
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
		bidder := adaptBidder(bidderImpl, server.Client())
		currencyConverter := currencies.NewRateConverter(
			&http.Client{},
			mockedHTTPServer.URL,
			time.Duration(10)*time.Second,
		)
		seatBid, errs := bidder.requestBid(
			context.Background(),
			&openrtb.BidRequest{
				Cur: tc.bidRequestCurrencies,
			},
			"test",
			1,
			currencyConverter.Rates(),
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

// TestBadResponseLogging makes sure that openrtb_ext works properly on malformed HTTP requests.
func TestBadRequestLogging(t *testing.T) {
	info := &httpCallInfo{
		err: errors.New("Bad request"),
	}
	ext := makeExt(info)
	if ext.Uri != "" {
		t.Errorf("The URI should be empty. Got %s", ext.Uri)
	}
	if ext.RequestBody != "" {
		t.Errorf("The request body should be empty. Got %s", ext.RequestBody)
	}
	if ext.ResponseBody != "" {
		t.Errorf("The response body should be empty. Got %s", ext.ResponseBody)
	}
	if ext.Status != 0 {
		t.Errorf("The Status code should be 0. Got %d", ext.Status)
	}
}

// TestBadResponseLogging makes sure that openrtb_ext works properly if we don't get a sensible HTTP response.
func TestBadResponseLogging(t *testing.T) {
	info := &httpCallInfo{
		request: &adapters.RequestData{
			Uri:  "test.com",
			Body: []byte("request body"),
		},
		err: errors.New("Bad response"),
	}
	ext := makeExt(info)
	if ext.Uri != info.request.Uri {
		t.Errorf("The URI should be test.com. Got %s", ext.Uri)
	}
	if ext.RequestBody != string(info.request.Body) {
		t.Errorf("The request body should be empty. Got %s", ext.RequestBody)
	}
	if ext.ResponseBody != "" {
		t.Errorf("The response body should be empty. Got %s", ext.ResponseBody)
	}
	if ext.Status != 0 {
		t.Errorf("The Status code should be 0. Got %d", ext.Status)
	}
}

// TestSuccessfulResponseLogging makes sure that openrtb_ext works properly if the HTTP request is successful.
func TestSuccessfulResponseLogging(t *testing.T) {
	info := &httpCallInfo{
		request: &adapters.RequestData{
			Uri:  "test.com",
			Body: []byte("request body"),
		},
		response: &adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte("response body"),
		},
	}
	ext := makeExt(info)
	if ext.Uri != info.request.Uri {
		t.Errorf("The URI should be test.com. Got %s", ext.Uri)
	}
	if ext.RequestBody != string(info.request.Body) {
		t.Errorf("The request body should be \"request body\". Got %s", ext.RequestBody)
	}
	if ext.ResponseBody != string(info.response.Body) {
		t.Errorf("The response body should be \"response body\". Got %s", ext.ResponseBody)
	}
	if ext.Status != info.response.StatusCode {
		t.Errorf("The Status code should be 0. Got %d", ext.Status)
	}
}

// TestServerCallDebugging makes sure that we log the server calls made by the Bidder on test bids.
func TestServerCallDebugging(t *testing.T) {
	respBody := "{\"bid\":false}"
	respStatus := 200
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	reqBody := "{\"key\":\"val\"}"
	reqUrl := server.URL
	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     reqUrl,
			Body:    []byte(reqBody),
			Headers: http.Header{},
		},
	}
	bidder := adaptBidder(bidderImpl, server.Client())
	currencyConverter := currencies.NewRateConverterDefault()

	bids, _ := bidder.requestBid(
		context.Background(),
		&openrtb.BidRequest{
			Test: 1,
		},
		"test",
		1.0,
		currencyConverter.Rates(),
	)

	if len(bids.httpCalls) != 1 {
		t.Errorf("We should log the server call if this is a test bid. Got %d", len(bids.httpCalls))
	}
	if bids.httpCalls[0].Uri != reqUrl {
		t.Errorf("Wrong httpcalls URI. Expected %s, got %s", reqUrl, bids.httpCalls[0].Uri)
	}
	if bids.httpCalls[0].RequestBody != reqBody {
		t.Errorf("Wrong httpcalls RequestBody. Expected %s, got %s", reqBody, bids.httpCalls[0].RequestBody)
	}
	if bids.httpCalls[0].ResponseBody != respBody {
		t.Errorf("Wrong httpcalls ResponseBody. Expected %s, got %s", respBody, bids.httpCalls[0].ResponseBody)
	}
	if bids.httpCalls[0].Status != respStatus {
		t.Errorf("Wrong httpcalls Status. Expected %d, got %d", respStatus, bids.httpCalls[0].Status)
	}
}

func TestErrorReporting(t *testing.T) {
	bidder := adaptBidder(&bidRejector{}, nil)
	currencyConverter := currencies.NewRateConverterDefault()
	bids, errs := bidder.requestBid(context.Background(), &openrtb.BidRequest{}, "test", 1.0, currencyConverter.Rates())
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

type goodSingleBidder struct {
	bidRequest   *openrtb.BidRequest
	httpRequest  *adapters.RequestData
	httpResponse *adapters.ResponseData
	bidResponse  *adapters.BidderResponse
}

func (bidder *goodSingleBidder) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	return []*adapters.RequestData{bidder.httpRequest}, nil
}

func (bidder *goodSingleBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponse = response
	return bidder.bidResponse, nil
}

type goodMultiHTTPCallsBidder struct {
	bidRequest        *openrtb.BidRequest
	httpRequest       []*adapters.RequestData
	httpResponses     []*adapters.ResponseData
	bidResponses      []*adapters.BidderResponse
	bidResponseNumber int
}

func (bidder *goodMultiHTTPCallsBidder) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	response := make([]*adapters.RequestData, len(bidder.httpRequest))

	for i, r := range bidder.httpRequest {
		response[i] = r
	}
	return response, nil
}

func (bidder *goodMultiHTTPCallsBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	br := bidder.bidResponses[bidder.bidResponseNumber]
	bidder.bidResponseNumber++
	bidder.httpResponses = append(bidder.httpResponses, response)

	return br, nil
}

type mixedMultiBidder struct {
	bidRequest    *openrtb.BidRequest
	httpRequests  []*adapters.RequestData
	httpResponses []*adapters.ResponseData
	bidResponse   *adapters.BidderResponse
}

func (bidder *mixedMultiBidder) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	bidder.bidRequest = request
	return bidder.httpRequests, []error{errors.New("The requests weren't ideal.")}
}

func (bidder *mixedMultiBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponses = append(bidder.httpResponses, response)
	return bidder.bidResponse, []error{errors.New("The bidResponse weren't ideal.")}
}

type bidRejector struct {
	httpRequest  *adapters.RequestData
	httpResponse *adapters.ResponseData
}

func (bidder *bidRejector) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	return nil, []error{errors.New("Invalid params on BidRequest.")}
}

func (bidder *bidRejector) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidder.httpResponse = response
	return nil, []error{errors.New("Can't make a response.")}
}
