// Package impressions provides various algorithms to get the number of impressions
// along with minimum and maximum duration of each impression.
// It uses Ad pod request for it
package impressions

import (
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/util"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

// Algorithm indicates type of algorithms supported
// Currently it supports
//	1. MaximizeForDuration
//  2. MinMaxAlgorithm
type Algorithm int

const (
	// MaximizeForDuration algorithm tends towards Ad Pod Maximum Duration, Ad Slot Maximum Duration
	// and Maximum number of Ads. Accordingly it computes the number of impressions
	MaximizeForDuration Algorithm = iota
	// MinMaxAlgorithm algorithm ensures all possible impression breaks are plotted by considering
	// minimum as well as maxmimum durations and ads received in the ad pod request.
	// It computes number of impressions with following steps
	//  1. Passes input configuration as it is (Equivalent of MaximizeForDuration algorithm)
	//	2. Ad Pod Duration = Ad Pod Max Duration, Number of Ads = max ads
	//	3. Ad Pod Duration = Ad Pod Max Duration, Number of Ads = min ads
	//	4. Ad Pod Duration = Ad Pod Min Duration, Number of Ads = max ads
	//	5. Ad Pod Duration = Ad Pod Min Duration, Number of Ads = min ads
	MinMaxAlgorithm
)

// MonitorKey provides the unique key for moniroting the impressions algorithm
var MonitorKey = map[Algorithm]string{
	MaximizeForDuration: `a1_max`,
	MinMaxAlgorithm:     `a2_min_max`,
}

// Value use to compute Ad Slot Durations and Pod Durations for internal computation
// Right now this value is set to 5, based on passed data observations
// Observed that typically video impression contains contains minimum and maximum duration in multiples of  5
var multipleOf = int64(5)

// IImpressions ...
type IImpressions interface {
	Get() [][2]int64
	Algorithm() Algorithm // returns algorithm used for computing number of impressions
}

// NewImpressions generate object of impression generator
// based on input algorithm type
// if invalid algorithm type is passed, it returns default algorithm which will compute
// impressions based on minimum ad slot duration
func NewImpressions(podMinDuration, podMaxDuration int64, vPod *openrtb_ext.VideoAdPod, algorithm Algorithm) IImpressions {
	switch algorithm {
	case MaximizeForDuration:
		util.Logf("Selected 'MaximizeForDuration'")
		g := newMaximizeForDuration(podMinDuration, podMaxDuration, *vPod)
		return &g

	case MinMaxAlgorithm:
		util.Logf("Selected 'MinMaxAlgorithm'")
		g := newMinMaxAlgorithm(podMinDuration, podMaxDuration, *vPod)
		return &g
	}

	// return default algorithm with slot durations set to minimum slot duration
	util.Logf("Selected 'DefaultAlgorithm'")
	defaultGenerator := newConfig(podMinDuration, podMinDuration, openrtb_ext.VideoAdPod{
		MinAds:      vPod.MinAds,
		MaxAds:      vPod.MaxAds,
		MinDuration: vPod.MinDuration,
		MaxDuration: vPod.MinDuration, // sending slot minduration as max duration
	})
	return &defaultGenerator
}

// Duration indicates the position
// where the required min or max duration value can be found
// within given impression object
type Duration int

const (
	// MinDuration represents index value where we can get minimum duration of given impression object
	MinDuration Duration = iota
	// MaxDuration represents index value where we can get maximum duration of given impression object
	MaxDuration
)
