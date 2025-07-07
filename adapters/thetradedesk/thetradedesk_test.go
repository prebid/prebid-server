package thetradedesk

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"net/http"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTheTradeDesk, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestBadConfig(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTheTradeDesk, config.Adapter{
		Endpoint:         `http://it.doesnt.matter/bid`,
		ExtraAdapterInfo: "12365217635",
	},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestCorrectConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTheTradeDesk, config.Adapter{
		Endpoint:         `http://it.doesnt.matter/bid`,
		ExtraAdapterInfo: `abcde`,
	},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr)
	assert.NotNil(t, bidder)
}

func TestEmptyConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTheTradeDesk, config.Adapter{
		Endpoint:         `https://direct.adsrvr.org/bid/bidder/{{.SupplyId}}`,
		ExtraAdapterInfo: `ttd`,
	},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr)
	assert.NotNil(t, bidder)
}

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderTheTradeDesk,
		config.Adapter{Endpoint: "https://direct.adsrvr.org/bid/bidder/{{.SupplyId}}", ExtraAdapterInfo: "ttd"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "1"},
	)
	assert.Nil(t, err)
	adapterstest.RunJSONBidderTest(t, "thetradedesktest", bidder)
}

func TestGetBidType(t *testing.T) {
	type args struct {
		markupType openrtb2.MarkupType
	}
	tests := []struct {
		name              string
		args              args
		markupType        openrtb2.MarkupType
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
		{
			name: "native",
			args: args{
				markupType: openrtb2.MarkupNative,
			},
			expectedBidTypeId: openrtb_ext.BidTypeNative,
			wantErr:           false,
		},
		{
			name: "invalid",
			args: args{
				markupType: -1,
			},
			expectedBidTypeId: "",
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidType, err := getBidType(tt.args.markupType)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedBidTypeId, bidType)
		})
	}
}

func TestGetExtensionInfo(t *testing.T) {
	type args struct {
		impressions []openrtb2.Imp
	}
	tests := []struct {
		name                   string
		args                   args
		expectedPublisherId    string
		expectedSupplySourceId string
		wantErr                bool
	}{
		{
			name: "valid_publisher_Id",
			args: args{
				impressions: []openrtb2.Imp{
					{
						Video: &openrtb2.Video{},
						Ext:   json.RawMessage(`{"bidder":{"publisherId":"1", "supplySourceId": "abc"}}`),
					},
				},
			},
			expectedPublisherId:    "1",
			expectedSupplySourceId: "abc",
			wantErr:                false,
		},
		{
			name: "multiple_valid_publisher_Id",
			args: args{
				impressions: []openrtb2.Imp{
					{
						Video: &openrtb2.Video{},
						Ext:   json.RawMessage(`{"bidder":{"publisherId":"1", "supplySourceId": "abc"}}`),
					},
					{
						Video: &openrtb2.Video{},
						Ext:   json.RawMessage(`{"bidder":{"publisherId":"2",  "supplySourceId": "def"}}`),
					},
				},
			},
			expectedPublisherId:    "1",
			expectedSupplySourceId: "abc",
			wantErr:                false,
		},
		{
			name: "not_publisherId_present",
			args: args{
				impressions: []openrtb2.Imp{
					{
						Video: &openrtb2.Video{},
						Ext:   json.RawMessage(`{"bidder":{}}`),
					},
				},
			},
			expectedPublisherId:    "",
			expectedSupplySourceId: "",
			wantErr:                false,
		},
		{
			name: "nil_publisherId_present",
			args: args{
				impressions: []openrtb2.Imp{
					{
						Video: &openrtb2.Video{},
						Ext:   json.RawMessage(`{"bidder":{"publisherId":""}}`),
					},
				},
			},
			expectedPublisherId:    "",
			expectedSupplySourceId: "",
			wantErr:                false,
		},
		{
			name: "no_impressions",
			args: args{
				impressions: []openrtb2.Imp{},
			},
			expectedPublisherId:    "",
			expectedSupplySourceId: "",
			wantErr:                false,
		},
		{
			name: "invalid_bidder_object",
			args: args{
				impressions: []openrtb2.Imp{
					{
						Video: &openrtb2.Video{},
						Ext:   json.RawMessage(`{"bidder":{"doesnotexistprop":""}}`),
					},
				},
			},
			expectedPublisherId:    "",
			expectedSupplySourceId: "",
			wantErr:                false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisherId, supplySourceId, err := getExtensionInfo(tt.args.impressions)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedPublisherId, publisherId)
			assert.Equal(t, tt.expectedSupplySourceId, supplySourceId)
		})
	}
}

func TestTheTradeDeskAdapter_MakeRequests(t *testing.T) {
	type fields struct {
		URI string
	}
	type args struct {
		request *openrtb2.BidRequest
		reqInfo *adapters.ExtraRequestInfo
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		expectedReqData []*adapters.RequestData
		wantErr         bool
	}{
		{
			name: "invalid_bidderparams",
			args: args{
				request: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams":{:"123"}}}`)},
			},
			wantErr: true,
		},
		{
			name: "request_with_App",
			args: args{
				request: &openrtb2.BidRequest{
					App: &openrtb2.App{},
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":"123"}}}`),
				},
			},
			wantErr: false,
		},
		{
			name: "request_with_App_and_publisher",
			args: args{
				request: &openrtb2.BidRequest{
					App: &openrtb2.App{Publisher: &openrtb2.Publisher{}},
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":"123"}}}`),
				},
			},
			wantErr: false,
		},
		{
			name: "request_with_Site",
			args: args{
				request: &openrtb2.BidRequest{
					Site: &openrtb2.Site{},
					Ext:  json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":"123"}}}`),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, buildErr := Builder(openrtb_ext.BidderTheTradeDesk, config.Adapter{
				Endpoint:         `https://adsrvr.org/bid/bidder/{{.SupplyId}}`,
				ExtraAdapterInfo: "test",
			}, config.Server{})
			assert.Nil(t, buildErr)

			gotReqData, gotErr := a.MakeRequests(tt.args.request, tt.args.reqInfo)
			assert.Equal(t, tt.wantErr, len(gotErr) != 0)
			if tt.wantErr == false {
				assert.NotNil(t, gotReqData)
			}
		})
	}
}

func TestTheTradeDeskAdapter_MakeBids(t *testing.T) {
	type fields struct {
		URI string
	}
	type args struct {
		internalRequest *openrtb2.BidRequest
		externalRequest *adapters.RequestData
		response        *adapters.ResponseData
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  []error
		wantResp *adapters.BidderResponse
	}{
		{
			name: "happy_path_valid_response_with_all_bid_params",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"mtype": 1, "id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["ttd.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": 1}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
				},
			},
			wantErr: nil,
			wantResp: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							ID:      "7706636740145184841",
							ImpID:   "test-imp-id",
							Price:   0.500000,
							AdID:    "29681110",
							AdM:     "some-test-ad",
							ADomain: []string{"ttd.com"},
							CrID:    "29681110",
							H:       250,
							W:       300,
							DealID:  "testdeal",
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": 1}`),
							MType:   openrtb2.MarkupBanner,
						},
						BidType: openrtb_ext.BidTypeBanner,
					},
				},
				Currency: "USD",
			},
		},
		{
			name: "ignore_invalid_prebiddealpriority",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"mtype": 2, "id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["ttd.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
				},
			},
			wantErr: nil,
			wantResp: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							ID:      "7706636740145184841",
							ImpID:   "test-imp-id",
							Price:   0.500000,
							AdID:    "29681110",
							AdM:     "some-test-ad",
							ADomain: []string{"ttd.com"},
							CrID:    "29681110",
							H:       250,
							W:       300,
							DealID:  "testdeal",
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1}`),
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
			wantResp: adapters.NewBidderResponse(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, buildErr := Builder(openrtb_ext.BidderTheTradeDesk, config.Adapter{
				Endpoint:         `https://adsrvr.org/bid/bidder/{{.SupplyId}}`,
				ExtraAdapterInfo: "test",
			}, config.Server{})
			assert.Nil(t, buildErr)
			gotResp, gotErr := a.MakeBids(tt.args.internalRequest, tt.args.externalRequest, tt.args.response)
			assert.Equal(t, tt.wantErr, gotErr, gotErr)
			assert.Equal(t, tt.wantResp, gotResp)
		})
	}
}

func TestTheTradeDeskAdapter_BuildEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		supplySourceId   string
		defaultEndpoint  string
		expectedEndpoint string
		wantErr          []error
	}{
		{
			name:             "valid_supply_source_id",
			supplySourceId:   "pub_abc",
			defaultEndpoint:  "https://direct.adsrvr.org/bid/bidder/default_publisher",
			expectedEndpoint: "https://direct.adsrvr.org/bid/bidder/pub_abc",
			wantErr:          nil,
		},
		{
			name:             "empty_supply_source_id",
			supplySourceId:   "",
			defaultEndpoint:  "https://direct.adsrvr.org/bid/bidder/default_publisher",
			expectedEndpoint: "https://direct.adsrvr.org/bid/bidder/default_publisher",
			wantErr:          nil,
		},
		{
			name:             "empty_ssi_and_no_default_expect_err",
			supplySourceId:   "",
			defaultEndpoint:  "",
			expectedEndpoint: "",
			wantErr:          nil,
		},
	}

	endpointTemplate, err := template.New("endpointTemplate").Parse("https://direct.adsrvr.org/bid/bidder/{{.SupplyId}}")
	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &adapter{
				bidderEndpointTemplate: "https://direct.adsrvr.org/bid/bidder/{{.SupplyId}}",
				defaultEndpoint:        tt.defaultEndpoint,
				templateEndpoint:       endpointTemplate,
			}
			finalEndpoint, err := a.buildEndpointURL(tt.supplySourceId)
			if tt.wantErr != nil {
				assert.NotNil(t, err)
			}
			assert.Equal(t, tt.expectedEndpoint, finalEndpoint)
		})
	}
}
