package vastbidder

import (
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

//TestMakeRequests verifies
// 1. default and custom headers are set
func TestMakeRequests(t *testing.T) {

	type args struct {
		customHeaders map[string]string
		req           *openrtb2.BidRequest
	}
	type want struct {
		impIDReqHeaderMap map[string]http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "multi_impression_req",
			args: args{
				customHeaders: map[string]string{
					"my-custom-header": "custom-value",
				},
				req: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP:       "1.1.1.1",
						UA:       "user-agent",
						Language: "en",
					},
					Site: &openrtb2.Site{
						Page: "http://test.com/",
					},
					Imp: []openrtb2.Imp{
						{ // vast 2.0
							ID: "vast_2_0_imp_req",
							Video: &openrtb2.Video{
								Protocols: []openrtb2.Protocol{
									openrtb2.ProtocolVAST20,
								},
							},
							Ext: []byte(`{"bidder" :{}}`),
						},
						{
							ID: "vast_4_0_imp_req",
							Video: &openrtb2.Video{ // vast 4.0
								Protocols: []openrtb2.Protocol{
									openrtb2.ProtocolVAST40,
								},
							},
							Ext: []byte(`{"bidder" :{}}`),
						},
						{
							ID: "vast_2_0_4_0_wrapper_imp_req",
							Video: &openrtb2.Video{ // vast 2 and 4.0 wrapper
								Protocols: []openrtb2.Protocol{
									openrtb2.ProtocolVAST40Wrapper,
									openrtb2.ProtocolVAST20,
								},
							},
							Ext: []byte(`{"bidder" :{}}`),
						},
						{
							ID: "other_non_vast_protocol",
							Video: &openrtb2.Video{ // DAAST 1.0
								Protocols: []openrtb2.Protocol{
									openrtb2.ProtocolDAAST10,
								},
							},
							Ext: []byte(`{"bidder" :{}}`),
						},
						{

							ID: "no_protocol_field_set",
							Video: &openrtb2.Video{ // vast 2 and 4.0 wrapper
								Protocols: []openrtb2.Protocol{},
							},
							Ext: []byte(`{"bidder" :{}}`),
						},
					},
				},
			},
			want: want{
				impIDReqHeaderMap: map[string]http.Header{
					"vast_2_0_imp_req": {
						"X-Forwarded-For":  []string{"1.1.1.1"},
						"User-Agent":       []string{"user-agent"},
						"My-Custom-Header": []string{"custom-value"},
					},
					"vast_4_0_imp_req": {
						"X-Device-Ip":              []string{"1.1.1.1"},
						"X-Device-User-Agent":      []string{"user-agent"},
						"X-Device-Referer":         []string{"http://test.com/"},
						"X-Device-Accept-Language": []string{"en"},
						"My-Custom-Header":         []string{"custom-value"},
					},
					"vast_2_0_4_0_wrapper_imp_req": {
						"X-Device-Ip":              []string{"1.1.1.1"},
						"X-Forwarded-For":          []string{"1.1.1.1"},
						"X-Device-User-Agent":      []string{"user-agent"},
						"User-Agent":               []string{"user-agent"},
						"X-Device-Referer":         []string{"http://test.com/"},
						"X-Device-Accept-Language": []string{"en"},
						"My-Custom-Header":         []string{"custom-value"},
					},
					"other_non_vast_protocol": {
						"My-Custom-Header": []string{"custom-value"},
					}, // no default headers expected
					"no_protocol_field_set": { // set all default headers
						"X-Device-Ip":              []string{"1.1.1.1"},
						"X-Forwarded-For":          []string{"1.1.1.1"},
						"X-Device-User-Agent":      []string{"user-agent"},
						"User-Agent":               []string{"user-agent"},
						"X-Device-Referer":         []string{"http://test.com/"},
						"X-Device-Accept-Language": []string{"en"},
						"My-Custom-Header":         []string{"custom-value"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidderName := openrtb_ext.BidderName("myVastBidderMacro")
			RegisterNewBidderMacro(bidderName, func() IBidderMacro {
				return newMyVastBidderMacro(tt.args.customHeaders)
			})
			bidder := NewTagBidder(bidderName, config.Adapter{})
			reqData, err := bidder.MakeRequests(tt.args.req, nil)
			assert.Nil(t, err)
			for _, req := range reqData {
				impID := tt.args.req.Imp[req.Params.ImpIndex].ID
				expectedHeaders := tt.want.impIDReqHeaderMap[impID]
				assert.Equal(t, expectedHeaders, req.Headers, "test for - "+impID)
			}
		})
	}
}
