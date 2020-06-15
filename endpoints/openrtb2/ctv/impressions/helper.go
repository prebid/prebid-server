package impressions

import (
	"math"

	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

//  newConfig initializes the generator instance
func newConfig(podMinDuration, podMaxDuration int64, vPod openrtb_ext.VideoAdPod) generator {
	config := generator{}
	config.totalSlotTime = new(int64)
	// configure requested pod
	config.requested = pod{
		podMinDuration:  podMinDuration,
		podMaxDuration:  podMaxDuration,
		slotMinDuration: int64(*vPod.MinDuration),
		slotMaxDuration: int64(*vPod.MaxDuration),
		minAds:          int64(*vPod.MinAds),
		maxAds:          int64(*vPod.MaxAds),
	}

	// configure internal pod (FOR INTERNAL USE ONLY)
	// this pod is used for internal computation
	// and contains modified values of podMinDuration, podMaxDuration
	// slotMinDuration and slotMaxDuration in multiples of multipleOf factor
	// This function will by deault intialize this pod with same values
	// as of requestedPod
	// There is another function newConfigWithMultipleOf, which computes and assigns
	// values to this object
	config.internal = pod{
		podMinDuration:  config.requested.podMinDuration,
		podMaxDuration:  config.requested.podMaxDuration,
		slotMinDuration: config.requested.slotMinDuration,
		slotMaxDuration: config.requested.slotMaxDuration,
		minAds:          config.requested.minAds,
		maxAds:          config.requested.maxAds,
	}
	return config
}

// newConfigWithMultipleOf initializes the generator instance
// it internally calls newConfig to obtain the generator instance
// then it computes closed to factor basedon 'multipleOf' parameter value
// and accordingly determines the Pod Min/Max and Slot Min/Max values for internal
// computation only.
func newConfigWithMultipleOf(podMinDuration, podMaxDuration int64, vPod openrtb_ext.VideoAdPod, multipleOf int64) generator {
	config := newConfig(podMinDuration, podMaxDuration, vPod)

	// override the values of internalPod
	// config.internal

	if config.requested.podMinDuration == config.requested.podMaxDuration {
		/*TestCase 16*/
		ctv.Logf("requested.podMinDuration = requested.podMaxDuration = %v\n", config.requested.podMinDuration)
		config.internal.podMinDuration = config.requested.podMinDuration
		config.internal.podMaxDuration = config.requested.podMaxDuration
	} else {
		config.internal.podMinDuration = getClosestFactorForMinDuration(config.requested.podMinDuration, multipleOf)
		config.internal.podMaxDuration = getClosestFactorForMaxDuration(config.requested.podMaxDuration, multipleOf)
	}

	// if config.requestedSlotMinDuration == config.requestedSlotMaxDuration {
	if config.requested.slotMinDuration == config.requested.slotMaxDuration {
		/*TestCase 30*/
		ctv.Logf("requested.SlotMinDuration = requested.SlotMaxDuration = %v\n", config.requested.slotMinDuration)
		config.internal.slotMinDuration = config.requested.slotMinDuration
		config.internal.slotMaxDuration = config.requested.slotMaxDuration
	} else {
		config.internal.slotMinDuration = getClosestFactorForMinDuration(int64(config.requested.slotMinDuration), multipleOf)
		config.internal.slotMaxDuration = getClosestFactorForMaxDuration(int64(config.requested.slotMaxDuration), multipleOf)
	}
	return config
}

// Returns true if num is multipleof second argument. False otherwise
func isMultipleOf(num, multipleOf int64) bool {
	return math.Mod(float64(num), float64(multipleOf)) == 0
}

// Returns closest factor for num, with  respect  input multipleOf
//  Example: Closest Factor of 9, in multiples of 5 is '10'
func getClosestFactor(num, multipleOf int64) int64 {
	return int64(math.Round(float64(num)/float64(multipleOf)) * float64(multipleOf))
}

// Returns closestfactor of MinDuration, with  respect to multipleOf
// If computed factor < MinDuration then it will ensure and return
// close factor >=  MinDuration
func getClosestFactorForMinDuration(MinDuration int64, multipleOf int64) int64 {
	closedMinDuration := getClosestFactor(MinDuration, multipleOf)

	if closedMinDuration == 0 {
		return multipleOf
	}

	if closedMinDuration < MinDuration {
		return closedMinDuration + multipleOf
	}

	return closedMinDuration
}

// Returns closestfactor of maxduration, with  respect to multipleOf
// If computed factor > maxduration then it will ensure and return
// close factor <=  maxduration
func getClosestFactorForMaxDuration(maxduration, multipleOf int64) int64 {
	closedMaxDuration := getClosestFactor(maxduration, multipleOf)
	if closedMaxDuration == maxduration {
		return maxduration
	}

	// set closest maxduration closed to masduration
	for i := closedMaxDuration; i <= maxduration; {
		if closedMaxDuration < maxduration {
			closedMaxDuration = i + multipleOf
			i = closedMaxDuration
		}
	}

	if closedMaxDuration > maxduration {
		duration := closedMaxDuration - multipleOf
		if duration == 0 {
			// return input value as is instead of zero to avoid NPE
			return maxduration
		}
		return duration
	}

	return closedMaxDuration
}

// Returns Maximum number out off 2 input numbers
func max(num1, num2 int64) int64 {

	if num1 > num2 {
		return num1
	}

	if num2 > num1 {
		return num2
	}
	// both must be equal here
	return num1
}
