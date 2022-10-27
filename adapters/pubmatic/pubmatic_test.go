package pubmatic

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPubmatic, config.Adapter{
		Endpoint: "https://hbopenbid.pubmatic.com/translator?source=prebid-server"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "pubmatictest", bidder)
}

func TestGetBidTypeVideo(t *testing.T) {
	pubmaticExt := &pubmaticBidExt{}
	pubmaticExt.BidType = new(int)
	*pubmaticExt.BidType = 1
	actualBidTypeValue := getBidType(pubmaticExt)
	if actualBidTypeValue != openrtb_ext.BidTypeVideo {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeVideo, actualBidTypeValue)
	}
}

func TestGetBidTypeForMissingBidTypeExt(t *testing.T) {
	pubmaticExt := &pubmaticBidExt{}
	actualBidTypeValue := getBidType(pubmaticExt)
	// banner is the default bid type when no bidType key is present in the bid.ext
	if actualBidTypeValue != "banner" {
		t.Errorf("Expected Bid Type value was: banner, actual value is: %v", actualBidTypeValue)
	}
}

func TestGetBidTypeBanner(t *testing.T) {
	pubmaticExt := &pubmaticBidExt{}
	pubmaticExt.BidType = new(int)
	*pubmaticExt.BidType = 0
	actualBidTypeValue := getBidType(pubmaticExt)
	if actualBidTypeValue != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeBanner, actualBidTypeValue)
	}
}

func TestGetBidTypeNative(t *testing.T) {
	pubmaticExt := &pubmaticBidExt{}
	pubmaticExt.BidType = new(int)
	*pubmaticExt.BidType = 2
	actualBidTypeValue := getBidType(pubmaticExt)
	if actualBidTypeValue != openrtb_ext.BidTypeNative {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeNative, actualBidTypeValue)
	}
}

func TestGetBidTypeForUnsupportedCode(t *testing.T) {
	pubmaticExt := &pubmaticBidExt{}
	pubmaticExt.BidType = new(int)
	*pubmaticExt.BidType = 99
	actualBidTypeValue := getBidType(pubmaticExt)
	if actualBidTypeValue != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeBanner, actualBidTypeValue)
	}
}

func TestParseImpressionObject(t *testing.T) {
	type args struct {
		imp                      *openrtb2.Imp
		extractWrapperExtFromImp bool
		extractPubIDFromImp      bool
	}
	tests := []struct {
		name                string
		args                args
		expectedWrapperExt  *pubmaticWrapperExt
		expectedPublisherId string
		wantErr             bool
		expectedBidfloor    float64
	}{
		{
			name: "imp.bidfloor empty and kadfloor set",
			args: args{
				imp: &openrtb2.Imp{
					Video: &openrtb2.Video{},
					Ext:   json.RawMessage(`{"bidder":{"kadfloor":"0.12"}}`),
				},
			},
			expectedBidfloor: 0.12,
		},
		{
			name: "imp.bidfloor set and kadfloor empty",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{}}`),
				},
			},
			expectedBidfloor: 0.12,
		},
		{
			name: "imp.bidfloor set and kadfloor invalid",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":"aaa"}}`),
				},
			},
			expectedBidfloor: 0.12,
		},
		{
			name: "imp.bidfloor set and kadfloor set, preference to kadfloor",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":"0.11"}}`),
				},
			},
			expectedBidfloor: 0.11,
		},
		{
			name: "kadfloor string set with whitespace",
			args: args{
				imp: &openrtb2.Imp{
					BidFloor: 0.12,
					Video:    &openrtb2.Video{},
					Ext:      json.RawMessage(`{"bidder":{"kadfloor":" \t  0.13  "}}`),
				},
			},
			expectedBidfloor: 0.13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedWrapperExt, receivedPublisherId, err := parseImpressionObject(tt.args.imp, tt.args.extractWrapperExtFromImp, tt.args.extractPubIDFromImp)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedWrapperExt, receivedWrapperExt)
			assert.Equal(t, tt.expectedPublisherId, receivedPublisherId)
			assert.Equal(t, tt.expectedBidfloor, tt.args.imp.BidFloor)
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
			name: "nil request",
			args: args{
				request: nil,
			},
			wantErr: false,
		},
		{
			name: "nil req.ext",
			args: args{
				request: &openrtb2.BidRequest{Ext: nil},
			},
			wantErr: false,
		},
		{
			name: "Pubmatic wrapper ext missing/empty (empty bidderparms)",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				ExtRequest: openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						BidderParams: json.RawMessage("{}"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Only Pubmatic wrapper ext present",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Wrapper: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
				ExtRequest: openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						BidderParams: json.RawMessage(`{"wrapper":{"profile":123,"version":456}}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid Pubmatic wrapper ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":"}}}`),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Pubmatic acat ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":[" drg \t","dlu","ssr"],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Wrapper: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
				Acat:    []string{"drg", "dlu", "ssr"},
				ExtRequest: openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						BidderParams: json.RawMessage(`{"acat":[" drg \t","dlu","ssr"],"wrapper":{"profile":123,"version":456}}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid Pubmatic acat ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":[1,3,4],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Wrapper: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
				ExtRequest: openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						BidderParams: json.RawMessage(`{"acat":[1,3,4],"wrapper":{"profile":123,"version":456}}`),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Pubmatic marketplace ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}},"bidderparams":{"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedReqExt: extRequestAdServer{
				Marketplace: &marketplaceReqExt{AllowedBidders: []string{"pubmatic", "groupm"}},
				Wrapper:     &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
				ExtRequest: openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						BidderParams:         json.RawMessage(`{"wrapper":{"profile":123,"version":456}}`),
						AlternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{Enabled: true, Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{"pubmatic": {Enabled: true, AllowedBidderCodes: []string{"groupm"}}}},
					},
				},
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
			name: "invalid bidderparams",
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
			name: "happy path, valid response with all bid params",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["pubmatic.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": 1}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
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
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": 1}`),
						},
						DealPriority: 1,
						BidType:      openrtb_ext.BidTypeBanner,
						BidVideo:     &openrtb_ext.ExtBidPrebidVideo{},
					},
				},
				Currency: "USD",
			},
		},
		{
			name: "ignore invalid prebiddealpriority",
			args: args{
				response: &adapters.ResponseData{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"id": "test-request-id", "seatbid":[{"seat": "958", "bid":[{"id": "7706636740145184841", "impid": "test-imp-id", "price": 0.500000, "adid": "29681110", "adm": "some-test-ad", "adomain":["pubmatic.com"], "crid": "29681110", "h": 250, "w": 300, "dealid": "testdeal", "ext":{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1}}]}], "bidid": "5778926625248726496", "cur": "USD"}`),
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
							Ext:     json.RawMessage(`{"dspid": 6, "deal_channel": 1, "prebiddealpriority": -1}`),
						},
						BidType:  openrtb_ext.BidTypeBanner,
						BidVideo: &openrtb_ext.ExtBidPrebidVideo{},
					},
				},
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
			name: "request.ext nil",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: nil},
			},
			want: nil,
		},
		{
			name: "alternatebiddercodes not present in request.ext",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			},
			want: nil,
		},
		{
			name: "alternatebiddercodes feature disabled",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`)},
			},
			want: []string{"pubmatic"},
		},
		{
			name: "alternatebiddercodes disabled at bidder level",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":false,"allowedbiddercodes":["groupm"]}}}}}`)},
			},
			want: []string{"pubmatic"},
		},
		{
			name: "alternatebiddercodes list not defined",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true}}}}}`)},
			},
			want: []string{"all"},
		},
		{
			name: "wildcard in alternatebiddercodes list",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["*"]}}}}}`)},
			},
			want: []string{"all"},
		},
		{
			name: "empty alternatebiddercodes list",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":[]}}}}}`)},
			},
			want: []string{"pubmatic"},
		},
		{
			name: "only groupm in alternatebiddercodes allowed",
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
