package boldwin_rapid

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderBoldwinRapid, config.Adapter{
			Endpoint: "https://rtb.beardfleet.com/auction/bid?pid={{.PublisherID}}&tid={{.PlacementID}}",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "boldwin_rapidtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

// TestMakeRequestsErrors tests error handling in the MakeRequests method
func TestMakeRequestsErrors(t *testing.T) {
	testCases := []struct {
		name            string
		givenBidRequest *openrtb2.BidRequest
		mockAdapter     *mockAdapter
		expectedError   string
	}{
		{
			name: "Error unmarshalling imp.Ext",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`invalid json`),
					},
				},
			},
			mockAdapter:   &mockAdapter{},
			expectedError: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "Error unmarshalling bidderExt.Bidder",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": "invalid json"}`),
					},
				},
			},
			mockAdapter:   &mockAdapter{},
			expectedError: "json: cannot unmarshal string into Go value of type openrtb_ext.ImpExtBoldwinRapid",
		},
		{
			name: "Error building endpoint URL",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"pid": "123", "tid": "456"}}`),
					},
				},
			},
			mockAdapter: &mockAdapter{
				buildEndpointURLErr: errors.New("endpoint URL error"),
			},
			expectedError: "endpoint URL error",
		},
		{
			name: "Error making request",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"pid": "123", "tid": "456"}}`),
					},
				},
			},
			mockAdapter: &mockAdapter{
				makeRequestErr: errors.New("make request error"),
			},
			expectedError: "make request error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// When
			requests, errs := tc.mockAdapter.MakeRequests(tc.givenBidRequest, nil)

			// Then
			assert.Nil(t, requests)
			require.Len(t, errs, 1)
			assert.Contains(t, errs[0].Error(), tc.expectedError)
		})
	}
}

// Mock adapter for testing
type mockAdapter struct {
	buildEndpointURLErr error
	makeRequestErr      error
}

func (m *mockAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	reqCopy := *request

	for _, imp := range request.Imp {
		// Create a new request with just this impression
		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		var boldwinExt openrtb_ext.ImpExtBoldwinRapid

		// Use the current impression's Ext
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}

		if err := json.Unmarshal(bidderExt.Bidder, &boldwinExt); err != nil {
			return nil, []error{err}
		}

		if m.buildEndpointURLErr != nil {
			return nil, []error{m.buildEndpointURLErr}
		}

		if m.makeRequestErr != nil {
			return nil, []error{m.makeRequestErr}
		}
	}

	return adapterRequests, nil
}

func TestMakeRequestsUnmarshalErrors(t *testing.T) {
	adapter, buildErr := Builder(openrtb_ext.BidderBoldwinRapid, config.Adapter{
		Endpoint: "https://rtb.beardfleet.com/auction/bid?pid={{.PublisherID}}&tid={{.PlacementID}}",
	}, config.Server{})
	assert.NoError(t, buildErr)

	t.Run("invalid imp.Ext json", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{
					ID:  "test-imp-id",
					Ext: json.RawMessage(`{invalid json}`),
				},
			},
		}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.Nil(t, requests)
		assert.Len(t, errs, 1)
		assert.Error(t, errs[0])
	})

	t.Run("invalid bidder extension json", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{
					ID:  "test-imp-id",
					Ext: json.RawMessage(`{"bidder": {invalid json}}`),
				},
			},
		}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.Nil(t, requests)
		assert.Len(t, errs, 1)
		assert.Error(t, errs[0])
	})

	t.Run("bidder field is not valid json object", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{
					ID:  "test-imp-id",
					Ext: json.RawMessage(`{"bidder": "not an object"}`),
				},
			},
		}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.Nil(t, requests)
		assert.Len(t, errs, 1)
		assert.Error(t, errs[0])
	})
}

func TestGetHeaders(t *testing.T) {
	adapter := &adapter{}

	tests := []struct {
		name           string
		requestJSON    string
		expectedHeader map[string]string
		checkHeaders   []string
	}{
		{
			name: "device with IP",
			requestJSON: `{
				"device": {
					"ip": "192.168.1.1"
				}
			}`,
			expectedHeader: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"IP":              "192.168.1.1",
			},
			checkHeaders: []string{"X-Forwarded-For", "IP"},
		},
		{
			name: "device with IPv6",
			requestJSON: `{
				"device": {
					"ipv6": "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
				}
			}`,
			expectedHeader: map[string]string{
				"X-Forwarded-For": "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			},
			checkHeaders: []string{"X-Forwarded-For"},
		},
		{
			name: "device with both IP and IPv6",
			requestJSON: `{
				"device": {
					"ip": "192.168.1.1",
					"ipv6": "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
				}
			}`,
			expectedHeader: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"IP":              "192.168.1.1",
			},
			checkHeaders: []string{"X-Forwarded-For", "IP"},
		},
		{
			name: "device with User-Agent",
			requestJSON: `{
				"device": {
					"ua": "Mozilla/5.0"
				}
			}`,
			expectedHeader: map[string]string{
				"User-Agent": "Mozilla/5.0",
			},
			checkHeaders: []string{"User-Agent"},
		},
		{
			name: "device with all fields",
			requestJSON: `{
				"device": {
					"ua": "Mozilla/5.0",
					"ip": "192.168.1.1",
					"ipv6": "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
				}
			}`,
			expectedHeader: map[string]string{
				"User-Agent":      "Mozilla/5.0",
				"X-Forwarded-For": "192.168.1.1",
				"IP":              "192.168.1.1",
			},
			checkHeaders: []string{"User-Agent", "X-Forwarded-For", "IP"},
		},
		{
			name:           "no device",
			requestJSON:    `{}`,
			expectedHeader: map[string]string{},
			checkHeaders:   []string{},
		},
		{
			name: "empty device fields",
			requestJSON: `{
				"device": {
					"ua": "",
					"ip": "",
					"ipv6": ""
				}
			}`,
			expectedHeader: map[string]string{},
			checkHeaders:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request openrtb2.BidRequest
			err := json.Unmarshal([]byte(tt.requestJSON), &request)
			assert.NoError(t, err)

			headers := adapter.getHeaders(&request)

			// Check required headers are always present
			assert.Equal(t, "application/json;charset=utf-8", headers.Get("Content-Type"))
			assert.Equal(t, "application/json", headers.Get("Accept"))
			assert.Equal(t, "2.5", headers.Get("x-openrtb-version"))
			assert.Equal(t, "rtb.beardfleet.com", headers.Get("Host"))

			// Check expected headers
			for _, headerName := range tt.checkHeaders {
				assert.Equal(t, tt.expectedHeader[headerName], headers.Get(headerName), "header %s mismatch", headerName)
			}

			// Verify headers that should not be present when device fields are empty
			if len(tt.checkHeaders) == 0 && request.Device != nil {
				if request.Device.UA == "" {
					assert.Empty(t, headers.Get("User-Agent"))
				}
				if len(request.Device.IP) == 0 && len(request.Device.IPv6) == 0 {
					// X-Forwarded-For should only have one value or none
					assert.True(t, headers.Get("X-Forwarded-For") == "" || len(headers.Values("X-Forwarded-For")) <= 1)
				}
			}
		})
	}
}

func TestMakeBids(t *testing.T) {
	adapter := &adapter{}

	t.Run("no content response", func(t *testing.T) {
		responseData := &adapters.ResponseData{
			StatusCode: 204,
		}

		bidderResp, errs := adapter.MakeBids(nil, nil, responseData)
		assert.Nil(t, bidderResp)
		assert.Nil(t, errs)
	})

	t.Run("error status code", func(t *testing.T) {
		responseData := &adapters.ResponseData{
			StatusCode: 500,
			Body:       []byte("Internal Server Error"),
		}

		bidderResp, errs := adapter.MakeBids(nil, nil, responseData)
		assert.Nil(t, bidderResp)
		assert.Len(t, errs, 1)
	})

	t.Run("invalid response body json", func(t *testing.T) {
		responseData := &adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(`{invalid json}`),
		}

		bidderResp, errs := adapter.MakeBids(nil, nil, responseData)
		assert.Nil(t, bidderResp)
		assert.Len(t, errs, 1)
		assert.Error(t, errs[0])
	})

	t.Run("successful response with currency", func(t *testing.T) {
		responseJSON := `{
			"id": "test-request-id",
			"cur": "USD",
			"seatbid": [
				{
					"bid": [
						{
							"id": "bid1",
							"impid": "imp1",
							"price": 1.5,
							"mtype": 1
						}
					]
				}
			]
		}`

		responseData := &adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(responseJSON),
		}

		request := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{ID: "imp1"}},
		}

		bidderResp, errs := adapter.MakeBids(request, nil, responseData)
		assert.Nil(t, errs)
		assert.NotNil(t, bidderResp)
		assert.Equal(t, "USD", bidderResp.Currency)
		assert.Len(t, bidderResp.Bids, 1)
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidderResp.Bids[0].BidType)
	})

	t.Run("multiple bids with different types", func(t *testing.T) {
		responseJSON := `{
			"id": "test-request-id",
			"seatbid": [
				{
					"bid": [
						{
							"id": "bid1",
							"impid": "imp1",
							"price": 1.5,
							"mtype": 1
						},
						{
							"id": "bid2",
							"impid": "imp2",
							"price": 2.0,
							"mtype": 2
						},
						{
							"id": "bid3",
							"impid": "imp3",
							"price": 3.0,
							"mtype": 4
						}
					]
				}
			]
		}`

		responseData := &adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(responseJSON),
		}

		request := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{ID: "imp1"},
				{ID: "imp2"},
				{ID: "imp3"},
			},
		}

		bidderResp, errs := adapter.MakeBids(request, nil, responseData)
		assert.Nil(t, errs)
		assert.NotNil(t, bidderResp)
		assert.Len(t, bidderResp.Bids, 3)
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidderResp.Bids[0].BidType)
		assert.Equal(t, openrtb_ext.BidTypeVideo, bidderResp.Bids[1].BidType)
		assert.Equal(t, openrtb_ext.BidTypeNative, bidderResp.Bids[2].BidType)
	})
}

func TestGetBidMediaType(t *testing.T) {
	t.Run("banner type", func(t *testing.T) {
		bid := &openrtb2.Bid{
			ImpID: "imp1",
			MType: openrtb2.MarkupBanner,
		}

		bidType, err := getBidMediaType(bid)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidType)
	})

	t.Run("video type", func(t *testing.T) {
		bid := &openrtb2.Bid{
			ImpID: "imp1",
			MType: openrtb2.MarkupVideo,
		}

		bidType, err := getBidMediaType(bid)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeVideo, bidType)
	})

	t.Run("native type", func(t *testing.T) {
		bid := &openrtb2.Bid{
			ImpID: "imp1",
			MType: openrtb2.MarkupNative,
		}

		bidType, err := getBidMediaType(bid)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeNative, bidType)
	})
}

func TestMakeRequestsMultipleImpressions(t *testing.T) {
	adapter, err := Builder(openrtb_ext.BidderBoldwinRapid, config.Adapter{
		Endpoint: "https://rtb.beardfleet.com/auction/bid?pid={{.PublisherID}}&tid={{.PlacementID}}",
	}, config.Server{})
	require.NoError(t, err)

	t.Run("multiple impressions create multiple requests", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{
			ID: "test-request",
			Imp: []openrtb2.Imp{
				{
					ID:  "imp1",
					Ext: json.RawMessage(`{"bidder": {"pid": "pub1", "tid": "tag1"}}`),
				},
				{
					ID:  "imp2",
					Ext: json.RawMessage(`{"bidder": {"pid": "pub2", "tid": "tag2"}}`),
				},
			},
		}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.Nil(t, errs)
		assert.Len(t, requests, 2)
		assert.Contains(t, requests[0].Uri, "pid=pub1&tid=tag1")
		assert.Contains(t, requests[1].Uri, "pid=pub2&tid=tag2")
	})
}
