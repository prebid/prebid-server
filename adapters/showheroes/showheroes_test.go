package showheroes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

// Mock currency conversion
type mockCurrencyConversion struct {
	rate float64
}

func (m mockCurrencyConversion) GetRate(from, to string) (float64, error) {
	if from == "USD" && to == "EUR" {
		return m.rate, nil
	}
	return 1.0, nil
}

func (m mockCurrencyConversion) GetRates() *map[string]map[string]float64 {
	conversions := map[string]map[string]float64{
		"USD": {
			"EUR": m.rate,
		},
	}
	return &conversions
}

func TestGetBidType(t *testing.T) {
	type args struct {
		markupType openrtb2.MarkupType
	}
	tests := []struct {
		name              string
		args              args
		expectedBidTypeId openrtb_ext.BidType
		wantErr           bool
	}{
		{
			name: "banner",
			args: args{
				markupType: openrtb2.MarkupBanner,
			},
			expectedBidTypeId: openrtb_ext.BidTypeBanner,
			wantErr:           false,
		},
		{
			name: "video",
			args: args{
				markupType: openrtb2.MarkupVideo,
			},
			expectedBidTypeId: openrtb_ext.BidTypeVideo,
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidType := getBidType(tt.args.markupType)
			assert.Equal(t, tt.expectedBidTypeId, bidType)
		})
	}
}

func TestShowheroesAdapter_MakeRequests(t *testing.T) {
	type args struct {
		request *openrtb2.BidRequest
		reqInfo *adapters.ExtraRequestInfo
	}
	tests := []struct {
		name            string
		args            args
		expectedReqData []*adapters.RequestData
		wantErr         bool
	}{
		{
			name: "no_site_no_app",
			args: args{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: true,
		},
		{
			name: "site_without_page",
			args: args{
				request: &openrtb2.BidRequest{
					Site: &openrtb2.Site{}, // Missing Page URL
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: true,
		},
		{
			name: "app_without_bundle",
			args: args{
				request: &openrtb2.BidRequest{
					App: &openrtb2.App{}, // Missing Bundle ID
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: true,
		},
		{
			name: "invalid_bidder_params",
			args: args{
				request: &openrtb2.BidRequest{
					Site: &openrtb2.Site{Page: "http://example.com"},
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{}}`), // Missing unitId
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: true,
		},
		{
			name: "no_banner_or_video",
			args: args{
				request: &openrtb2.BidRequest{
					Site: &openrtb2.Site{Page: "http://example.com"},
					Imp: []openrtb2.Imp{
						{
							// No Banner or Video
							Ext: json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: true,
		},
		{
			name: "valid_request_with_site",
			args: args{
				request: &openrtb2.BidRequest{
					Site: &openrtb2.Site{Page: "http://example.com"},
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: false,
		},
		{
			name: "valid_request_with_app",
			args: args{
				request: &openrtb2.BidRequest{
					App: &openrtb2.App{Bundle: "com.example.app"},
					Imp: []openrtb2.Imp{
						{
							Video: &openrtb2.Video{},
							Ext:   json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
					Ext: json.RawMessage(`{"prebid":{"channel":{"name":"prebidjs","version":"1.0"}}}`),
				},
				reqInfo: &adapters.ExtraRequestInfo{},
			},
			wantErr: false,
		},
		{
			name: "currency_conversion",
			args: args{
				request: &openrtb2.BidRequest{
					Site: &openrtb2.Site{Page: "http://example.com"},
					Imp: []openrtb2.Imp{
						{
							Banner:      &openrtb2.Banner{},
							BidFloor:    1.0,
							BidFloorCur: "USD",
							Ext:         json.RawMessage(`{"bidder":{"unitId":"123456"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{
					CurrencyConversions: mockCurrencyConversion{
						rate: 0.85, // USD to EUR
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, buildErr := Builder(openrtb_ext.BidderShowheroes, config.Adapter{
				Endpoint: `https://bid.showheroes.com/api/v1/bid`,
			}, config.Server{})
			assert.Nil(t, buildErr)

			gotReqData, gotErr := a.MakeRequests(tt.args.request, tt.args.reqInfo)
			assert.Equal(t, tt.wantErr, len(gotErr) != 0)
			if tt.wantErr == false {
				assert.NotNil(t, gotReqData)
				assert.Equal(t, "https://bid.showheroes.com/api/v1/bid", gotReqData[0].Uri)
				assert.Equal(t, "POST", gotReqData[0].Method)

				var processedRequest openrtb2.BidRequest
				err := json.Unmarshal(gotReqData[0].Body, &processedRequest)
				assert.NoError(t, err)

				// Check if imps have been properly processed
				if len(processedRequest.Imp) > 0 {
					if processedRequest.Imp[0].BidFloorCur != "" {
						assert.Equal(t, "EUR", processedRequest.Imp[0].BidFloorCur)
					}
				}
			}
		})
	}
}

func TestJsonSamples(t *testing.T) {
	_, err := Builder(openrtb_ext.BidderShowheroes, config.Adapter{
		Endpoint: "https://bid.showheroes.com/api/v1/bid",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	bidder, _ := Builder(openrtb_ext.BidderShowheroes, config.Adapter{
		Endpoint: "https://bid.showheroes.com/api/v1/bid",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	adapterstest.RunJSONBidderTest(t, "showheroes", bidder)
}

func TestShowheroesAdapter_MakeBids(t *testing.T) {
	type args struct {
		internalRequest *openrtb2.BidRequest
		externalRequest *adapters.RequestData
		response        *adapters.ResponseData
	}
	tests := []struct {
		name     string
		args     args
		wantErr  []error
		wantResp *adapters.BidderResponse
	}{
		{
			name: "happy_path_banner",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "showheroes", "bid":[{"mtype": 1, "id": "bid-id-1", "impid": "test-imp-id", "price": 0.50, "adid": "ad-id-1", "adm": "some-banner-ad", "adomain":["showheroes.com"], "crid": "creative-id-1", "h": 250, "w": 300, "dealid": "deal-id-1"}]}], "bidid": "bid-request-1", "cur": "EUR"}`),
				},
			},
			wantErr: nil,
			wantResp: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							ID:      "bid-id-1",
							ImpID:   "test-imp-id",
							Price:   0.50,
							AdID:    "ad-id-1",
							AdM:     "some-banner-ad",
							ADomain: []string{"showheroes.com"},
							CrID:    "creative-id-1",
							H:       250,
							W:       300,
							DealID:  "deal-id-1",
							MType:   openrtb2.MarkupBanner,
						},
						BidType: openrtb_ext.BidTypeBanner,
					},
				},
				Currency: "EUR",
			},
		},
		{
			name: "happy_path_video",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "showheroes", "bid":[{"mtype": 2, "id": "bid-id-2", "impid": "test-imp-id", "price": 1.50, "adid": "ad-id-2", "adm": "some-video-ad", "adomain":["showheroes.com"], "crid": "creative-id-2", "h": 480, "w": 640}]}], "bidid": "bid-request-2", "cur": "USD"}`),
				},
			},
			wantErr: nil,
			wantResp: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							ID:      "bid-id-2",
							ImpID:   "test-imp-id",
							Price:   1.50,
							AdID:    "ad-id-2",
							AdM:     "some-video-ad",
							ADomain: []string{"showheroes.com"},
							CrID:    "creative-id-2",
							H:       480,
							W:       640,
							MType:   openrtb2.MarkupVideo,
						},
						BidType: openrtb_ext.BidTypeVideo,
					},
				},
				Currency: "USD",
			},
		},
		{
			name: "no_content_response",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusNoContent,
					Body:       nil,
				},
			},
			wantErr:  nil,
			wantResp: nil,
		},
		{
			name: "invalid_status_code",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusBadRequest,
					Body:       []byte(`Bad Request`),
				},
			},
			wantErr:  []error{fmt.Errorf("unexpected status code: 400")},
			wantResp: nil,
		},
		{
			name: "invalid_json_response",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "showheroes", "bid":[{]}`),
				},
			},
			wantErr:  []error{&errortypes.FailedToUnmarshal{}},
			wantResp: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, buildErr := Builder(openrtb_ext.BidderShowheroes, config.Adapter{
				Endpoint: "https://bid.showheroes.com/api/v1/bid",
			}, config.Server{})
			assert.Nil(t, buildErr)

			gotResp, gotErr := a.MakeBids(tt.args.internalRequest, tt.args.externalRequest, tt.args.response)

			if tt.wantErr == nil {
				assert.Equal(t, tt.wantErr, gotErr)
			} else if len(tt.wantErr) > 0 {
				assert.NotNil(t, gotErr)
				if len(gotErr) > 0 && len(tt.wantErr) > 0 {
					assert.IsType(t, tt.wantErr[0], gotErr[0])
				}
			}

			if tt.wantResp != nil {
				assert.Equal(t, tt.wantResp.Currency, gotResp.Currency)
				assert.Equal(t, len(tt.wantResp.Bids), len(gotResp.Bids))
				if len(tt.wantResp.Bids) > 0 {
					assert.Equal(t, tt.wantResp.Bids[0].BidType, gotResp.Bids[0].BidType)
					assert.Equal(t, tt.wantResp.Bids[0].Bid.ID, gotResp.Bids[0].Bid.ID)
					assert.Equal(t, tt.wantResp.Bids[0].Bid.MType, gotResp.Bids[0].Bid.MType)
				}
			} else if tt.wantResp == nil && tt.args.response.StatusCode == http.StatusNoContent {
				assert.Nil(t, gotResp)
			}
		})
	}
}

func TestPbsSourceExt(t *testing.T) {
	request := &openrtb2.BidRequest{}
	setPBSVersion(request, "test_version")

	source := request.Source
	assert.NotNil(t, source)
	assert.NotNil(t, source.Ext)

	var sourceExtMap map[string]map[string]string
	if err := jsonutil.Unmarshal(source.Ext, &sourceExtMap); err != nil {
		t.Fatalf("failed to unmarshal source.ext: %v", err)
	}

	assert.Equal(t, "test_version", sourceExtMap["pbs"]["pbsv"])
	assert.Equal(t, "go", sourceExtMap["pbs"]["pbsp"])
}
