package adapters

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"reflect"
	"testing"
)

func TestExtractAdapterReqBidderParams(t *testing.T) {
	type args struct {
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]json.RawMessage
		wantErr bool
	}{
		{
			name: "extract bidder params from nil req",
			args: args{
				bidRequest: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "extract bidder params from nil req.Ext",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "extract bidder params from req.Ext for input request in adapter code",
			args: args{
				bidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams": {"profile": 1234, "version": 1}}}`)},
			},
			want:    map[string]json.RawMessage{"profile": json.RawMessage(`1234`), "version": json.RawMessage(`1`)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractAdapterReqBidderParams(tt.args.bidRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractReqExtBidderParams() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractReqExtBidderParams() got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestExtractReqExtBidderParams(t *testing.T) {
	type args struct {
		request *openrtb2.BidRequest
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]map[string]json.RawMessage
		wantErr bool
	}{
		{
			name: "extract bidder params from nil req",
			args: args{
				request: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "extract bidder params from nil req.Ext.prebid",
			args: args{
				request: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "extract bidder params from nil req.Ext",
			args: args{
				request: &openrtb2.BidRequest{Ext: nil},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "extract bidder params from req.Ext for input request before adapter code",
			args: args{
				request: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams": {"pubmatic": {"profile": 1234, "version": 1}, "appnexus": {"key1": 123, "key2": {"innerKey1":"innerValue1"} } }}}`)},
			},
			want: map[string]map[string]json.RawMessage{
				"pubmatic": {"profile": json.RawMessage(`1234`), "version": json.RawMessage(`1`)},
				"appnexus": {"key1": json.RawMessage(`123`), "key2": json.RawMessage(`{"innerKey1":"innerValue1"}`)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractReqExtBidderParams(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractReqExtBidderParams() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractReqExtBidderParams() got = %v, want = %v", got, tt.want)
			}
		})
	}
}
