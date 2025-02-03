package openrtb_ext

import (
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

var maxBid1 = ptrutil.ToPtr(1)
var maxBid2 = ptrutil.ToPtr(2)
var maxBid3 = ptrutil.ToPtr(3)
var maxBid9 = ptrutil.ToPtr(9)

func TestValidateAndBuildExtMultiBid(t *testing.T) {
	var maxBid0 = ptrutil.ToPtr(0)
	var maxBid10 = ptrutil.ToPtr(10)

	type args struct {
		prebid *ExtRequestPrebid
	}
	tests := []struct {
		name                   string
		args                   args
		wantValidatedMultiBids []*ExtMultiBid
		wantErrs               []error
	}{
		{
			name:                   "prebid nil",
			wantValidatedMultiBids: nil,
			wantErrs:               nil,
		},
		{
			name: "prebid.MultiBid nil",
			args: args{
				prebid: &ExtRequestPrebid{MultiBid: nil},
			},
			wantValidatedMultiBids: nil,
			wantErrs:               nil,
		},
		{
			name: "prebid.MultiBid empty",
			args: args{
				prebid: &ExtRequestPrebid{MultiBid: make([]*ExtMultiBid, 0)},
			},
			wantValidatedMultiBids: nil,
			wantErrs:               nil,
		},
		{
			name: "MaxBids not defined",
			args: args{
				prebid: &ExtRequestPrebid{MultiBid: []*ExtMultiBid{{Bidder: "pubmatic"}}},
			},
			wantValidatedMultiBids: nil,
			wantErrs:               []error{fmt.Errorf("maxBids not defined for %v", ExtMultiBid{Bidder: "pubmatic"})},
		},
		{
			name: "Bidder or Bidders not defined",
			args: args{
				prebid: &ExtRequestPrebid{MultiBid: []*ExtMultiBid{{MaxBids: maxBid2}}},
			},
			wantValidatedMultiBids: nil,
			wantErrs:               []error{fmt.Errorf("bidder(s) not specified for %v", ExtMultiBid{MaxBids: maxBid2})},
		},
		{
			name: "Input bidder is already present in multibid",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{Bidder: "pubmatic", MaxBids: maxBid2, TargetBidderCodePrefix: "pm"},
						{Bidder: "pubmatic", MaxBids: maxBid3, TargetBidderCodePrefix: "pubm"},
					},
				},
			},
			wantErrs:               []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", ExtMultiBid{MaxBids: maxBid3, Bidder: "pubmatic", TargetBidderCodePrefix: "pubm"})},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "pubmatic", MaxBids: maxBid2, TargetBidderCodePrefix: "pm"}},
		},
		{
			name: "Bidder and Bidders both defined (only Bidder will be used)",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{Bidder: "pubmatic", MaxBids: maxBid2, TargetBidderCodePrefix: "pm"},
						{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"},
					},
				},
			},
			wantErrs:               []error{fmt.Errorf("ignoring bidders from %v", ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"})},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "pubmatic", MaxBids: maxBid2, TargetBidderCodePrefix: "pm"}, {Bidder: "appnexus", MaxBids: maxBid2, TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Only Bidder defined",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
					},
				},
			},
			wantErrs:               nil,
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "appnexus", MaxBids: maxBid2, TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Only Bidders defined",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{MaxBids: maxBid2, Bidders: []string{"appnexus", "someBidder"}},
					},
				},
			},
			wantErrs:               nil,
			wantValidatedMultiBids: []*ExtMultiBid{{Bidders: []string{"appnexus", "someBidder"}, MaxBids: maxBid2}},
		},
		{
			name: "Bidders defined where a bidder is already present in multibid",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{Bidder: "pubmatic", MaxBids: maxBid9, TargetBidderCodePrefix: "pm"},
						{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}},
					},
				},
			},
			wantErrs:               []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}})},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "pubmatic", MaxBids: maxBid9, TargetBidderCodePrefix: "pm"}, {Bidders: []string{"appnexus"}, MaxBids: maxBid2}},
		},
		{
			name: "Bidders defined where all the bidders are already present in multibid",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{Bidder: "pubmatic", MaxBids: maxBid9, TargetBidderCodePrefix: "pm"},
						{Bidders: []string{"appnexus"}, MaxBids: maxBid3},
						{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}},
					},
				},
			},
			wantErrs: []error{
				fmt.Errorf("multiBid already defined for appnexus, ignoring this instance %v", ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}}),
				fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}}),
			},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "pubmatic", MaxBids: maxBid9, TargetBidderCodePrefix: "pm"}, {Bidders: []string{"appnexus"}, MaxBids: maxBid3}},
		},
		{
			name: "Bidders defined along with TargetBidderCodePrefix",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"},
					},
				},
			},
			wantErrs:               []error{fmt.Errorf("ignoring targetbiddercodeprefix for %v", ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"})},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidders: []string{"appnexus"}, MaxBids: maxBid2}},
		},
		{
			name: "MaxBids defined below minimum limit",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
					},
				},
			},
			wantErrs:               []error{fmt.Errorf("invalid maxBids value, using minimum 1 limit for %v", ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "appnexus", TargetBidderCodePrefix: "appN", MaxBids: maxBid1}},
		},
		{
			name: "MaxBids defined over max limit",
			args: args{
				prebid: &ExtRequestPrebid{
					MultiBid: []*ExtMultiBid{
						{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
					},
				},
			},
			wantErrs:               []error{fmt.Errorf("invalid maxBids value, using maximum 9 limit for %v", ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantValidatedMultiBids: []*ExtMultiBid{{Bidder: "appnexus", TargetBidderCodePrefix: "appN", MaxBids: maxBid9}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValidatedMultiBids, gotErrs := ValidateAndBuildExtMultiBid(tt.args.prebid)
			assert.Equal(t, tt.wantValidatedMultiBids, gotValidatedMultiBids)
			assert.Equal(t, tt.wantErrs, gotErrs)
		})
	}
}

func Test_addMultiBid(t *testing.T) {
	var maxBid0 = ptrutil.ToPtr(0)
	var maxBid10 = ptrutil.ToPtr(10)

	type args struct {
		multiBidMap map[string]struct{}
		multiBid    *ExtMultiBid
	}
	tests := []struct {
		name          string
		args          args
		wantErrs      []error
		wantMultiBids []*ExtMultiBid
	}{
		{
			name: "MaxBids not defined",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{},
			},
			wantErrs:      []error{fmt.Errorf("maxBids not defined for %v", ExtMultiBid{})},
			wantMultiBids: nil,
		},
		{
			name: "Bidder or Bidders not defined",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2},
			},
			wantErrs:      []error{fmt.Errorf("bidder(s) not specified for %v", ExtMultiBid{MaxBids: maxBid2})},
			wantMultiBids: nil,
		},
		{
			name: "Input bidder is already present in multibid",
			args: args{
				multiBidMap: map[string]struct{}{"pubmatic": {}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidder: "pubmatic"},
			},
			wantErrs:      []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", ExtMultiBid{MaxBids: maxBid2, Bidder: "pubmatic"})},
			wantMultiBids: nil,
		},
		{
			name: "Bidder and Bidders both defined (only Bidder will be used)",
			args: args{
				multiBidMap: map[string]struct{}{"pubmatic": {}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"},
			},
			wantErrs:      []error{fmt.Errorf("ignoring bidders from %v", ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", Bidders: []string{"rubicon"}, TargetBidderCodePrefix: "appN"})},
			wantMultiBids: []*ExtMultiBid{{Bidder: "appnexus", MaxBids: maxBid2, TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Only Bidder defined",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:      []error{},
			wantMultiBids: []*ExtMultiBid{{Bidder: "appnexus", MaxBids: maxBid2, TargetBidderCodePrefix: "appN"}},
		},
		{
			name: "Only Bidders defined",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "someBidder"}},
			},
			wantErrs:      []error{},
			wantMultiBids: []*ExtMultiBid{{Bidders: []string{"appnexus", "someBidder"}, MaxBids: maxBid2}},
		},
		{
			name: "Bidders defined where a bidder is already present in multibid",
			args: args{
				multiBidMap: map[string]struct{}{"pubmatic": {}},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}},
			},
			wantErrs:      []error{fmt.Errorf("multiBid already defined for pubmatic, ignoring this instance %v", ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus", "pubmatic"}})},
			wantMultiBids: []*ExtMultiBid{{Bidders: []string{"appnexus"}, MaxBids: maxBid2}},
		},
		{
			name: "Bidders defined along with TargetBidderCodePrefix",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"},
			},
			wantErrs:      []error{fmt.Errorf("ignoring targetbiddercodeprefix for %v", ExtMultiBid{MaxBids: maxBid2, Bidders: []string{"appnexus"}, TargetBidderCodePrefix: "appN"})},
			wantMultiBids: []*ExtMultiBid{{Bidders: []string{"appnexus"}, MaxBids: maxBid2}},
		},
		{
			name: "MaxBids defined below minimum limit",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:      []error{fmt.Errorf("invalid maxBids value, using minimum 1 limit for %v", ExtMultiBid{MaxBids: maxBid0, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantMultiBids: []*ExtMultiBid{{Bidder: "appnexus", TargetBidderCodePrefix: "appN", MaxBids: maxBid1}},
		},
		{
			name: "MaxBids defined over max limit",
			args: args{
				multiBidMap: map[string]struct{}{},
				multiBid:    &ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"},
			},
			wantErrs:      []error{fmt.Errorf("invalid maxBids value, using maximum 9 limit for %v", ExtMultiBid{MaxBids: maxBid10, Bidder: "appnexus", TargetBidderCodePrefix: "appN"})},
			wantMultiBids: []*ExtMultiBid{{Bidder: "appnexus", TargetBidderCodePrefix: "appN", MaxBids: maxBid9}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMultiBids, gotErrs := addMultiBid(tt.args.multiBidMap, tt.args.multiBid)
			assert.Equal(t, tt.wantErrs, gotErrs)
			assert.Equal(t, tt.wantMultiBids, gotMultiBids)
		})
	}
}
