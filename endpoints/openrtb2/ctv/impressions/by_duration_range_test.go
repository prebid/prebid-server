package impressions

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetImpressionsByDurationRanges(t *testing.T) {
	type args struct {
		policy        openrtb_ext.OWVideoLengthMatchingPolicy
		durations     []int
		maxAds        int
		adMinDuration int
		adMaxDuration int
	}
	type want struct {
		imps [][2]int64
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			// do not generate impressions
			name: "no_adpod_context",
			args: args{},
			want: want{
				imps: [][2]int64{},
			},
		},
		{
			// do not generate impressions
			name: "nil_durations",
			args: args{
				durations: nil,
			},
			want: want{
				imps: make([][2]int64, 0),
			},
		},
		{
			// do not generate impressions
			name: "empty_durations",
			args: args{
				durations: make([]int, 0),
			},
			want: want{
				imps: make([][2]int64, 0),
			},
		},
		{
			name: "zero_valid_durations_under_boundary",
			args: args{
				policy:        openrtb_ext.OWExactVideoLengthsMatching,
				durations:     []int{5, 10, 15},
				maxAds:        5,
				adMinDuration: 2,
				adMaxDuration: 2,
			},
			want: want{
				imps: [][2]int64{},
			},
		},
		{
			name: "zero_valid_durations_out_of_bound",
			args: args{
				policy:        openrtb_ext.OWExactVideoLengthsMatching,
				durations:     []int{5, 10, 15},
				maxAds:        5,
				adMinDuration: 20,
				adMaxDuration: 20,
			},
			want: want{
				imps: [][2]int64{},
			},
		},
		{
			name: "valid_durations_less_than_maxAds",
			args: args{
				policy:        openrtb_ext.OWExactVideoLengthsMatching,
				durations:     []int{5, 10, 15, 20, 25},
				maxAds:        5,
				adMinDuration: 10,
				adMaxDuration: 20,
			},
			want: want{
				imps: [][2]int64{
					{10, 10},
					{15, 15},
					{20, 20},
					//got repeated because of current video duration impressions are less than maxads
					{10, 10},
					{15, 15},
				},
			},
		},
		{
			name: "valid_durations_greater_than_maxAds",
			args: args{
				policy:        openrtb_ext.OWExactVideoLengthsMatching,
				durations:     []int{5, 10, 15, 20, 25},
				maxAds:        2,
				adMinDuration: 10,
				adMaxDuration: 20,
			},
			want: want{
				imps: [][2]int64{
					{10, 10},
					{15, 15},
					{20, 20},
				},
			},
		},
		{
			name: "roundup_policy_valid_durations",
			args: args{
				policy:        openrtb_ext.OWRoundupVideoLengthMatching,
				durations:     []int{5, 10, 15, 20, 25},
				maxAds:        5,
				adMinDuration: 10,
				adMaxDuration: 20,
			},
			want: want{
				imps: [][2]int64{
					{10, 10},
					{10, 15},
					{10, 20},
					{10, 10},
					{10, 15},
				},
			},
		},
		{
			name: "roundup_policy_zero_valid_durations",
			args: args{
				policy:        openrtb_ext.OWRoundupVideoLengthMatching,
				durations:     []int{5, 10, 15, 20, 25},
				maxAds:        5,
				adMinDuration: 30,
				adMaxDuration: 30,
			},
			want: want{
				imps: [][2]int64{},
			},
		},
		{
			name: "roundup_policy_valid_max_ads_more_than_max_ads",
			args: args{
				policy:        openrtb_ext.OWRoundupVideoLengthMatching,
				durations:     []int{5, 10, 15, 20, 25},
				maxAds:        2,
				adMinDuration: 10,
				adMaxDuration: 20,
			},
			want: want{
				imps: [][2]int64{
					{10, 10},
					{10, 15},
					{10, 20},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			gen := newByDurationRanges(args.policy, args.durations, args.maxAds, args.adMinDuration, args.adMaxDuration)
			imps := gen.Get()
			assert.Equal(t, tt.want.imps, imps)
		})
	}
}
