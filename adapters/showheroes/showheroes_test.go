package showheroes

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

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
							Ext:    json.RawMessage(`{"bidder":{"unitId":"12345678"}}`),
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
					Site: &openrtb2.Site{},
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{"unitId":"12345678"}}`),
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
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{},
							Ext:    json.RawMessage(`{"bidder":{"unitId":"12345678"}}`),
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
							Ext:    json.RawMessage(`{"bidder":{"unitId":"12345678"}}`),
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
							Ext:   json.RawMessage(`{"bidder":{"unitId":"12345678"}}`),
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
							Ext:         json.RawMessage(`{"bidder":{"unitId":"12345678"}}`),
						},
					},
				},
				reqInfo: &adapters.ExtraRequestInfo{
					CurrencyConversions: mockCurrencyConversion{
						rate: 0.85,
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
	bidder, err := Builder(openrtb_ext.BidderShowheroes, config.Adapter{
		Endpoint: "https://bid.showheroes.com/api/v1/bid",
	}, config.Server{})
	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	adapterstest.RunJSONBidderTest(t, "showheroestest", bidder)
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
