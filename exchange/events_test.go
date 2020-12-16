package exchange

import (
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func Test_eventsData_makeBidExtEvents(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bidType           openrtb_ext.BidType
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.ExtBidPrebidEvents
	}{
		{
			name: "banner: events enabled for request, disabled for account",
			args: args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "banner: events enabled for account, disabled for request",
			args: args{enabledForAccount: true, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "banner: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner},
			want: nil,
		},
		{
			name: "video: events enabled for account and request",
			args: args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeVideo},
			want: nil,
		},
		{
			name: "video: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeVideo},
			want: nil,
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
			bid := &pbsOrtbBid{bid: &openrtb.Bid{ID: "BID-1"}, bidType: tt.args.bidType}
			assert.Equal(t, tt.want, evData.makeBidExtEvents(bid, openrtb_ext.BidderOpenx))
		})
	}
}

func Test_eventsData_modifyBidJSON(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bidType           openrtb_ext.BidType
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "banner: events enabled for request, disabled for account",
			args: args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner},
			want: []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name: "banner: events enabled for account, disabled for request",
			args: args{enabledForAccount: true, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner},
			want: []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name: "banner: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner},
			want: []byte(`{"ID": "something"}`),
		},
		{
			name: "video: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeVideo},
			want: []byte(`{"ID": "something"}`),
		},
		{
			name: "video: events enabled for account and request",
			args: args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeVideo},
			want: []byte(`{"ID": "something"}`),
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
			bid := &pbsOrtbBid{bid: &openrtb.Bid{ID: "BID-1"}, bidType: tt.args.bidType}
			jsonBytes := []byte(`{"ID": "something"}`)
			assert.JSONEq(t, string(tt.want), string(evData.modifyBidJSON(bid, openrtb_ext.BidderOpenx, jsonBytes)))
		})
	}
}
