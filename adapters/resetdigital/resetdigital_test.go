package resetdigital

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "resetdigitaltest", bidder)
}

func TestValidateAndFilterCurrencies(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{"USD"},
		},
		{
			name:     "nil array",
			input:    nil,
			expected: []string{"USD"},
		},
		{
			name:     "single empty string",
			input:    []string{""},
			expected: []string{"USD"},
		},
		{
			name:     "multiple empty strings",
			input:    []string{"", ""},
			expected: []string{"USD"},
		},
		{
			name:     "whitespace only strings",
			input:    []string{"   ", "\t", "\n"},
			expected: []string{"USD"},
		},
		{
			name:     "single valid currency",
			input:    []string{"EUR"},
			expected: []string{"EUR"},
		},
		{
			name:     "multiple valid currencies",
			input:    []string{"USD", "EUR", "GBP"},
			expected: []string{"USD", "EUR", "GBP"},
		},
		{
			name:     "single invalid currency",
			input:    []string{"INVALID"},
			expected: []string{"USD"},
		},
		{
			name:     "mixed valid and invalid currencies",
			input:    []string{"USD", "INVALID", "EUR"},
			expected: []string{"USD", "EUR"},
		},
		{
			name:     "currencies with whitespace",
			input:    []string{" USD ", "\tEUR\t", " GBP\n"},
			expected: []string{"USD", "EUR", "GBP"},
		},
		{
			name:     "lowercase currencies",
			input:    []string{"usd", "eur"},
			expected: []string{"USD", "EUR"},
		},
		{
			name:     "mixed case currencies",
			input:    []string{"Usd", "EUR", "gbp"},
			expected: []string{"USD", "EUR", "GBP"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAndFilterCurrencies(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMediaType(t *testing.T) {
	tests := []struct {
		name     string
		imp      openrtb2.Imp
		expected openrtb_ext.BidType
	}{
		{
			name: "Banner type",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
			},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name: "Video type",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{},
			},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name: "Audio type",
			imp: openrtb2.Imp{
				Audio: &openrtb2.Audio{},
			},
			expected: openrtb_ext.BidTypeAudio,
		},
		{
			name: "Native type",
			imp: openrtb2.Imp{
				Native: &openrtb2.Native{},
			},
			expected: openrtb_ext.BidTypeNative,
		},
		{
			name: "Multiple media types - prioritize video",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name: "Multiple media types - prioritize audio over banner",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
				Audio:  &openrtb2.Audio{},
			},
			expected: openrtb_ext.BidTypeAudio,
		},
		{
			name: "Multiple media types - prioritize native over banner",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
				Native: &openrtb2.Native{},
			},
			expected: openrtb_ext.BidTypeNative,
		},
		{
			name: "No media type defaults to banner",
			imp:  openrtb2.Imp{},
			expected: openrtb_ext.BidTypeBanner,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := getMediaType(test.imp)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestGetBidType(t *testing.T) {
	tests := []struct {
		name     string
		bid      openrtb2.Bid
		request  *openrtb2.BidRequest
		expected openrtb_ext.BidType
		hasError bool
	}{
		{
			name: "Banner MType",
			bid: openrtb2.Bid{
				MType: openrtb2.MarkupBanner,
			},
			request:  &openrtb2.BidRequest{},
			expected: openrtb_ext.BidTypeBanner,
			hasError: false,
		},
		{
			name: "Video MType",
			bid: openrtb2.Bid{
				MType: openrtb2.MarkupVideo,
			},
			request:  &openrtb2.BidRequest{},
			expected: openrtb_ext.BidTypeVideo,
			hasError: false,
		},
		{
			name: "Audio MType",
			bid: openrtb2.Bid{
				MType: openrtb2.MarkupAudio,
			},
			request:  &openrtb2.BidRequest{},
			expected: openrtb_ext.BidTypeAudio,
			hasError: false,
		},
		{
			name: "Native MType",
			bid: openrtb2.Bid{
				MType: openrtb2.MarkupNative,
			},
			request:  &openrtb2.BidRequest{},
			expected: openrtb_ext.BidTypeNative,
			hasError: false,
		},
		{
			name: "No MType, lookup impression - banner",
			bid: openrtb2.Bid{
				ImpID: "imp-1",
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			expected: openrtb_ext.BidTypeBanner,
			hasError: false,
		},
		{
			name: "No MType, lookup impression - video",
			bid: openrtb2.Bid{
				ImpID: "imp-1",
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp-1",
						Video: &openrtb2.Video{},
					},
				},
			},
			expected: openrtb_ext.BidTypeVideo,
			hasError: false,
		},
		{
			name: "No matching impression",
			bid: openrtb2.Bid{
				ImpID: "imp-not-found",
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			expected: "",
			hasError: true,
		},
		{
			name: "Unknown MType falls back to impression lookup",
			bid: openrtb2.Bid{
				ImpID: "imp-1",
				MType: openrtb2.MarkupType(99),
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			expected: openrtb_ext.BidTypeBanner,
			hasError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidType, err := getBidType(test.bid, test.request)

			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, bidType)
			}
		})
	}
}

func TestMakeBidsErrorCases(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name         string
		responseData *adapters.ResponseData
		expectNil    bool
		expectErrs   int
		errType      interface{}
	}{
		{
			name: "Status 204 No Content",
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusNoContent,
			},
			expectNil:  true,
			expectErrs: 0,
		},
		{
			name: "Status 400 Bad Request",
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusBadRequest,
				Body:       []byte(`{"error": "Bad Request"}`),
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
		},
		{
			name: "Status 500 Internal Server Error",
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusInternalServerError,
				Body:       []byte(`{"error": "Internal Server Error"}`),
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadServerResponse{},
		},
		{
			name: "Invalid JSON response",
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{invalid json`),
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadServerResponse{},
		},
		{
			name: "Empty seatbid array",
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"id": "test", "seatbid": []}`),
			},
			expectNil:  true,
			expectErrs: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{}
			bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, test.responseData)

			if test.expectNil {
				assert.Nil(t, bidResponse)
			} else {
				assert.NotNil(t, bidResponse)
			}

			assert.Len(t, errs, test.expectErrs)

			if test.expectErrs > 0 && test.errType != nil {
				assert.IsType(t, test.errType, errs[0])
			}
		})
	}
}

func TestMakeRequestsErrorCases(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name       string
		request    *openrtb2.BidRequest
		expectNil  bool
		expectErrs int
		errType    interface{}
	}{
		{
			name: "Multiple impressions",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Ext: json.RawMessage(`{"bidder": {"placement_id": "test"}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"bidder": {"placement_id": "test"}}`)},
				},
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
		},
		{
			name: "Malformed imp.ext",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{invalid json`),
					},
				},
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
		},
		{
			name: "Malformed bidder params",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {invalid json}`),
					},
				},
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reqs, errs := bidder.MakeRequests(test.request, &adapters.ExtraRequestInfo{})

			if test.expectNil {
				assert.Nil(t, reqs)
			} else {
				assert.NotNil(t, reqs)
			}

			assert.Len(t, errs, test.expectErrs)

			if test.expectErrs > 0 && test.errType != nil {
				assert.IsType(t, test.errType, errs[0])
			}
		})
	}
}

func TestParseBidResponseEdgeCases(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name         string
		request      *openrtb2.BidRequest
		responseData *adapters.ResponseData
		expectBids   int
		expectErrs   int
	}{
		{
			name: "Bid with zero price filtered out",
			request: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{ID: "test-imp-id", Banner: &openrtb2.Banner{}},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"id": "test-request-id",
					"seatbid": [
						{
							"bid": [
								{
									"id": "test-bid-id",
									"impid": "test-imp-id",
									"price": 0.0,
									"adm": "<div>test ad</div>"
								}
							]
						}
					],
					"cur": "USD"
				}`),
			},
			expectBids: 0,
			expectErrs: 1,
		},
		{
			name: "Bid with negative price filtered out",
			request: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{ID: "test-imp-id", Banner: &openrtb2.Banner{}},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"id": "test-request-id",
					"seatbid": [
						{
							"bid": [
								{
									"id": "test-bid-id",
									"impid": "test-imp-id",
									"price": -1.5,
									"adm": "<div>test ad</div>"
								}
							]
						}
					],
					"cur": "USD"
				}`),
			},
			expectBids: 0,
			expectErrs: 1,
		},
		{
			name: "Currency fallback to request currency",
			request: &openrtb2.BidRequest{
				ID:  "test-request-id",
				Cur: []string{"EUR"},
				Imp: []openrtb2.Imp{
					{ID: "test-imp-id", Banner: &openrtb2.Banner{}},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"id": "test-request-id",
					"seatbid": [
						{
							"bid": [
								{
									"id": "test-bid-id",
									"impid": "test-imp-id",
									"price": 1.5,
									"adm": "<div>test ad</div>"
								}
							]
						}
					]
				}`),
			},
			expectBids: 1,
			expectErrs: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidResponse, errs := bidder.MakeBids(test.request, &adapters.RequestData{}, test.responseData)

			if test.expectBids > 0 {
				assert.NotNil(t, bidResponse)
				assert.Len(t, bidResponse.Bids, test.expectBids)
			}

			assert.Len(t, errs, test.expectErrs)
		})
	}
}

func TestSchainPassthrough(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		expectSchain   bool
		expectedSchain *openrtb2.SupplyChain
	}{
		{
			name: "Request with schain should pass it through",
			request: &openrtb2.BidRequest{
				ID: "test-request-id",
				Source: &openrtb2.Source{
					TID: "test-transaction-id",
					SChain: &openrtb2.SupplyChain{
						Complete: 1,
						Ver:      "1.0",
						Nodes: []openrtb2.SupplyChainNode{
							{
								ASI: "example.com",
								SID: "12345",
								HP:  openrtb2.Int8Ptr(1),
								RID: "request-id-123",
							},
						},
					},
				},
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
					},
				},
			},
			expectSchain: true,
			expectedSchain: &openrtb2.SupplyChain{
				Complete: 1,
				Ver:      "1.0",
				Nodes: []openrtb2.SupplyChainNode{
					{
						ASI: "example.com",
						SID: "12345",
						HP:  openrtb2.Int8Ptr(1),
						RID: "request-id-123",
					},
				},
			},
		},
		{
			name: "Request without schain should not include it",
			request: &openrtb2.BidRequest{
				ID: "test-request-id",
				Source: &openrtb2.Source{
					TID: "test-transaction-id",
				},
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
					},
				},
			},
			expectSchain:   false,
			expectedSchain: nil,
		},
		{
			name: "Request without source should not include schain",
			request: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
					},
				},
			},
			expectSchain:   false,
			expectedSchain: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reqs, errs := bidder.MakeRequests(test.request, &adapters.ExtraRequestInfo{})

			assert.Empty(t, errs)
			assert.Len(t, reqs, 1)

			var sentRequest openrtb2.BidRequest
			err := json.Unmarshal(reqs[0].Body, &sentRequest)
			assert.NoError(t, err)

			if test.expectSchain {
				assert.NotNil(t, sentRequest.Source)
				assert.NotNil(t, sentRequest.Source.SChain)
				assert.Equal(t, test.expectedSchain.Complete, sentRequest.Source.SChain.Complete)
				assert.Equal(t, test.expectedSchain.Ver, sentRequest.Source.SChain.Ver)
				assert.Len(t, sentRequest.Source.SChain.Nodes, len(test.expectedSchain.Nodes))
				
				if len(sentRequest.Source.SChain.Nodes) > 0 {
					assert.Equal(t, test.expectedSchain.Nodes[0].ASI, sentRequest.Source.SChain.Nodes[0].ASI)
					assert.Equal(t, test.expectedSchain.Nodes[0].SID, sentRequest.Source.SChain.Nodes[0].SID)
					assert.Equal(t, test.expectedSchain.Nodes[0].HP, sentRequest.Source.SChain.Nodes[0].HP)
					assert.Equal(t, test.expectedSchain.Nodes[0].RID, sentRequest.Source.SChain.Nodes[0].RID)
				}
			} else {
				if sentRequest.Source != nil {
					assert.Nil(t, sentRequest.Source.SChain)
				}
			}
		})
	}
}

func TestSourceFieldsPassthrough(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Source: &openrtb2.Source{
			TID: "test-transaction-id",
			FD:  openrtb2.Int8Ptr(1),
			Ext: json.RawMessage(`{"custom": "data"}`),
		},
		Imp: []openrtb2.Imp{
			{
				ID:  "test-imp-id",
				Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
			},
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, reqs, 1)

	var sentRequest openrtb2.BidRequest
	err := json.Unmarshal(reqs[0].Body, &sentRequest)
	assert.NoError(t, err)

	assert.NotNil(t, sentRequest.Source)
	assert.Equal(t, "test-transaction-id", sentRequest.Source.TID)
	assert.Equal(t, openrtb2.Int8Ptr(1), sentRequest.Source.FD)
	
	var expectedExt, actualExt map[string]interface{}
	err = json.Unmarshal([]byte(`{"custom": "data"}`), &expectedExt)
	assert.NoError(t, err)
	err = json.Unmarshal(sentRequest.Source.Ext, &actualExt)
	assert.NoError(t, err)
	assert.Equal(t, expectedExt, actualExt)
}
