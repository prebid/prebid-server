package impressions

import (
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

// newMaximizeForDuration Constucts the generator object from openrtb_ext.VideoAdPod
// It computes durations for Ad Slot and Ad Pod in multiple of X
func newMaximizeForDuration(podMinDuration, podMaxDuration int64, vPod openrtb_ext.VideoAdPod) generator {
	config := newConfigWithMultipleOf(podMinDuration, podMaxDuration, vPod, multipleOf)

	ctv.Logf("Computed podMinDuration = %v in multiples of %v (requestedPodMinDuration = %v)\n", config.internal.podMinDuration, multipleOf, config.requested.podMinDuration)
	ctv.Logf("Computed podMaxDuration = %v in multiples of %v (requestedPodMaxDuration = %v)\n", config.internal.podMaxDuration, multipleOf, config.requested.podMaxDuration)
	ctv.Logf("Computed slotMinDuration = %v in multiples of %v (requestedSlotMinDuration = %v)\n", config.internal.slotMinDuration, multipleOf, config.requested.slotMinDuration)
	ctv.Logf("Computed slotMaxDuration = %v in multiples of %v (requestedSlotMaxDuration = %v)\n", config.internal.slotMaxDuration, multipleOf, *vPod.MaxDuration)
	ctv.Logf("Requested minAds = %v\n", config.requested.minAds)
	ctv.Logf("Requested maxAds = %v\n", config.requested.maxAds)
	return config
}

// Algorithm returns MaximizeForDuration
func (config generator) Algorithm() Algorithm {
	return MaximizeForDuration
}
