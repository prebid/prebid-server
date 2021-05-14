package combination

import (
	"testing"

	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/types"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestCombination(t *testing.T) {
	buckets := make(types.BidsBuckets)

	dBids := make([]*types.Bid, 0)
	for i := 1; i <= 3; i++ {
		bid := new(types.Bid)
		bid.Duration = 10 * i
		dBids = append(dBids, bid)
		buckets[bid.Duration] = dBids
	}

	config := new(openrtb_ext.VideoAdPod)
	config.MinAds = new(int)
	*config.MinAds = 2
	config.MaxAds = new(int)
	*config.MaxAds = 4

	c := NewCombination(buckets, 30, 70, config)

	for true {
		comb := c.generator.Next()
		if nil == comb || len(comb) == 0 {
			assert.True(t, nil == comb || len(comb) == 0)
			break
		}
	}

}
