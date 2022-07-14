package impressions

import (
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/util"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// byDurRangeConfig struct will be used for creating impressions object based on list of duration ranges
type byDurRangeConfig struct {
	IImpressions                                          //IImpressions interface
	policy        openrtb_ext.OWVideoLengthMatchingPolicy //duration matching algorithm round/exact
	durations     []int                                   //durations list of durations in seconds used for creating impressions object
	maxAds        int                                     //maxAds is number of max impressions can be created
	adMinDuration int                                     //adpod slot mininum duration
	adMaxDuration int                                     //adpod slot maximum duration
}

// newByDurationRanges will create new object ob byDurRangeConfig for creating impressions for adpod request
func newByDurationRanges(policy openrtb_ext.OWVideoLengthMatchingPolicy, durations []int,
	maxAds, adMinDuration, adMaxDuration int) byDurRangeConfig {

	return byDurRangeConfig{
		policy:        policy,
		durations:     durations,
		maxAds:        maxAds,
		adMinDuration: adMinDuration,
		adMaxDuration: adMaxDuration,
	}
}

// Get function returns lists of min,max duration ranges ganerated based on durations
// it will return valid durations, duration must be within podMinDuration and podMaxDuration range
// if len(durations) < maxAds then clone valid durations from starting till we reach maxAds length
func (c *byDurRangeConfig) Get() [][2]int64 {
	if len(c.durations) == 0 {
		util.Logf("durations is nil. [%v] algorithm returning not generated impressions", c.Algorithm())
		return make([][2]int64, 0)
	}

	isRoundupDurationMatchingPolicy := (openrtb_ext.OWRoundupVideoLengthMatching == c.policy)
	var minDuration = -1
	var validDurations []int

	for _, dur := range c.durations {
		// validate durations (adminduration <= lineitemduration <= admaxduration) (adpod adslot min and max duration)
		if !(c.adMinDuration <= dur && dur <= c.adMaxDuration) {
			continue // invalid duration
		}

		// finding minimum duration for roundup policy, this may include valid or invalid duration
		if isRoundupDurationMatchingPolicy && (minDuration == -1 || minDuration >= dur) {
			minDuration = dur
		}

		validDurations = append(validDurations, dur)
	}

	imps := make([][2]int64, 0)
	for _, dur := range validDurations {
		/*
			minimum value is depends on duration matching policy
			openrtb_ext.OWAdPodRoundupDurationMatching (round): minduration would be min(duration)
			openrtb_ext.OWAdPodExactDurationMatching (exact) or empty: minduration would be same as maxduration
		*/
		if isRoundupDurationMatchingPolicy {
			imps = append(imps, [2]int64{int64(minDuration), int64(dur)})
		} else {
			imps = append(imps, [2]int64{int64(dur), int64(dur)})
		}
	}

	//calculate max ads
	maxAds := c.maxAds
	if len(validDurations) > maxAds {
		maxAds = len(validDurations)
	}

	//adding extra impressions incase of total impressions generated are less than pod max ads.
	if len(imps) > 0 {
		for i := 0; len(imps) < maxAds; i++ {
			imps = append(imps, [2]int64{imps[i][0], imps[i][1]})
		}
	}

	return imps
}

// Algorithm returns MinMaxAlgorithm
func (c *byDurRangeConfig) Algorithm() Algorithm {
	return ByDurationRanges
}
