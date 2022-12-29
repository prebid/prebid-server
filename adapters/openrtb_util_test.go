package adapters

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
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
