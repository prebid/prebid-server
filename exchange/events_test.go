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
		bid               *openrtb.Bid
		bidderName        openrtb_ext.BidderName
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.ExtBidPrebidEvents
	}{
		{
			name: "Events enabled for request, disabled for account",
			args: args{
				enabledForAccount: false,
				enabledForRequest: true,
				bid:               &openrtb.Bid{ID: "BID-1"},
				bidderName:        openrtb_ext.BidderOpenx,
			},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "Events enabled for account, disabled for request",
			args: args{
				enabledForAccount: false,
				enabledForRequest: true,
				bid:               &openrtb.Bid{ID: "BID-1"},
				bidderName:        openrtb_ext.BidderOpenx,
			},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "Events disabled for account and request",
			args: args{
				enabledForAccount: false,
				enabledForRequest: false,
				bid:               &openrtb.Bid{ID: "BID-1"},
				bidderName:        openrtb_ext.BidderOpenx,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventsData{
				enabledForAccount:  tt.args.enabledForAccount,
				enabledForRequest:  tt.args.enabledForRequest,
				accountID:          "123456",
				auctionTimestampMs: 1234567890,
				externalURL:        "http://localhost",
			}
			assert.Equal(t, tt.want, evData.makeBidExtEvents(tt.args.bid, tt.args.bidderName))
		})
	}
}

func Test_eventsData_modifyBidJSON(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bid               *openrtb.Bid
		bidderName        openrtb_ext.BidderName
		jsonBytes         []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Events enabled for request, disabled for account",
			args: args{
				enabledForAccount: false,
				enabledForRequest: true,
				bid:               &openrtb.Bid{ID: "BID-1"},
				bidderName:        openrtb_ext.BidderOpenx,
				jsonBytes:         []byte(`{"ID": "something"}`),
			},
			want: []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name: "Events enabled for account, disabled for request",
			args: args{
				enabledForAccount: false,
				enabledForRequest: true,
				bid:               &openrtb.Bid{ID: "BID-1"},
				bidderName:        openrtb_ext.BidderOpenx,
				jsonBytes:         []byte(`{"ID": "something"}`),
			},
			want: []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name: "Events disabled for account and request",
			args: args{
				enabledForAccount: false,
				enabledForRequest: false,
				bid:               &openrtb.Bid{ID: "BID-1"},
				bidderName:        openrtb_ext.BidderOpenx,
				jsonBytes:         []byte(`{"ID": "something"}`),
			},
			want: []byte(`{"ID": "something"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventsData{
				enabledForAccount:  tt.args.enabledForAccount,
				enabledForRequest:  tt.args.enabledForRequest,
				accountID:          "123456",
				auctionTimestampMs: 1234567890,
				externalURL:        "http://localhost",
			}
			assert.JSONEq(t, string(tt.want), string(evData.modifyBidJSON(tt.args.bid, tt.args.bidderName, tt.args.jsonBytes)))
		})
	}
}
