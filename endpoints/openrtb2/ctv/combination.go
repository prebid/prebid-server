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
	generator PodDurationCombination
	config    *openrtb_ext.VideoAdPod
}

// NewCombination ...
func NewCombination(data []int, config *openrtb_ext.VideoAdPod) *Combination {
	generator := new(PodDurationCombination)
	generator.Init(config, nil)
	return &Combination{
		generator: *generator,
		config:    config,
	}
}

// Get next valid combination
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
