package ctv

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestCombination(t *testing.T) {
	buckets := make(BidsBuckets)

	dBids := make([]*Bid, 0)
	for i := 1; i <= 3; i++ {
		bid := new(Bid)
		bid.Duration = 10 * i
		dBids = append(dBids, bid)
		buckets[bid.Duration] = dBids
	}

	config := new(openrtb_ext.VideoAdPod)
	config.MinAds = new(int)
	*config.MinAds = 2
	config.MaxAds = new(int)
	*config.MaxAds = 4
	config.MinDuration = new(int)
	*config.MinDuration = 30
	config.MaxDuration = new(int)
	*config.MaxDuration = 70

	c := NewCombination(buckets, config)

	for true {
		comb := c.generator.Next()
		if nil == comb || len(comb) == 0 {
			assert.True(t, nil == comb || len(comb) == 0)
			break
		}
	}

}
