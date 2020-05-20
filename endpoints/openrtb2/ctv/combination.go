package ctv

import "github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"

type ICombination interface {
	Get() []int
}

type Combination struct {
	ICombination
	data   []int
	config *openrtb_ext.VideoAdPod
}

func NewCombination(data []int, config *openrtb_ext.VideoAdPod) *Combination {
	return &Combination{
		data:   data[:],
		config: config,
	}
}

func (c *Combination) Get() []int {
	return c.data[:]
}
