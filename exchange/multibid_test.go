package exchange

import (
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var getIntPtr = func(m int) *int { return &m }

func TestExtMultiBidMap_Add(t *testing.T) {
	maxBid0 := getIntPtr(0)
	maxBid1 := getIntPtr(1)
	maxBid2 := getIntPtr(2)
	maxBid3 := getIntPtr(3)
	maxBid9 := getIntPtr(9)
	maxBid10 := getIntPtr(10)

	type args struct {
		multiBid *openrtb_ext.ExtMultiBid
	}
	tests := []struct {
		name            string
		mb              *ExtMultiBidMap
		args            args
		wantErrs        []error
		wantMultiBidMap *ExtMultiBidMap
	}{
		{
			name: "MaxBids not defined",
			mb:   &ExtMultiBidMap{},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{},
			},
			wantErrs:        []error{fmt.Errorf("maxBid not defined %v", &openrtb_ext.ExtMultiBid{})},
			wantMultiBidMap: &ExtMultiBidMap{},
		},
		{
			name: "Bidder or Bidders not defined",
			mb:   &ExtMultiBidMap{},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid2},
			},
			wantErrs:        []error{fmt.Errorf("bidder(s) not specified %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid2})},
			wantMultiBidMap: &ExtMultiBidMap{},
		},
		{
			name: "Input bidder is already present in multibid",
			mb:   &ExtMultiBidMap{"pubmatic": {MaxBids: maxBid3, TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidder: "pubmatic"},
			},
			wantErrs:        []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidder: "pubmatic"})},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {MaxBids: maxBid3, TargetBidderCodePrefix: "pm"}},
		},
		{
			name: "Bidder and Bidders both defined (only Bidder will be used)",
			mb:   &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("ignoring bidders from %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Only Bidder defined",
			mb:   &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Bidders defined where a bidder is already present in multibid",
			mb:   &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}},
			},
			wantErrs:        []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}})},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}}},
		},
		{
			name: "Bidders defined along with TargetBidderCodePrefix",
			mb:   &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("ignoring targetbiddercodeprefix for %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid2, Bidders: []string{"appnexus"}}},
		},
		{
			name: "MaxBids defined below minimum limit",
			mb:   &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("using default maxBid minimum 1 limit %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid1, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "MaxBids defined over max limit",
			mb:   &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}},
			args: args{
				multiBid: &openrtb_ext.ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:        []error{fmt.Errorf("using default maxBid maximum 9 limit %v", &openrtb_ext.ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantMultiBidMap: &ExtMultiBidMap{"pubmatic": {TargetBidderCodePrefix: "pm"}, "appnexus": {MaxBids: maxBid9, Bidder: "appnexus", TargetBidderCodePrefix: "appN"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErrs := tt.mb.Add(tt.args.multiBid)
			assert.Equal(t, tt.wantErrs, gotErrs)
			assert.Equal(t, tt.wantMultiBidMap, tt.mb)
		})
	}
}

func TestExtMultiBidMap_GetMaxBids(t *testing.T) {
	type args struct {
		bidder string
	}
	tests := []struct {
		name string
		mb   *ExtMultiBidMap
		args args
		want int
	}{
		{
			name: "return default 1 for non multibid bidders",
			mb:   &ExtMultiBidMap{},
			args: args{bidder: "pubmatic"},
			want: 1,
		},
		{
			name: "return maxBid when bidder is present in multibid",
			mb: &ExtMultiBidMap{
				"pubmatic": {MaxBids: getIntPtr(5)},
			},
			args: args{bidder: "pubmatic"},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mb.GetMaxBids(tt.args.bidder); got != tt.want {
				t.Errorf("ExtMultiBidMap.GetMaxBids() = %v, want %v", got, tt.want)
			}
		})
	}
}
