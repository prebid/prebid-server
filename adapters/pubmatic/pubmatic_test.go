package pubmatic

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
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
		name               string
		args               args
		expectedWrapperExt *pubmaticWrapperExt
		expectedAcat       []string
		expectedCookie     []string
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
					Ext: json.RawMessage(`{"prebid":{"bidderparams":{"acat":[" drg \t","dlu","ssr"],"wrapper":{"profile":123,"version":456}}}}`),
				},
			},
			expectedWrapperExt: &pubmaticWrapperExt{ProfileID: 123, VersionID: 456},
			expectedAcat:       []string{"drg", "dlu", "ssr"},
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
			gotWrapperExt, gotAcat, gotCookie, err := extractPubmaticExtFromRequest(tt.args.request)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectedWrapperExt, gotWrapperExt)
			assert.Equal(t, tt.expectedAcat, gotAcat)
			assert.Equal(t, tt.expectedCookie, gotCookie)
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
			name: "Only Targeting present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting and adserver object present in imp.ext.data",
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
			name: "Targeting and pbadslot key present in imp.ext.data ",
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
			name: "Targeting and Invalid Adserver object in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"adserver": "invalid","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "sport=rugby,cricket",
			},
		},
		{
			name: "key_val already present in imp.ext.data",
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
			name: "int data present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"age": 25}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "age=25",
			},
		},
		{
			name: "float data present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"floor": 0.15}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "floor=0.15",
			},
		},
		{
			name: "bool data present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"k1": true}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "k1=true",
			},
		},
		{
			name: "imp.ext.data is not present",
			args: args{
				data:      nil,
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{},
		},
		{
			name: "string with spaces present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"  category  ": "   cinema  "}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "category=cinema",
			},
		},
		{
			name: "string array with spaces present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"  country\t": ["  India", "\tChina  "]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "country=India,China",
			},
		},
		{
			name: "Invalid data present in imp.ext.data",
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
		input  interface{}
		output []string
	}{
		{
			name:   "Valid String Array",
			input:  append(make([]interface{}, 0), "hello", "world"),
			output: []string{"hello", "world"},
		},
		{
			name:   "Invalid String Array",
			input:  "hello",
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

func TestIsStringArray(t *testing.T) {
	tests := []struct {
		name   string
		input  []interface{}
		output bool
	}{
		{
			name:   "Valid String Array",
			input:  append(make([]interface{}, 0), "hello", "world"),
			output: true,
		},
		{
			name:   "Invalid String Array",
			input:  append(make([]interface{}, 0), 1, 2),
			output: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStringArray(tt.input)
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
			name:  "Valid JSON",
			input: json.RawMessage("{\"buyid\":\"testBuyId\"}"),
			output: map[string]interface{}{
				"buyid": "testBuyId",
			},
		},
		{
			name:   "Invalid JSON",
			input:  json.RawMessage("{\"buyid\":}"),
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
