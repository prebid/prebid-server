package openrtb_ext

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var getIntPtr = func(m int) *int { return &m }

func Test_addMultiBid(t *testing.T) {
	maxBid0 := getIntPtr(0)
	maxBid1 := getIntPtr(1)
	maxBid2 := getIntPtr(2)
	maxBid3 := getIntPtr(3)
	maxBid9 := getIntPtr(9)
	maxBid10 := getIntPtr(10)

	type args struct {
		multiBidMap map[string]ExtMultiBid
		multiBid    *ExtMultiBid
	}
	tests := []struct {
		name            string
		args            args
		wantErrs        []error
		wantMultiBidMap map[string]ExtMultiBid
	}{
		{
			name: "MaxBids not defined",
			args: args{
				multiBidMap: map[string]ExtMultiBid{},
				multiBid:    &ExtMultiBid{},
			},
			wantErrs:        []error{fmt.Errorf("maxBid not defined %v", &ExtMultiBid{})},
			wantMultiBidMap: map[string]ExtMultiBid{},
		},
		{
			name: "Bidder or Bidders not defined",
			args: args{
				multiBidMap: map[string]ExtMultiBid{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2},
			},
			wantErrs:        []error{fmt.Errorf("bidder(s) not specified %v", &ExtMultiBid{MaxBids: maxBid2})},
			wantMultiBidMap: map[string]ExtMultiBid{},
		},
		{
			name: "Input bidder is already present in multibid",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {MaxBids: maxBid3, TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidder: "pubmatic"},
			},
			wantErrs:        []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", &ExtMultiBid{MaxBids: maxBid2, Bidder: "pubmatic"})},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {MaxBids: maxBid3, TargetBidderCodePrefix: "pm"}},
		},
		{
			name: "Bidder and Bidders both defined (only Bidder will be used)",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("ignoring bidders from %v", &ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Only Bidder defined",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Bidders defined where a bidder is already present in multibid",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}},
			},
			wantErrs:        []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}})},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}}},
		},
		{
			name: "Bidders defined along with TargetBidderCodePrefix",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("ignoring targetbiddercodeprefix for %v", &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidders: []string{"appnexus"}}},
		},
		{
			name: "MaxBids defined below minimum limit",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("using default maxBid minimum 1 limit %v", &ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid1, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "MaxBids defined over max limit",
			args: args{
				multiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("using default maxBid maximum 9 limit %v", &ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: map[string]ExtMultiBid{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid9, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErrs := addMultiBid(tt.args.multiBidMap, tt.args.multiBid)
			assert.Equal(t, tt.wantErrs, gotErrs)
			assert.Equal(t, tt.wantMultiBidMap, tt.args.multiBidMap)
		})
	}
}
