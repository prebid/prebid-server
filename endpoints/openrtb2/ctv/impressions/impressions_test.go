// Package impressions provides various algorithms to get the number of impressions
// along with minimum and maximum duration of each impression.
// It uses Ad pod request for it
package impressions

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestSelectAlgorithm(t *testing.T) {
	type args struct {
		reqAdPod *openrtb_ext.ExtRequestAdPod
	}
	tests := []struct {
		name string
		args args
		want Algorithm
	}{
		{
			name: "default",
			args: args{},
			want: MinMaxAlgorithm,
		},
		{
			name: "missing_videolengths",
			args: args{reqAdPod: &openrtb_ext.ExtRequestAdPod{}},
			want: MinMaxAlgorithm,
		},
		{
			name: "roundup_matching_algo",
			args: args{reqAdPod: &openrtb_ext.ExtRequestAdPod{
				VideoLengths:        []int{15, 20},
				VideoLengthMatching: openrtb_ext.OWRoundupVideoLengthMatching,
			}},
			want: ByDurationRanges,
		},
		{
			name: "exact_matching_algo",
			args: args{reqAdPod: &openrtb_ext.ExtRequestAdPod{
				VideoLengths:        []int{15, 20},
				VideoLengthMatching: openrtb_ext.OWExactVideoLengthsMatching,
			}},
			want: ByDurationRanges,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectAlgorithm(tt.args.reqAdPod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewImpressions(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	type args struct {
		podMinDuration int64
		podMaxDuration int64
		reqAdPod       *openrtb_ext.ExtRequestAdPod
		vPod           *openrtb_ext.VideoAdPod
		algorithm      Algorithm
	}
	tests := []struct {
		name string
		args args
		want Algorithm
	}{
		{
			name: "Default-MaximizeForDuration",
			args: args{
				podMinDuration: 15,
				podMaxDuration: 90,
				reqAdPod:       &openrtb_ext.ExtRequestAdPod{},
				vPod: &openrtb_ext.VideoAdPod{
					MinAds:      intPtr(1),
					MaxAds:      intPtr(2),
					MinDuration: intPtr(5),
					MaxDuration: intPtr(10),
				},
				algorithm: Algorithm(-1),
			},
			want: MaximizeForDuration,
		},
		{
			name: "MaximizeForDuration",
			args: args{
				podMinDuration: 15,
				podMaxDuration: 90,
				reqAdPod:       &openrtb_ext.ExtRequestAdPod{},
				vPod: &openrtb_ext.VideoAdPod{
					MinAds:      intPtr(1),
					MaxAds:      intPtr(2),
					MinDuration: intPtr(5),
					MaxDuration: intPtr(10),
				},
				algorithm: MaximizeForDuration,
			},
			want: MaximizeForDuration,
		},
		{
			name: "MinMaxAlgorithm",
			args: args{
				podMinDuration: 15,
				podMaxDuration: 90,
				reqAdPod:       &openrtb_ext.ExtRequestAdPod{},
				vPod: &openrtb_ext.VideoAdPod{
					MinAds:      intPtr(1),
					MaxAds:      intPtr(2),
					MinDuration: intPtr(5),
					MaxDuration: intPtr(10),
				},
				algorithm: MinMaxAlgorithm,
			},
			want: MinMaxAlgorithm,
		},
		{
			name: "ByDurationRanges",
			args: args{
				podMinDuration: 15,
				podMaxDuration: 90,
				reqAdPod: &openrtb_ext.ExtRequestAdPod{
					VideoLengths: []int{10, 15},
				},
				vPod: &openrtb_ext.VideoAdPod{
					MinAds:      intPtr(1),
					MaxAds:      intPtr(2),
					MinDuration: intPtr(5),
					MaxDuration: intPtr(10),
				},
				algorithm: ByDurationRanges,
			},
			want: ByDurationRanges,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewImpressions(tt.args.podMinDuration, tt.args.podMaxDuration, tt.args.reqAdPod, tt.args.vPod, tt.args.algorithm)
			assert.Equal(t, tt.want, got.Algorithm())
		})
	}
}
