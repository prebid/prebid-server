package ctv

/********************* AdPodGenerator Functions *********************/

//IAdPodGenerator interface for generating AdPod from Ads
type IAdPodGenerator interface {
	GetAdPodBids() []*Bid
}

//AdPodGenerator AdPodGenerator
type AdPodGenerator struct {
	IAdPodGenerator
	buckets BidsBuckets
	comb    ICombination
}

//NewAdPodGenerator will generate adpod based on configuration
func NewAdPodGenerator(buckets BidsBuckets, comb ICombination) *AdPodGenerator {
	return &AdPodGenerator{
		buckets: buckets,
		comb:    comb,
	}
}

//GetAdPodBids will return Adpod based on configurations
func (o *AdPodGenerator) GetAdPodBids() []*Bid {
	durations := o.comb.Get()
	result := make([]*Bid, len(durations))

	indices := map[int]int{}

	//Init all to 0
	for _, duration := range durations {
		indices[duration] = 0
	}

	for i, duration := range durations {
		bids := o.buckets[duration]
		index := indices[duration]
		if index > len(bids) {
			index = 0
		}
		result[i] = bids[index]
		indices[duration]++
	}
	return result[:]
}
