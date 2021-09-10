package exchange

import (
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func Test_eventsData_makeBidExtEvents(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bidType           openrtb_ext.BidType
		generatedBidId    string
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.ExtBidPrebidEvents
	}{
		{
			name: "banner: events enabled for request, disabled for account",
			args: args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "banner: events enabled for account, disabled for request",
			args: args{enabledForAccount: true, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "banner: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			want: nil,
		},
		{
			name: "video: events enabled for account and request",
			args: args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			want: nil,
		},
		{
			name: "video: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			want: nil,
		},
		{
			name: "banner: use generated bid id",
			args: args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: "randomId"},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=randomId&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=randomId&a=123456&bidder=openx&ts=1234567890",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventTracking{
				enabledForAccount:  tt.args.enabledForAccount,
				enabledForRequest:  tt.args.enabledForRequest,
				accountID:          "123456",
				auctionTimestampMs: 1234567890,
				externalURL:        "http://localhost",
			}
			bid := &pbsOrtbBid{bid: &openrtb2.Bid{ID: "BID-1"}, bidType: tt.args.bidType, generatedBidID: tt.args.generatedBidId}
			assert.Equal(t, tt.want, evData.makeBidExtEvents(bid, openrtb_ext.BidderOpenx))
		})
	}
}

func Test_eventsData_modifyBidJSON(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bidType           openrtb_ext.BidType
		generatedBidId    string
	}
	tests := []struct {
		name      string
		args      args
		jsonBytes []byte
		want      []byte
	}{
		{
			name:      "banner: events enabled for request, disabled for account",
			args:      args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name:      "banner: events enabled for account, disabled for request",
			args:      args{enabledForAccount: true, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name:      "banner: events disabled for account and request",
			args:      args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something"}`),
		},
		{
			name:      "video: events disabled for account and request",
			args:      args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something"}`),
		},
		{
			name:      "video: events enabled for account and request",
			args:      args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something"}`),
		},
		{
			name:      "banner: broken json expected to fail patching",
			args:      args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`broken json`),
			want:      nil,
		},
		{
			name:      "banner: generate bid id enabled",
			args:      args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: "randomID"},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something", "wurl":"http://localhost/event?t=win&b=randomID&a=123456&bidder=openx&ts=1234567890"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventTracking{
				enabledForAccount:  tt.args.enabledForAccount,
				enabledForRequest:  tt.args.enabledForRequest,
				accountID:          "123456",
				auctionTimestampMs: 1234567890,
				externalURL:        "http://localhost",
			}
			bid := &pbsOrtbBid{bid: &openrtb2.Bid{ID: "BID-1"}, bidType: tt.args.bidType, generatedBidID: tt.args.generatedBidId}
			modifiedJSON, err := evData.modifyBidJSON(bid, openrtb_ext.BidderOpenx, tt.jsonBytes)
			if tt.want != nil {
				assert.NoError(t, err, "Unexpected error")
				assert.JSONEq(t, string(tt.want), string(modifiedJSON))
			} else {
				assert.Error(t, err)
				assert.Equal(t, string(tt.jsonBytes), string(modifiedJSON), "Expected original json on failure to modify")
			}
		})
	}
}
