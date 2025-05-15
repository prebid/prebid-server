package startio

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderStartIO, config.Adapter{
		Endpoint: "http://localhost:8080/bidder/?identifier=test"}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "startiotest", bidder)
}
func TestBuilder(t *testing.T) {
	testCases := []struct {
		name        string
		endpoint    string
		expectError bool
	}{
		{
			name:        "Valid endpoint",
			endpoint:    "http://valid-endpoint.com",
			expectError: false,
		},
		{
			name:        "Invalid endpoint",
			endpoint:    "invalid-url",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Adapter{Endpoint: tc.endpoint}
			bidder, err := Builder(openrtb_ext.BidderStartIO, cfg, config.Server{})
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, bidder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, bidder)
				startioAdapter, ok := bidder.(*adapter)
				assert.True(t, ok)
				assert.Equal(t, tc.endpoint, startioAdapter.endpoint)
			}
		})
	}
}

func TestMakeRequests(t *testing.T) {
	testCases := []struct {
		name           string
		bidRequest     *openrtb2.BidRequest
		expectedErrors []error
		expectedReqLen int
	}{
		{
			name: "Valid request with single impression",
			bidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{ID: "123"},
				Imp: []openrtb2.Imp{
					{ID: "1", Banner: &openrtb2.Banner{}},
				},
			},
			expectedReqLen: 1,
		},
		{
			name: "Unsupported currency",
			bidRequest: &openrtb2.BidRequest{
				Cur: []string{"EUR"},
				App: &openrtb2.App{ID: "456"},
				Imp: []openrtb2.Imp{
					{ID: "1", Banner: &openrtb2.Banner{}},
				},
			},
			expectedErrors: []error{wrapReqError("unsupported currency: only USD is accepted")},
		},
		{
			name: "Missing site and app ID",
			bidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "1", Banner: &openrtb2.Banner{}},
				},
			},
			expectedErrors: []error{wrapReqError("request must contain either site.id or app.id")},
		},
		{
			name: "Mixed valid and invalid impressions",
			bidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{ID: "123"},
				Imp: []openrtb2.Imp{
					{ID: "1", Banner: &openrtb2.Banner{}},
					{ID: "2", Audio: &openrtb2.Audio{}},
					{ID: "3", Video: &openrtb2.Video{}},
				},
			},
			expectedReqLen: 2,
		},
	}

	adapter := &adapter{endpoint: "http://test-endpoint.com"}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqs, errs := adapter.MakeRequests(tc.bidRequest, &adapters.ExtraRequestInfo{})

			assert.Equal(t, len(tc.expectedErrors), len(errs))
			if len(tc.expectedErrors) > 0 {
				assert.Contains(t, errs[0].Error(), tc.expectedErrors[0].Error())
			}

			assert.Equal(t, tc.expectedReqLen, len(reqs))

			for _, req := range reqs {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, adapter.endpoint, req.Uri)
				assert.Equal(t, "application/json;charset=utf-8", req.Headers.Get("Content-Type"))
				assert.Equal(t, "application/json", req.Headers.Get("Accept"))
				assert.Equal(t, "2.5", req.Headers.Get("X-Openrtb-Version"))
			}
		})
	}
}

func TestMakeBids(t *testing.T) {
	testCases := []struct {
		name           string
		response       *adapters.ResponseData
		expectedErrors bool
		expectedBids   int
	}{
		{
			name:         "Valid response with bids",
			response:     &adapters.ResponseData{StatusCode: http.StatusOK, Body: json.RawMessage(`{"seatbid":[{"bid":[{"id":"1","impid":"123","price":1.23,"ext":{"prebid":{"type":"video"}}}]}]}`)},
			expectedBids: 1,
		},
		{
			name:           "No content response",
			response:       &adapters.ResponseData{StatusCode: http.StatusNoContent},
			expectedBids:   0,
			expectedErrors: false,
		},
		{
			name:           "Server error response",
			response:       &adapters.ResponseData{StatusCode: http.StatusInternalServerError},
			expectedErrors: true,
		},
		{
			name:           "Invalid JSON response",
			response:       &adapters.ResponseData{StatusCode: http.StatusOK, Body: json.RawMessage(`invalid`)},
			expectedErrors: true,
		},
		{
			name:         "Correct bid type detection",
			response:     &adapters.ResponseData{StatusCode: http.StatusOK, Body: json.RawMessage(`{"seatbid":[{"bid":[{"id":"1","impid":"bannerImp","price":1.23,"ext":{"prebid":{"type":"banner"}}},{"id":"2","impid":"videoImp","price":2.34,"ext":{"prebid":{"type":"video"}}},{"id":"3","impid":"nativeImp","price":3.45,"ext":{"prebid":{"type":"native"}}}]}]}`)},
			expectedBids: 3,
		},
		{
			name:           "Invalid or missing bid media type",
			response:       &adapters.ResponseData{StatusCode: http.StatusOK, Body: json.RawMessage(`{"seatbid":[{"bid":[{"id":"1","impid":"noExt","price":1.23},{"id":"2","impid":"invalidType","price":2.34,"ext":{"prebid":{"type":"invalid"}}},{"id":"3","impid":"nativeImp","price":3.45,"ext":{"prebid":{"type":"audio"}}}]}]}`)},
			expectedErrors: true,
		},
	}

	adapter := &adapter{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bidderResponse, errs := adapter.MakeBids(nil, &adapters.RequestData{}, tc.response)

			if tc.expectedErrors {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}

			if tc.expectedBids > 0 {
				assert.Equal(t, tc.expectedBids, len(bidderResponse.Bids))
				for _, bid := range bidderResponse.Bids {
					switch bid.Bid.ImpID {
					case "bannerImp":
						assert.Equal(t, openrtb_ext.BidTypeBanner, bid.BidType)
					case "videoImp":
						assert.Equal(t, openrtb_ext.BidTypeVideo, bid.BidType)
					case "nativeImp":
						assert.Equal(t, openrtb_ext.BidTypeNative, bid.BidType)
					}
				}
			} else if bidderResponse != nil {
				assert.Empty(t, bidderResponse.Bids)
			}
		})
	}
}

func TestValidateRequest(t *testing.T) {
	testCases := []struct {
		name        string
		request     openrtb2.BidRequest
		expectError bool
	}{
		{
			name: "Valid with site ID",
			request: openrtb2.BidRequest{
				Site: &openrtb2.Site{ID: "123"},
			},
			expectError: false,
		},
		{
			name: "Valid with app ID",
			request: openrtb2.BidRequest{
				App: &openrtb2.App{ID: "456"},
			},
			expectError: false,
		},
		{
			name:        "Invalid missing both site and app",
			request:     openrtb2.BidRequest{},
			expectError: true,
		},
		{
			name: "Invalid currency",
			request: openrtb2.BidRequest{
				Site: &openrtb2.Site{ID: "123"},
				Cur:  []string{"EUR"},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRequest(tc.request)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetValidImpressions(t *testing.T) {
	testCases := []struct {
		name          string
		impressions   []openrtb2.Imp
		expectedCount int
		expectError   bool
	}{
		{
			name: "Valid banner impression",
			impressions: []openrtb2.Imp{
				{Banner: &openrtb2.Banner{}},
			},
			expectedCount: 1,
		},
		{
			name: "Valid video impression",
			impressions: []openrtb2.Imp{
				{Video: &openrtb2.Video{}},
			},
			expectedCount: 1,
		},
		{
			name: "Valid native impression",
			impressions: []openrtb2.Imp{
				{Native: &openrtb2.Native{}},
			},
			expectedCount: 1,
		},
		{
			name: "Invalid audio impression",
			impressions: []openrtb2.Imp{
				{Audio: &openrtb2.Audio{}},
			},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "Mixed valid and invalid impressions",
			impressions: []openrtb2.Imp{
				{Banner: &openrtb2.Banner{}},
				{Audio: &openrtb2.Audio{}},
				{Video: &openrtb2.Video{}},
			},
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validImps, err := getValidImpressions(tc.impressions)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedCount, len(validImps))
		})
	}
}
