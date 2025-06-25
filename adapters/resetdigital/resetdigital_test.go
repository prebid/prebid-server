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

	adapterstest.RunJSONBidderTest(t, "testdata", bidder)
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

func TestBuildRequest(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp := openrtb2.Imp{
		ID: "test-imp-id",
		Banner: &openrtb2.Banner{
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		},
		Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
	}

	request := &openrtb2.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Page:   "https://example.com/page",
			Domain: "example.com",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, reqs, 1)
	assert.Equal(t, "https://example.com?pid=test-placement", reqs[0].Uri)
	assert.Equal(t, http.MethodPost, reqs[0].Method)
	assert.NotEmpty(t, reqs[0].Body)
	assert.Equal(t, "application/json", reqs[0].Headers.Get("Content-Type"))
}

func TestMakeRequestsEdgeCases(t *testing.T) {
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
		errMsg     string
	}{
		{
			name: "Malformed imp.ext",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{`),
					},
				},
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
			errMsg:     "Error parsing bidderExt",
		},
		{
			name: "Empty placement_id",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"placement_id": ""}}`),
					},
				},
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
			errMsg:     "Missing required parameter 'placement_id'",
		},
		{
			name: "Malformed bidder params",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {`),
					},
				},
			},
			expectNil:  true,
			expectErrs: 1,
			errType:    &errortypes.BadInput{},
			errMsg:     "Error parsing",
		},
		{
			name: "Empty currency",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
					},
				},
				Cur: []string{""},
			},
			expectNil:  false,
			expectErrs: 0,
		},
		{
			name: "TagID not overwritten when present",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "test-imp-id",
						TagID: "existing-tag-id",
						Ext:   json.RawMessage(`{"bidder": {"placement_id": "test-placement"}}`),
					},
				},
			},
			expectNil:  false,
			expectErrs: 0,
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

			if test.expectErrs > 0 {
				assert.IsType(t, test.errType, errs[0])
				assert.Contains(t, errs[0].Error(), test.errMsg)
			}

			if test.name == "Empty currency" {
				reqBody := reqs[0].Body
				var reqParsed openrtb2.BidRequest
				err := json.Unmarshal(reqBody, &reqParsed)
				assert.NoError(t, err)
				assert.Equal(t, []string{"USD"}, reqParsed.Cur)
			}

			if test.name == "TagID not overwritten when present" {
				reqBody := reqs[0].Body
				var reqParsed openrtb2.BidRequest
				err := json.Unmarshal(reqBody, &reqParsed)
				assert.NoError(t, err)
				assert.Equal(t, "existing-tag-id", reqParsed.Imp[0].TagID)
			}
		})
	}
}

func TestMultipleImpressions(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID: "test-imp-id1",
		Banner: &openrtb2.Banner{
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		},
		Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement-1"}}`),
	}

	imp2 := openrtb2.Imp{
		ID: "test-imp-id2",
		Banner: &openrtb2.Banner{
			Format: []openrtb2.Format{
				{W: 728, H: 90},
			},
		},
		Ext: json.RawMessage(`{"bidder": {"placement_id": "test-placement-2"}}`),
	}

	request := &openrtb2.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb2.Imp{imp1, imp2},
		Site: &openrtb2.Site{
			Page:   "https://example.com/page",
			Domain: "example.com",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Nil(t, reqs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "ResetDigital adapter supports only one impression per request")

	_, ok := errs[0].(*errortypes.BadInput)
	assert.True(t, ok, "Error should be of type BadInput")
}

func TestParseBidResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
					},
				},
			},
		},
	}

	requestData := &adapters.RequestData{}

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`{
			"id": "test-request-id",
			"seatbid": [
				{
					"bid": [
						{
							"id": "test-bid-id",
							"impid": "test-imp-id",
							"price": 3.14,
							"adm": "<div>test ad</div>",
							"crid": "test-creative",
							"w": 300,
							"h": 250
						}
					],
					"seat": "resetdigital"
				}
			],
			"cur": "USD"
		}`),
	}

	bidResponse, errs := bidder.MakeBids(request, requestData, responseData)

	assert.Empty(t, errs)
	assert.NotNil(t, bidResponse)
	assert.Equal(t, "USD", bidResponse.Currency)
	assert.Len(t, bidResponse.Bids, 1)
	assert.Equal(t, "test-bid-id", bidResponse.Bids[0].Bid.ID)
	assert.Equal(t, float64(3.14), bidResponse.Bids[0].Bid.Price)
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType)
}

func TestMakeBidsStatus204(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Nil(t, bidResponse)
	assert.Nil(t, errs)
}

func TestMakeBidsStatus500(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusInternalServerError,
		Body:       []byte(`{"error": "Internal Server Error"}`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Nil(t, bidResponse)
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Unexpected status code")

	_, ok := errs[0].(*errortypes.BadServerResponse)
	assert.True(t, ok, "Error should be of type BadServerResponse")
}

func TestMakeBidsStatus400(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusBadRequest,
		Body:       []byte(`{"error": "Bad Request"}`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Nil(t, bidResponse)
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Unexpected status code")

	_, ok := errs[0].(*errortypes.BadInput)
	assert.True(t, ok, "Error should be of type BadInput")
}

func TestMakeBidsEmptySeatBid(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id": "test-request-id", "seatbid": []}`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Nil(t, bidResponse)
	assert.Nil(t, errs)
}

func TestMakeBidsInvalidJson(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Nil(t, bidResponse)
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Bad server response")

	_, ok := errs[0].(*errortypes.BadServerResponse)
	assert.True(t, ok, "Error should be of type BadServerResponse")
}

func TestMakeBidsStatusRedirect(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusFound,
		Body:       []byte(`{"id": "test-request-id"}`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Nil(t, bidResponse)
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Unexpected status code")

	_, ok := errs[0].(*errortypes.BadServerResponse)
	assert.True(t, ok, "Error should be of type BadServerResponse")
}

func TestBidPriceNegative(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-id",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	responseData := &adapters.ResponseData{
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
							"adm": "<div>test ad</div>",
							"crid": "test-creative",
							"w": 300,
							"h": 250
						}
					],
					"seat": "resetdigital"
				}
			],
			"cur": "USD"
		}`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Len(t, errs, 1)
	assert.IsType(t, &errortypes.Warning{}, errs[0])
	assert.Contains(t, errs[0].Error(), "price -1.500000 <= 0 filtered out")
	assert.NotNil(t, bidResponse)
	assert.Len(t, bidResponse.Bids, 0, "Bids with negative price should be filtered")
}

func TestGetBidTypeMultipleImps(t *testing.T) {
	bid := openrtb2.Bid{
		ImpID: "imp-2",
	}
	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			{
				ID:    "imp-2",
				Video: &openrtb2.Video{},
			},
		},
	}

	bidType, err := getBidType(bid, request)

	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeVideo, bidType)
}

func TestBidPriceZero(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://example.com",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-id",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	responseData := &adapters.ResponseData{
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
							"adm": "<div>test ad</div>",
							"crid": "test-creative",
							"w": 300,
							"h": 250
						}
					],
					"seat": "resetdigital"
				}
			],
			"cur": "USD"
		}`),
	}

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, responseData)

	assert.Len(t, errs, 1)
	assert.IsType(t, &errortypes.Warning{}, errs[0])
	assert.Contains(t, errs[0].Error(), "price 0.000000 <= 0 filtered out")
	assert.NotNil(t, bidResponse)
	assert.Len(t, bidResponse.Bids, 0, "Bids with price 0 should be filtered")
}
