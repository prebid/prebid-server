package pubmatic

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPubmatic, config.Adapter{
		Endpoint: "https://hbopenbid.pubmatic.com/translator?source=prebid-server"})

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

func TestExtractPubmaticExtFromRequest(t *testing.T) {
	type args struct {
		request *openrtb2.BidRequest
	}
	tests := []struct {
		name               string
		args               args
		expectedWrapperExt *pubmaticWrapperExt
		expectedAcat       []string
		wantErr            bool
	}{
		{
			name:    "Empty bidder param",
			wantErr: true,
		},
		{
			name: "Pubmatic wrapper ext missing/empty",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{}}}`),
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
			expectedWrapperExt: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			wantErr:            false,
		},
		{
			name: "Invalid Pubmatic wrapper ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"wrapper":{"profile":"123","version":456}}}}`),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Pubmatic acat ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":["drg”,”dlu”,”ssr"],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedWrapperExt: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			expectedAcat:       []string{"drg”,”dlu”,”ssr"},
			wantErr:            false,
		},
		{
			name: "Invalid Pubmatic acat ext. We are ok with acat being non nil in this case as we are returning unmarshal error",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":[1,3,4],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedWrapperExt: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			expectedAcat:       []string{"", "", ""},
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWrapperExt, gotAcat, err := extractPubmaticExtFromRequest(tt.args.request)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedWrapperExt, gotWrapperExt)
			assert.Equal(t, tt.expectedAcat, gotAcat)
		})
	}
}
