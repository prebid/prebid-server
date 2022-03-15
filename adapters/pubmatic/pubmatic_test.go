package pubmatic

import (
	"encoding/json"
	"reflect"
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

func Test_parseImpressionObject(t *testing.T) {
	type args struct {
		imp                      *openrtb2.Imp
		extractWrapperExtFromImp bool
		extractPubIDFromImp      bool
	}
	tests := []struct {
		name    string
		args    args
		want    *pubmaticWrapperExt
		want1   string
		wantErr bool
		wantImp *openrtb2.Imp
	}{
		{
			name: "imp.bidfloor empty and kadfloor set",
			args: args{
				imp: &openrtb2.Imp{
					Video: &openrtb2.Video{},
					Ext:   json.RawMessage(`{"bidder":{"kadfloor":"0.12"}}`),
				},
			},
			wantImp: &openrtb2.Imp{
				BidFloor: 0.12,
			},
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
			wantImp: &openrtb2.Imp{
				BidFloor: 0.12,
			},
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
			wantImp: &openrtb2.Imp{
				BidFloor: 0.12,
			},
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
			wantImp: &openrtb2.Imp{
				BidFloor: 0.11,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseImpressionObject(tt.args.imp, tt.args.extractWrapperExtFromImp, tt.args.extractPubIDFromImp)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseImpressionObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseImpressionObject() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseImpressionObject() got1 = %v, want %v", got1, tt.want1)
			}
			assert.Equal(t, tt.args.imp.BidFloor, tt.wantImp.BidFloor)
		})
	}
}
