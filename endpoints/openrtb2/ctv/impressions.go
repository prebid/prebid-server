// Package ctv provides functionalities for handling CTV specific Request  and responses
package ctv

import (
	"log"
	"math"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

// adPodConfig contains Pod Minimum Duration, Pod Maximum Duration, Slot Minimum Duration and Slot Maximum Duration
// It holds additional attributes required by this algorithm for  internal computation.
// 	It contains Slots attribute. This  attribute holds the output of this algorithm
type adPodConfig struct {
	minAds          int64 // Minimum number of Ads / Slots allowed inside Ad Pod
	maxAds          int64 // Maximum number of Ads / Slots allowed inside Ad Pod.
	slotMinDuration int64 // Minimum duration (in seconds) for each Ad Slot inside Ad Pod. It is not original value from request. It holds the value closed to original value and multiples of X.
	slotMaxDuration int64 // Maximum duration (in seconds) for each Ad Slot inside Ad Pod. It is not original value from request. It holds the value closed to original value and multiples of X.
	podMinDuration  int64 // Minimum total duration (in seconds) of Ad Pod. It is not original value from request. It holds the value closed to original value and multiples of X.
	podMaxDuration  int64 // Maximum total duration (in seconds) of Ad Pod. It is not original value from request. It holds the value closed to original value and multiples of X.

	requestedPodMinDuration  int64      // Requested Ad Pod minimum duration (in seconds)
	requestedPodMaxDuration  int64      // Requested Ad Pod maximum duration (in seconds)
	requestedSlotMinDuration int64      // Requested Ad Slot minimum duration (in seconds)
	requestedSlotMaxDuration int64      // Requested Ad Slot maximum duration (in seconds)
	Slots                    [][2]int64 // Holds Minimum and Maximum duration (in seconds) for each Ad Slot. Length indicates total number of Ad Slots/ Impressions for given Ad Pod
	totalSlotTime            *int64     // Total Sum of all Ad Slot durations (in seconds)
	freeTime                 int64      // Remaining Time (in seconds) not allocated. It is compared with RequestedPodMaxDuration
	slotsWithZeroTime        *int64     // Indicates number of slots with zero time (starting from 1).
}

// Value use to compute Ad Slot Durations and Pod Durations for internal computation
// Right now this value is set to 5, based on passed data observations
// Observed that typically video impression contains contains minimum and maximum duration in multiples of  5
var multipleOf = int64(5)

// Constucts the adPodConfig object from openrtb_ext.VideoAdPod
// It computes durations for Ad Slot and Ad Pod in multiple of X
func init0(podMinDuration, podMaxDuration int64, vPod openrtb_ext.VideoAdPod) adPodConfig {
	config := adPodConfig{}
	config.requestedPodMinDuration = podMinDuration
	config.requestedPodMaxDuration = podMaxDuration
	config.requestedSlotMinDuration = int64(*vPod.MinDuration)
	config.requestedSlotMaxDuration = int64(*vPod.MaxDuration)
	if config.requestedPodMinDuration == config.requestedPodMaxDuration {
		/*TestCase 16*/
		Logf("requestedPodMinDuration = requestedPodMaxDuration = %v\n", config.requestedPodMinDuration)
		config.podMinDuration = getClosetFactor(config.requestedPodMinDuration, multipleOf)
		config.podMaxDuration = config.podMinDuration
	} else {
		config.podMinDuration = getClosetFactorForMinDuration(config.requestedPodMinDuration, multipleOf)
		config.podMaxDuration = getClosetFactorForMaxDuration(config.requestedPodMaxDuration, multipleOf)
	}

	if config.requestedSlotMinDuration == config.requestedSlotMaxDuration {
		/*TestCase 30*/
		config.slotMinDuration = getClosetFactor(config.requestedSlotMinDuration, multipleOf)
		config.slotMaxDuration = config.slotMinDuration
	} else {
		config.slotMinDuration = getClosetFactorForMinDuration(int64(config.requestedSlotMinDuration), multipleOf)
		config.slotMaxDuration = getClosetFactorForMaxDuration(int64(*vPod.MaxDuration), multipleOf)
	}
	config.minAds = int64(*vPod.MinAds)
	config.maxAds = int64(*vPod.MaxAds)
	config.totalSlotTime = new(int64)

	Logf("Computed podMinDuration = %v in multiples of %v (requestedPodMinDuration = %v)\n", config.podMinDuration, multipleOf, config.requestedPodMinDuration)
	Logf("Computed podMaxDuration = %v in multiples of %v (requestedPodMaxDuration = %v)\n", config.podMaxDuration, multipleOf, config.requestedPodMaxDuration)
	Logf("Computed slotMinDuration = %v in multiples of %v (requestedSlotMinDuration = %v)\n", config.slotMinDuration, multipleOf, config.requestedSlotMinDuration)
	Logf("Computed slotMaxDuration = %v in multiples of %v (requestedSlotMaxDuration = %v)\n", config.slotMaxDuration, multipleOf, *vPod.MaxDuration)
	Logf("Requested minAds = %v\n", config.minAds)
	Logf("Requested maxAds = %v\n", config.maxAds)

	return config
}

// GetImpressions Returns the number of Ad Slots/Impression  that input Ad Pod can have.
// It also returs Minimum and  Maximum duration. Dimension 1, represents Minimum duration. Dimension 2, represents Maximum Duration
// for each Ad Slot.
// Minimum Duratiuon can contain either RequestedSlotMinDuration or Duration computed by algorithm for the Ad Slot
// Maximum Duration only contains Duration computed by algorithm for the Ad Slot
// podMinDuration - Minimum duration of Pod, podMaxDuration Maximum duration of Pod, vPod Video Pod Object
func GetImpressions(podMinDuration, podMaxDuration int64, vPod openrtb_ext.VideoAdPod) [][2]int64 {
	_, imps := getImpressions(podMinDuration, podMaxDuration, vPod)
	return imps
}

// getImpressions Returns the adPodConfig and number of Ad Slots/Impression  that input Ad Pod can have.
// It also returs Minimum and  Maximum duration. Dimension 1, represents Minimum duration. Dimension 2, represents Maximum Duration
// for each Ad Slot.
// Minimum Duratiuon can contain either RequestedSlotMinDuration or Duration computed by algorithm for the Ad Slot
// Maximum Duration only contains Duration computed by algorithm for the Ad Slot
// podMinDuration - Minimum duration of Pod, podMaxDuration Maximum duration of Pod, vPod Video Pod Object
func getImpressions(podMinDuration, podMaxDuration int64, vPod openrtb_ext.VideoAdPod) (adPodConfig, [][2]int64) {

	cfg := init0(podMinDuration, podMaxDuration, vPod)
	Logf("Pod Config with Internal Computation (using multiples of %v) = %+v\n", multipleOf, cfg)
	totalAds := computeTotalAds(cfg)
	timeForEachSlot := computeTimeForEachAdSlot(cfg, totalAds)

	cfg.Slots = make([][2]int64, totalAds)
	cfg.slotsWithZeroTime = new(int64)
	*cfg.slotsWithZeroTime = totalAds
	Logf("Plotted Ad Slots / Impressions of size = %v\n", len(cfg.Slots))
	// iterate over total time till it is < cfg.RequestedPodMaxDuration
	time := int64(0)
	Logf("Started allocating durations to each Ad Slot / Impression\n")
	fillZeroSlotsOnPriority := true
	noOfZeroSlotsFilledByLastRun := int64(0)
	for time < cfg.requestedPodMaxDuration {
		adjustedTime, slotsFull := cfg.addTime(timeForEachSlot, fillZeroSlotsOnPriority)
		time += adjustedTime
		timeForEachSlot = computeTimeLeastValue(cfg.requestedPodMaxDuration - time)
		if slotsFull {
			Logf("All slots are full of their capacity. validating slots\n")
			break
		}

		// instruct for filling zero capacity slots on priority if
		// 1. shouldAdjustSlotWithZeroDuration returns true
		// 2. there are slots with 0 duration
		// 3. there is at least ont slot with zero duration filled by last iteration
		fillZeroSlotsOnPriority = false
		noOfZeroSlotsFilledByLastRun = *cfg.slotsWithZeroTime - noOfZeroSlotsFilledByLastRun
		if cfg.shouldAdjustSlotWithZeroDuration() && *cfg.slotsWithZeroTime > 0 && noOfZeroSlotsFilledByLastRun > 0 {
			fillZeroSlotsOnPriority = true
		}
	}
	Logf("Completed allocating durations to each Ad Slot / Impression\n")

	// validate slots
	cfg.validateSlots()

	// log free time if present to stats server
	// also check algoritm computed the no. of ads
	if cfg.requestedPodMaxDuration-time > 0 && len(cfg.Slots) > 0 {
		cfg.freeTime = cfg.requestedPodMaxDuration - time
		log.Println("TO STATS SERVER : Free Time not allocated ", cfg.freeTime, "sec")
	}

	Logf("\nTotal Impressions = %v, Total Allocated Time = %v sec (out of %v sec, Max Pod Duration)\n%v", len(cfg.Slots), *cfg.totalSlotTime, cfg.requestedPodMaxDuration, cfg.Slots)
	return cfg, cfg.Slots
}

// Returns total number of Ad Slots/ impressions that the Ad Pod can have
func computeTotalAds(cfg adPodConfig) int64 {
	if cfg.slotMaxDuration <= 0 || cfg.slotMinDuration <= 0 {
		Logf("Either cfg.slotMaxDuration or cfg.slotMinDuration or both are <= 0. Hence, totalAds = 0")
		return 0
	}
	maxAds := cfg.podMaxDuration / cfg.slotMaxDuration
	minAds := cfg.podMaxDuration / cfg.slotMinDuration

	Logf("Computed minAds = %v , maxAds = %v\n", minAds, maxAds)

	totalAds := max(minAds, maxAds)
	Logf("Computed max(minAds, maxAds) = totalAds = %v\n", totalAds)

	if totalAds < cfg.minAds {
		totalAds = cfg.minAds
		Logf("Computed totalAds < requested  minAds (%v). Hence, setting totalAds =  minAds = %v\n", cfg.minAds, totalAds)
	}
	if totalAds > cfg.maxAds {
		totalAds = cfg.maxAds
		Logf("Computed totalAds > requested  maxAds (%v). Hence, setting totalAds =  maxAds = %v\n", cfg.maxAds, totalAds)
	}
	Logf("Computed Final totalAds = %v  [%v <= %v <= %v]\n", totalAds, cfg.minAds, totalAds, cfg.maxAds)
	return totalAds
}

// Returns duration in seconds that can be allocated to each Ad Slot
// Accepts cfg containing algorithm configurations and totalAds containing Total number of
// Ad Slots / Impressions that the Ad Pod can have.
func computeTimeForEachAdSlot(cfg adPodConfig, totalAds int64) int64 {
	// Compute time for each ad
	if totalAds <= 0 {
		Logf("totalAds = 0, Hence timeForEachSlot = 0")
		return 0
	}
	timeForEachSlot := cfg.podMaxDuration / totalAds

	Logf("Computed timeForEachSlot = %v (podMaxDuration/totalAds) (%v/%v)\n", timeForEachSlot, cfg.podMaxDuration, totalAds)

	if timeForEachSlot < cfg.slotMinDuration {
		timeForEachSlot = cfg.slotMinDuration
		Logf("Computed timeForEachSlot < requested  slotMinDuration (%v). Hence, setting timeForEachSlot =  slotMinDuration = %v\n", cfg.slotMinDuration, timeForEachSlot)
	}

	if timeForEachSlot > cfg.slotMaxDuration {
		timeForEachSlot = cfg.slotMaxDuration
		Logf("Computed timeForEachSlot > requested  slotMaxDuration (%v). Hence, setting timeForEachSlot =  slotMaxDuration = %v\n", cfg.slotMaxDuration, timeForEachSlot)
	}

	// ensure timeForEachSlot is multipleof given number
	if !isMultipleOf(timeForEachSlot, multipleOf) {
		// get close to value of multiple
		// here we muse get either cfg.SlotMinDuration or cfg.SlotMaxDuration
		// these values are already pre-computed in multiples of given number
		timeForEachSlot = getClosetFactor(timeForEachSlot, multipleOf)
		Logf("Computed closet factor %v, in multiples of %v for timeForEachSlot\n", timeForEachSlot, multipleOf)
	}
	Logf("Computed Final timeForEachSlot = %v  [%v <= %v <= %v]\n", timeForEachSlot, cfg.requestedSlotMinDuration, timeForEachSlot, cfg.requestedSlotMaxDuration)
	return timeForEachSlot
}

// Checks if multipleOf can be used as least time value
// this will ensure eack slot to maximize its time if possible
// if multipleOf can not be used as least value then default input value is returned as is
// accepts time containing, which least value to be computed.
// Returns the least value based on multiple of X
func computeTimeLeastValue(time int64) int64 {
	// time if Testcase#6
	// 1. multiple of x - get smallest factor N of multiple of x for time
	// 2. not multiple of x - try to obtain smallet no N multipe of x
	// ensure N <= timeForEachSlot
	leastFactor := multipleOf
	if leastFactor < time {
		time = leastFactor
	}
	return time
}

// Validate the algorithm computations
//  1. Verifies if 2D slice containing Min duration and Max duration values are non-zero
//  2. Idenfies the Ad Slots / Impressions with either Min Duration or Max Duration or both
//     having zero value and removes it from 2D slice
//  3. Ensures  Minimum Pod duration <= TotalSlotTime <= Maximum Pod Duration
// if  any validation fails it removes all the alloated slots and  makes is of size 0
// and sets the freeTime value as RequestedPodMaxDuration
func (config *adPodConfig) validateSlots() {

	// default return value if validation fails
	emptySlots := make([][2]int64, 0)
	if len(config.Slots) == 0 {
		return
	}

	returnEmptySlots := false

	// check slot with 0 values
	// remove them from config.Slots
	emptySlotCount := 0
	for index, slot := range config.Slots {
		if slot[0] == 0 || slot[1] == 0 {
			Logf("WARNING:Slot[%v][%v] is having 0 duration\n", index, slot)
			emptySlotCount++
			continue
		}

		// check slot boundaries
		if slot[1] < config.requestedSlotMinDuration || slot[1] > config.requestedSlotMaxDuration {
			Logf("ERROR: Slot%v Duration %v sec is out of either requestedSlotMinDuration (%v) or requestedSlotMaxDuration (%v)\n", index, slot[1], config.requestedSlotMinDuration, config.requestedSlotMaxDuration)
			returnEmptySlots = true
			break
		}
	}

	// remove empty slot
	if emptySlotCount > 0 {
		optimizedSlots := make([][2]int64, len(config.Slots)-emptySlotCount)
		for index, slot := range config.Slots {
			if slot[0] == 0 || slot[1] == 0 {
			} else {
				optimizedSlots[index][0] = slot[0]
				optimizedSlots[index][1] = slot[1]
			}
		}
		config.Slots = optimizedSlots
		Logf("Removed %v empty slots\n", emptySlotCount)
	}

	if int64(len(config.Slots)) < config.minAds || int64(len(config.Slots)) > config.maxAds {
		Logf("ERROR: slotSize %v is either less than Min Ads (%v) or greater than Max Ads (%v)\n", len(config.Slots), config.minAds, config.maxAds)
		returnEmptySlots = true
	}

	// ensure if min pod duration = max pod duration
	// config.TotalSlotTime = pod duration
	if config.requestedPodMinDuration == config.requestedPodMaxDuration && *config.totalSlotTime != config.requestedPodMaxDuration {
		Logf("ERROR: Total Slot Duration %v sec is not matching with Total Pod Duration %v sec\n", *config.totalSlotTime, config.requestedPodMaxDuration)
		returnEmptySlots = true
	}

	// ensure slot duration lies between requested min pod duration and  requested max pod duration
	// Testcase #15
	if *config.totalSlotTime < config.requestedPodMinDuration || *config.totalSlotTime > config.requestedPodMaxDuration {
		Logf("ERROR: Total Slot Duration %v sec is either less than Requested Pod Min Duration (%v sec) or greater than Requested  Pod Max Duration (%v sec)\n", *config.totalSlotTime, config.requestedPodMinDuration, config.requestedPodMaxDuration)
		returnEmptySlots = true
	}

	if returnEmptySlots {
		config.Slots = emptySlots
		config.freeTime = config.requestedPodMaxDuration
	}
}

// Adds time to possible slots and returns total added time
//
// Checks following for each Ad Slot
//  1. Can Ad Slot adjust the input time
//  2. If addition of new time to any slot not exeeding Total Pod Max Duration
// Performs the following operations
//  1. Populates Minimum duration slot[][0] - Either Slot Minimum Duration or Actual Slot Time computed
//  2. Populates Maximum duration slot[][1] - Always actual Slot Time computed
//  3. Counts the number of Ad Slots / Impressons full with  duration  capacity. If all Ad Slots / Impressions
//     are full of capacity it returns true as second return argument, indicating all slots are full with capacity
//  4. Keeps track of TotalSlotDuration when each new time is added to the Ad Slot
//  5. Keeps track of difference between computed PodMaxDuration and RequestedPodMaxDuration (TestCase #16) and used in step #2 above
// Returns argument 1 indicating total time adusted, argument 2 whether all slots are full of duration capacity
func (config adPodConfig) addTime(timeForEachSlot int64, fillZeroSlotsOnPriority bool) (int64, bool) {
	time := int64(0)

	// iterate over each ad
	slotCountFullWithCapacity := 0
	for ad := int64(0); ad < int64(len(config.Slots)); ad++ {

		slot := &config.Slots[ad]
		// check
		// 1. time(slot(0)) <= config.SlotMaxDuration
		// 2. if adding new time  to slot0 not exeeding config.SlotMaxDuration
		// 3. if sum(slot time) +  timeForEachSlot  <= config.RequestedPodMaxDuration
		canAdjustTime := (slot[0]+timeForEachSlot) <= config.requestedSlotMaxDuration && (slot[0]+timeForEachSlot) >= config.requestedSlotMinDuration
		totalSlotTimeWithNewTimeLessThanRequestedPodMaxDuration := *config.totalSlotTime+timeForEachSlot <= config.requestedPodMaxDuration

		// if fillZeroSlotsOnPriority= true ensure current slot value =  0
		allowCurrentSlot := !fillZeroSlotsOnPriority || (fillZeroSlotsOnPriority && slot[0] == 0)
		if slot[0] <= config.slotMaxDuration && canAdjustTime && totalSlotTimeWithNewTimeLessThanRequestedPodMaxDuration && allowCurrentSlot {
			slot[0] += timeForEachSlot

			// if we are adjusting the free time which will match up with config.RequestedPodMaxDuration
			// then set config.SlotMinDuration as min value for this slot
			// TestCase #16
			//if timeForEachSlot == maxPodDurationMatchUpTime {
			if timeForEachSlot < multipleOf {
				// override existing value of slot[0] here
				slot[0] = config.requestedSlotMinDuration
			}

			// check if this slot duration was zero
			if slot[1] == 0 {
				// decrememt config.slotsWithZeroTime as we added some time for this slot
				*config.slotsWithZeroTime--
			}

			slot[1] += timeForEachSlot
			*config.totalSlotTime += timeForEachSlot
			time += timeForEachSlot
			Logf("Slot %v = Added %v sec (New Time = %v)\n", ad, timeForEachSlot, slot[1])
		}
		// check slot capabity
		// !canAdjustTime - TestCase18
		if slot[1] == config.slotMaxDuration || !canAdjustTime {
			// slot is full
			slotCountFullWithCapacity++
		}
	}
	Logf("adjustedTime = %v\n ", time)
	return time, slotCountFullWithCapacity == len(config.Slots)
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

// Returns true if num is multipleof second argument. False otherwise
func isMultipleOf(num, multipleOf int64) bool {
	return math.Mod(float64(num), float64(multipleOf)) == 0
}

// Returns closet factor for num, with  respect  input multipleOf
//  Example: Closet Factor of 9, in multiples of 5 is '10'
func getClosetFactor(num, multipleOf int64) int64 {
	return int64(math.Round(float64(num)/float64(multipleOf)) * float64(multipleOf))
}

// Returns closetfactor of MinDuration, with  respect to multipleOf
// If computed factor < MinDuration then it will ensure and return
// close factor >=  MinDuration
func getClosetFactorForMinDuration(MinDuration int64, multipleOf int64) int64 {
	closedMinDuration := getClosetFactor(MinDuration, multipleOf)

	if closedMinDuration == 0 {
		return multipleOf
	}

	if closedMinDuration == MinDuration {
		return MinDuration
	}

	if closedMinDuration < MinDuration {
		return closedMinDuration + multipleOf
	}

	return closedMinDuration
}

// Returns closetfactor of maxduration, with  respect to multipleOf
// If computed factor > maxduration then it will ensure and return
// close factor <=  maxduration
func getClosetFactorForMaxDuration(maxduration, multipleOf int64) int64 {
	closedMaxDuration := getClosetFactor(maxduration, multipleOf)
	if closedMaxDuration == maxduration {
		return maxduration
	}

	// set closet maxduration closed to masduration
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

//shouldAdjustSlotWithZeroDuration - returns if slot with zero durations should be filled
// Currently it will return true in following condition
// cfg.minAds = cfg.maxads (i.e. Exact number of ads are required)
func (config adPodConfig) shouldAdjustSlotWithZeroDuration() bool {
	if config.minAds == config.maxAds {
		return true
	}
	return false
}
