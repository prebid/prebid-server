package ctv

import (
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

// ICombination ...
type ICombination interface {
	Get() []int
}

// Combination ...
type Combination struct {
	ICombination
	data      []int
	generator PodDurationCombination
	config    *openrtb_ext.VideoAdPod
}

// NewCombination ... Generates on demand valid combinations
// Valid combinations are those who satisifies
//  1. Pod Min Max duration
//  2. minAds <= size(combination) <= maxads
//  3. If  Combination contains repeatition for given duration then
//     repeatitions are <= no of ads received for the duration
// Use Get method to start getting valid combinations
func NewCombination(buckets BidsBuckets, config *openrtb_ext.VideoAdPod) *Combination {
	generator := new(PodDurationCombination)
	durationBidsCnts := make([][2]uint64, 0)
	for duration, bids := range buckets {
		durationBidsCnts = append(durationBidsCnts, [2]uint64{uint64(duration), uint64(len(bids))})
	}
	generator.Init(config, durationBidsCnts)
	return &Combination{
		generator: *generator,
		config:    config,
	}
}

// Get next valid combination
// Retuns empty slice if all combinations are generated
func (c *Combination) Get() []int {
	nextComb := c.generator.Next()
	nextCombInt := make([]int, len(nextComb))
	cnt := 0
	for _, duration := range nextComb {
		nextCombInt[cnt] = int(duration)
		cnt++
	}
	return nextCombInt
}
