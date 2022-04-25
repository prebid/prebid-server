package adapters

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestExtractAdapterReqBidderParamsMap(t *testing.T) {
	tests := []struct {
		name            string
		givenBidRequest *openrtb2.BidRequest
		want            map[string]json.RawMessage
		wantErr         error
	}{
		{
			name:            "nil req",
			givenBidRequest: nil,
			want:            nil,
			wantErr:         errors.New("error bidRequest should not be nil"),
		},
		{
			name:            "nil req.ext",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			want:            nil,
			wantErr:         nil,
		},
		{
			name:            "malformed req.ext",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage("malformed")},
			want:            nil,
			wantErr:         errors.New("error decoding Request.ext : invalid character 'm' looking for beginning of value"),
		},
		{
			name:            "extract bidder params from req.Ext for input request in adapter code",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams": {"profile": 1234, "version": 1}}}`)},
			want:            map[string]json.RawMessage{"profile": json.RawMessage(`1234`), "version": json.RawMessage(`1`)},
			wantErr:         nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractReqExtBidderParamsMap(tt.givenBidRequest)
			assert.Equal(t, tt.wantErr, err, "err")
			assert.Equal(t, tt.want, got, "result")
		})
	}
}

func TestExtractReqExtBidderParamsMap(t *testing.T) {
	tests := []struct {
		name            string
		givenBidRequest *openrtb2.BidRequest
		want            map[string]map[string]json.RawMessage
		wantErr         error
	}{
		{
			name:            "nil req",
			givenBidRequest: nil,
			want:            nil,
			wantErr:         errors.New("error bidRequest should not be nil"),
		},
		{
			name:            "nil req.ext",
			givenBidRequest: &openrtb2.BidRequest{Ext: nil},
			want:            nil,
			wantErr:         nil,
		},
		{
			name:            "malformed req.ext",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage("malformed")},
			want:            nil,
			wantErr:         errors.New("error decoding Request.ext : invalid character 'm' looking for beginning of value"),
		},
		{
			name:            "nil req.ext.prebid",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			want:            nil,
			wantErr:         nil,
		},
		{
			name:            "extract bidder params from req.Ext for input request before adapter code",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams": {"pubmatic": {"profile": 1234, "version": 1}, "appnexus": {"key1": 123, "key2": {"innerKey1":"innerValue1"} } }}}`)},
			want: map[string]map[string]json.RawMessage{
				"pubmatic": {"profile": json.RawMessage(`1234`), "version": json.RawMessage(`1`)},
				"appnexus": {"key1": json.RawMessage(`123`), "key2": json.RawMessage(`{"innerKey1":"innerValue1"}`)},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractReqExtBidderParamsEmbeddedMap(tt.givenBidRequest)
			assert.Equal(t, tt.wantErr, err, "err")
			assert.Equal(t, tt.want, got, "result")
		})
	}
}
func TestFilterArrayWithMap(t *testing.T) {

	staticList := []string{"abc", "def"}
	cases := []struct {
		tag       string
		baseList  []string
		filter    map[string]bool
		expOutput []string
		message   string
	}{
		{"Empty input", staticList, map[string]bool{}, []string{}, "failed to filter empty array\n"},
		{"Item not in list", staticList, map[string]bool{"xyz": true}, []string{}, "failed to item not in list\n"},
		{"Has one value", staticList, map[string]bool{"abc": true}, []string{"abc"}, "failed to filter one element\n"},
		{"Has more than one value", staticList, map[string]bool{"abc": true, "def": true}, []string{"abc", "def"}, "failed to filter more than one element\n"},
		{"No base set", []string{}, map[string]bool{"abc": true, "def": true}, []string{}, "found element in empty set\n"},
	}

	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			result := filterArrayWithMap(c.baseList, c.filter)
			if len(result) == 0 && len(c.expOutput) == 0 {
				return
			}
			if !reflect.DeepEqual(c.expOutput[:], result) {
				t.Fatalf(c.message)
			}
		})
	}
}
