package pubmatic

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPubmatic, config.Adapter{
		Endpoint: "https://hbopenbid.pubmatic.com/translator?source=prebid-server"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "pubmatictest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name          string
		mType         openrtb2.MarkupType
		expectedMType openrtb_ext.BidType
		expectedError error
	}{
		{
			name:          "Test Banner Bid Type",
			mType:         1,
			expectedMType: openrtb_ext.BidTypeBanner,
			expectedError: nil,
		},
		{
			name:          "Test Video Bid Type",
			mType:         2,
			expectedMType: openrtb_ext.BidTypeVideo,
			expectedError: nil,
		},
		{
			name:          "Test Audio Bid Type",
			mType:         3,
			expectedMType: openrtb_ext.BidTypeAudio,
			expectedError: nil,
		},
		{
			name:          "Test Native Bid Type",
			mType:         4,
			expectedMType: openrtb_ext.BidTypeNative,
			expectedError: nil,
		},
		{
			name:          "Test Unsupported MType (Invalid MType)",
			mType:         44,
			expectedMType: "", // default value for unsupported types
			expectedError: &errortypes.BadServerResponse{Message: "failed to parse bid mtype (44) for impression id "},
		},
		{
			name:          "Test Default MType (MType 0 or Not Set)",
			mType:         0,  // This represents the default case where MType is not explicitly set
			expectedMType: "", // Default is empty and return error
			expectedError: &errortypes.BadServerResponse{Message: "failed to parse bid mtype (0) for impression id "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bid := openrtb2.Bid{
				MType: tt.mType,
			}

			actualBidTypeValue, actualError := getMediaTypeForBid(&bid)

			assert.Equal(t, tt.expectedMType, actualBidTypeValue, "unexpected BidType value")

			if tt.expectedError != nil {
				require.EqualError(t, actualError, tt.expectedError.Error(), "unexpected error message")
				return
			}
			require.NoError(t, actualError, "expected no error but got one")
		})
	}
}

func TestParseImpressionObject(t *testing.T) {
	type args struct {
		imp                      *openrtb2.Imp
		extractWrapperExtFromImp bool
		extractPubIDFromImp      bool
		displayManager           string
		displayManagerVer        string
	}
	type want struct {
		bidfloor          float64
		impExt            json.RawMessage
		displayManager    string
		displayManagerVer string
	}
	tests := []struct {
		name                string
		args                args
		expectedWrapperExt  *pubmaticWrapperExt
		expectedPublisherId string
		want                want
		wantErr             bool
	}{
		{
			name: "imp.bidfloor_empty_and_kadfloor_set",
			args: args{
				imp: &openrtb2.Imp{
					Video: &openrtb2.Video{},
					Ext:   json.RawMessage(`{"bidder":{"kadfloor":"0.12"}}`),
				},
			},
			want: want{
				bidfloor: 0.12,
			},
		},
		{
			name: "imp.bidfloor_set_and_kadfloor_empty",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{}}`),
				},
			},
			want: want{
				bidfloor: 0.12,
			},
		},
		{
			name: "imp.bidfloor_set_and_kadfloor_invalid",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":"aaa"}}`),
				},
			},
			want: want{
				bidfloor: 0.12,
			}},
		{
			name: "imp.bidfloor_set_and_kadfloor_set_higher_imp.bidfloor",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":"0.11"}}`),
				},
			},
			want: want{
				bidfloor: 0.12,
			}},
		{
			name: "imp.bidfloor_set_and_kadfloor_set,_higher_kadfloor",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":"0.13"}}`),
				},
			},
			want: want{
				bidfloor: 0.13,
			}},
		{
			name: "kadfloor_string_set_with_whitespace",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":" \t  0.13  "}}`),
				},
			},
			want: want{
				bidfloor: 0.13,
			}},
		{
			name: "Populate_imp.displaymanager_and_imp.displaymanagerver_if_both_are_empty_in_imp",
			args: args{
				imp: &openrtb2.Imp{
					Video: &openrtb2.Video{},
					Ext:   json.RawMessage(`{"bidder":{"kadfloor":"0.12"}}`),
				},
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
			want: want{
				bidfloor:          0.12,
				impExt:            json.RawMessage(nil),
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
		{
			name: "do_not_populate_imp.displaymanager_and_imp.displaymanagerver_in_imp_if_only_displaymanager_or_displaymanagerver_is_present_in_args",
			args: args{
				imp: &openrtb2.Imp{
					Video:             &openrtb2.Video{},
					Ext:               json.RawMessage(`{"bidder":{"kadfloor":"0.12"}}`),
					DisplayManagerVer: "1.0.0",
				},
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
			want: want{
				bidfloor:          0.12,
				impExt:            json.RawMessage(nil),
				displayManagerVer: "1.0.0",
			},
		},
		{
			name: "do_not_populate_imp.displaymanager_and_imp.displaymanagerver_if_already_present_in_imp",
			args: args{
				imp: &openrtb2.Imp{
					Video:             &openrtb2.Video{},
					Ext:               json.RawMessage(`{"bidder":{"kadfloor":"0.12"}}`),
					DisplayManager:    "prebid-mobile",
					DisplayManagerVer: "1.0.0",
				},
				displayManager:    "prebid-android",
				displayManagerVer: "2.0.0",
			},
			want: want{
				bidfloor:          0.12,
				impExt:            json.RawMessage(nil),
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedWrapperExt, receivedPublisherId, err := parseImpressionObject(tt.args.imp, tt.args.extractWrapperExtFromImp, tt.args.extractPubIDFromImp, tt.args.displayManager, tt.args.displayManagerVer)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedWrapperExt, receivedWrapperExt)
			assert.Equal(t, tt.expectedPublisherId, receivedPublisherId)
			assert.Equal(t, tt.want.bidfloor, tt.args.imp.BidFloor)
			assert.Equal(t, tt.want.displayManager, tt.args.imp.DisplayManager)
			assert.Equal(t, tt.want.displayManagerVer, tt.args.imp.DisplayManagerVer)
		})
	}
}

func TestExtractPubmaticExtFromRequest(t *testing.T) {
	type args struct {
		request *openrtb2.BidRequest
	}
	tests := []struct {
		name           string
		args           args
		expectedReqExt extRequestAdServer
		wantErr        bool
	}{
		{
			name: "nil_request",
			args: args{
				request: nil,
			},
			wantErr: false,
		},
		{
			name: "nil_req.ext",
			args: args{
				request: &openrtb2.BidRequest{Ext: nil},
			},
			wantErr: false,
		},
		{
			name: "Pubmatic_wrapper_ext_missing/empty_(empty_bidderparms)",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{},
			wantErr:        false,
		},
		{
			name: "Only_Pubmatic_wrapper_ext_present",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Wrapper: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			},
			wantErr: false,
		},
		{
			name: "Invalid_Pubmatic_wrapper_ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":"}}}`),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid_Pubmatic_acat_ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":[" drg \t","dlu","ssr"],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Wrapper: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
				Acat:    []string{"drg", "dlu", "ssr"},
			},
			wantErr: false,
		},
		{
			name: "Invalid_Pubmatic_acat_ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":[1,3,4],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Wrapper: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			},
			wantErr: true,
		},
		{
			name: "Valid_Pubmatic_marketplace_ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}},"bidderparams":{"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Marketplace: &marketplaceReqExt{AllowedBidders: []string{"pubmatic", "groupm"}},
				Wrapper:     &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReqExt, err := extractPubmaticExtFromRequest(tt.args.request)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedReqExt, gotReqExt)
		})
	}
}

func TestPubmaticAdapter_MakeRequests(t *testing.T) {
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
		// Happy paths covered by TestJsonSamples()
		// Covering only error scenarios here
		{
			name: "invalid_bidderparams",
			args: args{
				request: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":"123"}}}`)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PubmaticAdapter{
				URI: tt.fields.URI,
			}
			gotReqData, gotErr := a.MakeRequests(tt.args.request, tt.args.reqInfo)
			assert.Equal(t, tt.wantErr, len(gotErr) != 0)
			assert.Equal(t, tt.expectedReqData, gotReqData)
		})
	}
}

func TestPubmaticAdapter_MakeBids(t *testing.T) {
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
			name: "happy_path,_valid_response_with_all_bid_params",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["pubmatic.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "mtype": 1, "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": 1}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
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
							ADomain: []string{"pubmatic.com"},
							CrID:    "29681110",
							H:       250,
							W:       300,
							DealID:  "testdeal",
							MType:   1,
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": 1}`),
						},
						DealPriority: 1,
						BidType:      openrtb_ext.BidTypeBanner,
						BidVideo:     &openrtb_ext.ExtBidPrebidVideo{},
						BidMeta: &openrtb_ext.ExtBidPrebidMeta{
							MediaType: "banner",
						},
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
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["pubmatic.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal","mtype": 1, "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
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
							ADomain: []string{"pubmatic.com"},
							CrID:    "29681110",
							H:       250,
							W:       300,
							DealID:  "testdeal",
							MType:   1,
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1}`),
						},
						BidType:  openrtb_ext.BidTypeBanner,
						BidVideo: &openrtb_ext.ExtBidPrebidVideo{},
						BidMeta: &openrtb_ext.ExtBidPrebidMeta{
							MediaType: "banner",
						},
					},
				},
				Currency: "USD",
			},
		},
		{
			name: "bid_ext_InBannerVideo_is_true",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["pubmatic.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "mtype": 1, "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1, "ibv": true}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
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
							ADomain: []string{"pubmatic.com"},
							CrID:    "29681110",
							H:       250,
							W:       300,
							DealID:  "testdeal",
							MType:   1,
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1, "ibv": true}`),
						},
						BidType:  openrtb_ext.BidTypeBanner,
						BidVideo: &openrtb_ext.ExtBidPrebidVideo{},
						BidMeta: &openrtb_ext.ExtBidPrebidMeta{
							MediaType: "video",
						},
					},
				},
				Currency: "USD",
			},
		},
		{
			name: "getMediaTypeForBid_return_error",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["pubmatic.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "mtype": 0, "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1, "ibv": true}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
				},
			},
			wantErr: []error{&errortypes.BadServerResponse{Message: "failed to parse bid mtype (0) for impression id test-imp-id"}},
			wantResp: &adapters.BidderResponse{
				Bids:     []*adapters.TypedBid{},
				Currency: "USD",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PubmaticAdapter{
				URI: tt.fields.URI,
			}
			gotResp, gotErr := a.MakeBids(tt.args.internalRequest, tt.args.externalRequest, tt.args.response)
			assert.Equal(t, tt.wantErr, gotErr, gotErr)
			assert.Equal(t, tt.wantResp, gotResp)
		})
	}
}

func Test_getAlternateBidderCodesFromRequest(t *testing.T) {
	type args struct {
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "request.ext_nil",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: nil},
			},
			want: nil,
		},
		{
			name: "alternatebiddercodes_not_present_in_request.ext",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			},
			want: nil,
		},
		{
			name: "alternatebiddercodes_feature_disabled",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`)},
			},
			want: []string{"pubmatic"},
		},
		{
			name: "alternatebiddercodes_disabled_at_bidder_level",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":false,"allowedbiddercodes":["groupm"]}}}}}`)},
			},
			want: []string{"pubmatic"},
		},
		{
			name: "alternatebiddercodes_list_not_defined",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true}}}}}`)},
			},
			want: []string{"all"},
		},
		{
			name: "wildcard_in_alternatebiddercodes_list",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["*"]}}}}}`)},
			},
			want: []string{"all"},
		},
		{
			name: "empty_alternatebiddercodes_list",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":[]}}}}}`)},
			},
			want: []string{"pubmatic"},
		},
		{
			name: "only_groupm_in_alternatebiddercodes_allowed",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`)},
			},
			want: []string{"pubmatic", "groupm"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqExt *openrtb_ext.ExtRequest
			if len(tt.args.bidRequest.Ext) > 0 {
				err := json.Unmarshal(tt.args.bidRequest.Ext, &reqExt)
				if err != nil {
					t.Errorf("getAlternateBidderCodesFromRequest() = %v", err)
				}
			}

			got := getAlternateBidderCodesFromRequestExt(reqExt)
			assert.ElementsMatch(t, got, tt.want, tt.name)
		})
	}
}

func TestPopulateFirstPartyDataImpAttributes(t *testing.T) {
	type args struct {
		data      json.RawMessage
		impExtMap map[string]interface{}
	}
	tests := []struct {
		name           string
		args           args
		expectedImpExt map[string]interface{}
	}{
		{
			name: "Only_Targeting_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting_and_adserver_object_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"adserver": {"name": "gam","adslot": "/1111/home"},"pbadslot": "/2222/home","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"dfp_ad_unit_code": "/1111/home",
				"key_val":          "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting_and_pbadslot_key_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"pbadslot": "/2222/home","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"dfp_ad_unit_code": "/2222/home",
				"key_val":          "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting_and_Invalid_Adserver_object_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"adserver": "invalid","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "sport=rugby,cricket",
			},
		},
		{
			name: "key_val_already_present_in_imp.ext.data",
			args: args{
				data: json.RawMessage(`{"sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{
					"key_val": "k1=v1|k2=v2",
				},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "k1=v1|k2=v2|sport=rugby,cricket",
			},
		},
		{
			name: "int_data_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"age": 25}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "age=25",
			},
		},
		{
			name: "float_data_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"floor": 0.15}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "floor=0.15",
			},
		},
		{
			name: "bool_data_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"k1": true}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "k1=true",
			},
		},
		{
			name: "imp.ext.data_is_not_present",
			args: args{
				data:      nil,
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{},
		},
		{
			name: "string_with_spaces_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"  category  ": "   cinema  "}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "category=cinema",
			},
		},
		{
			name: "string_array_with_spaces_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"  country\t": ["  India", "\tChina  "]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "country=India,China",
			},
		},
		{
			name: "Invalid_data_present_in_imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"country": [1, "India"],"category":"movies"}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "category=movies",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			populateFirstPartyDataImpAttributes(tt.args.data, tt.args.impExtMap)
			assert.Equal(t, tt.expectedImpExt, tt.args.impExtMap)
		})
	}
}

func TestPopulateFirstPartyDataImpAttributesForMultipleAttributes(t *testing.T) {
	impExtMap := map[string]interface{}{
		"key_val": "k1=v1|k2=v2",
	}
	data := json.RawMessage(`{"sport":["rugby","cricket"],"pageType":"article","age":30,"floor":1.25}`)
	expectedKeyValArr := []string{"age=30", "floor=1.25", "k1=v1", "k2=v2", "pageType=article", "sport=rugby,cricket"}

	populateFirstPartyDataImpAttributes(data, impExtMap)

	//read dctr value and split on "|" for comparison
	actualKeyValArr := strings.Split(impExtMap[dctrKeyName].(string), "|")
	sort.Strings(actualKeyValArr)
	assert.Equal(t, expectedKeyValArr, actualKeyValArr)
}

func TestGetStringArray(t *testing.T) {
	tests := []struct {
		name   string
		input  []interface{}
		output []string
	}{
		{
			name:   "Valid_String_Array",
			input:  append(make([]interface{}, 0), "hello", "world"),
			output: []string{"hello", "world"},
		},
		{
			name:   "Invalid_String_Array",
			input:  append(make([]interface{}, 0), 1, 2),
			output: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringArray(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestGetMapFromJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  json.RawMessage
		output map[string]interface{}
	}{
		{
			name:  "Valid_JSON",
			input: json.RawMessage(`{"buyid":"testBuyId"}`),
			output: map[string]interface{}{
				"buyid": "testBuyId",
			},
		},
		{
			name:   "Invalid_JSON",
			input:  json.RawMessage(`{"buyid":}`),
			output: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMapFromJSON(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestGetDisplayManagerAndVer(t *testing.T) {
	type args struct {
		app *openrtb2.App
	}
	type want struct {
		displayManager    string
		displayManagerVer string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "request_app_object_is_not_nil_but_app.ext_has_no_source_and_version",
			args: args{

				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_app.ext_has_source_and_version",
			args: args{

				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile","version":"1.0.0"}`),
				},
			},
			want: want{
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_app.ext.prebid_has_source_and_version",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"prebid":{"source":"prebid-mobile","version":"1.0.0"}}`),
				},
			},
			want: want{
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_app.ext_has_only_version",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"version":"1.0.0"}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_app.ext_has_only_source",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile"}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_app.ext_have_empty_source_but_version_is_present",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"", "version":"1.0.0"}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_app.ext_have_empty_version_but_source_is_present",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile", "version":""}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request_app_object_is_not_nil_and_both_app.ext_and_app.ext.prebid_have_source_and_version",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile-android","version":"2.0.0","prebid":{"source":"prebid-mobile","version":"1.0.0"}}`),
				},
			},
			want: want{
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			displayManager, displayManagerVer := getDisplayManagerAndVer(tt.args.app)
			assert.Equal(t, tt.want.displayManager, displayManager)
			assert.Equal(t, tt.want.displayManagerVer, displayManagerVer)
		})
	}
}
