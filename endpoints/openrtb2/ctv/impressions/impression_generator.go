package impressions

import (
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/util"
)

// generator contains Pod Minimum Duration, Pod Maximum Duration, Slot Minimum Duration and Slot Maximum Duration
// It holds additional attributes required by this algorithm for  internal computation.
// 	It contains Slots attribute. This  attribute holds the output of this algorithm
type generator struct {
	IImpressions
	Slots             [][2]int64 // Holds Minimum and Maximum duration (in seconds) for each Ad Slot. Length indicates total number of Ad Slots/ Impressions for given Ad Pod
	totalSlotTime     *int64     // Total Sum of all Ad Slot durations (in seconds)
	freeTime          int64      // Remaining Time (in seconds) not allocated. It is compared with RequestedPodMaxDuration
	slotsWithZeroTime *int64     // Indicates number of slots with zero time (starting from 1).
	// requested holds all the requested information received
	requested pod
	// internal holds the slot duration values closed to original value and multiples of X.
	// It helps in plotting impressions with duration values in multiples of given number
	internal internal
}

// pod for internal computation
// should not be used outside
type pod struct {
	minAds          int64
	maxAds          int64
	slotMinDuration int64
	slotMaxDuration int64
	podMinDuration  int64
	podMaxDuration  int64
}

// internal (FOR INTERNAL USE ONLY) holds the computed values slot min and max duration
// in multiples of given number. It also holds slotDurationComputed flag
// if slotDurationComputed = false, it means values computed were overlapping
type internal struct {
	slotMinDuration      int64
	slotMaxDuration      int64
	slotDurationComputed bool
}

// Get returns the number of Ad Slots/Impression  that input Ad Pod can have.
// It returns List 2D array containing following
//  1. Dimension 1 - Represents the minimum duration of an impression
//  2. Dimension 2 - Represents the maximum duration of an impression
func (config *generator) Get() [][2]int64 {
	util.Logf("Pod Config with Internal Computation (using multiples of %v) = %+v\n", multipleOf, config)
	totalAds := computeTotalAds(*config)
	timeForEachSlot := computeTimeForEachAdSlot(*config, totalAds)

	config.Slots = make([][2]int64, totalAds)
	config.slotsWithZeroTime = new(int64)
	*config.slotsWithZeroTime = totalAds
	util.Logf("Plotted Ad Slots / Impressions of size = %v\n", len(config.Slots))
	// iterate over total time till it is < cfg.RequestedPodMaxDuration
	time := int64(0)
	util.Logf("Started allocating durations to each Ad Slot / Impression\n")
	fillZeroSlotsOnPriority := true
	noOfZeroSlotsFilledByLastRun := int64(0)
	*config.totalSlotTime = 0
	for time < config.requested.podMaxDuration {
		adjustedTime, slotsFull := config.addTime(timeForEachSlot, fillZeroSlotsOnPriority)
		time += adjustedTime
		timeForEachSlot = computeTimeLeastValue(config.requested.podMaxDuration-time, config.requested.slotMaxDuration-timeForEachSlot)
		if slotsFull {
			util.Logf("All slots are full of their capacity. validating slots\n")
			break
		}

		// instruct for filling zero capacity slots on priority if
		// 1. shouldAdjustSlotWithZeroDuration returns true
		// 2. there are slots with 0 duration
		// 3. there is at least ont slot with zero duration filled by last iteration
		fillZeroSlotsOnPriority = false
		noOfZeroSlotsFilledByLastRun = *config.slotsWithZeroTime - noOfZeroSlotsFilledByLastRun
		if config.shouldAdjustSlotWithZeroDuration() && *config.slotsWithZeroTime > 0 && noOfZeroSlotsFilledByLastRun > 0 {
			fillZeroSlotsOnPriority = true
		}
	}
	util.Logf("Completed allocating durations to each Ad Slot / Impression\n")

	// validate slots
	config.validateSlots()

	// log free time if present to stats server
	// also check algoritm computed the no. of ads
	if config.requested.podMaxDuration-time > 0 && len(config.Slots) > 0 {
		config.freeTime = config.requested.podMaxDuration - time
		util.Logf("TO STATS SERVER : Free Time not allocated %v sec", config.freeTime)
	}

	util.Logf("\nTotal Impressions = %v, Total Allocated Time = %v sec (out of %v sec, Max Pod Duration)\n%v", len(config.Slots), *config.totalSlotTime, config.requested.podMaxDuration, config.Slots)
	return config.Slots
}

// Returns total number of Ad Slots/ impressions that the Ad Pod can have
func computeTotalAds(cfg generator) int64 {
	if cfg.internal.slotMaxDuration <= 0 || cfg.internal.slotMinDuration <= 0 {
		util.Logf("Either cfg.slotMaxDuration or cfg.slotMinDuration or both are <= 0. Hence, totalAds = 0")
		return 0
	}
	minAds := cfg.requested.podMaxDuration / cfg.internal.slotMaxDuration
	maxAds := cfg.requested.podMaxDuration / cfg.internal.slotMinDuration

	util.Logf("Computed minAds = %v , maxAds = %v\n", minAds, maxAds)

	totalAds := max(minAds, maxAds)
	util.Logf("Computed max(minAds, maxAds) = totalAds = %v\n", totalAds)

	if totalAds < cfg.requested.minAds {
		totalAds = cfg.requested.minAds
		util.Logf("Computed totalAds < requested  minAds (%v). Hence, setting totalAds =  minAds = %v\n", cfg.requested.minAds, totalAds)
	}
	if totalAds > cfg.requested.maxAds {
		totalAds = cfg.requested.maxAds
		util.Logf("Computed totalAds > requested  maxAds (%v). Hence, setting totalAds =  maxAds = %v\n", cfg.requested.maxAds, totalAds)
	}
	util.Logf("Computed Final totalAds = %v  [%v <= %v <= %v]\n", totalAds, cfg.requested.minAds, totalAds, cfg.requested.maxAds)
	return totalAds
}

// Returns duration in seconds that can be allocated to each Ad Slot
// Accepts cfg containing algorithm configurations and totalAds containing Total number of
// Ad Slots / Impressions that the Ad Pod can have.
func computeTimeForEachAdSlot(cfg generator, totalAds int64) int64 {
	// Compute time for each ad
	if totalAds <= 0 {
		util.Logf("totalAds = 0, Hence timeForEachSlot = 0")
		return 0
	}
	timeForEachSlot := cfg.requested.podMaxDuration / totalAds

	util.Logf("Computed timeForEachSlot = %v (podMaxDuration/totalAds) (%v/%v)\n", timeForEachSlot, cfg.requested.podMaxDuration, totalAds)

	if timeForEachSlot < cfg.internal.slotMinDuration {
		timeForEachSlot = cfg.internal.slotMinDuration
		util.Logf("Computed timeForEachSlot < requested  slotMinDuration (%v). Hence, setting timeForEachSlot =  slotMinDuration = %v\n", cfg.internal.slotMinDuration, timeForEachSlot)
	}

	if timeForEachSlot > cfg.internal.slotMaxDuration {
		timeForEachSlot = cfg.internal.slotMaxDuration
		util.Logf("Computed timeForEachSlot > requested  slotMaxDuration (%v). Hence, setting timeForEachSlot =  slotMaxDuration = %v\n", cfg.internal.slotMaxDuration, timeForEachSlot)
	}

	// Case - Exact slot duration is given. No scope for finding multiples
	// of given number. Prefer to return computed timeForEachSlot
	// In such case timeForEachSlot no necessarily to be multiples of given number
	if cfg.requested.slotMinDuration == cfg.requested.slotMaxDuration {
		util.Logf("requested.slotMinDuration = requested.slotMaxDuration = %v. Hence, not computing multiples of %v value.", cfg.requested.slotMaxDuration, multipleOf)
		return timeForEachSlot
	}

	// Case II - timeForEachSlot*totalAds > podmaxduration
	// In such case prefer to return cfg.podMaxDuration / totalAds
	// In such case timeForEachSlot no necessarily to be multiples of given number
	if (timeForEachSlot * totalAds) > cfg.requested.podMaxDuration {
		util.Logf("timeForEachSlot*totalAds (%v) > cfg.requested.podMaxDuration (%v) ", timeForEachSlot*totalAds, cfg.requested.podMaxDuration)
		util.Logf("Hence, not computing multiples of %v value.", multipleOf)
		// need that division again
		return cfg.requested.podMaxDuration / totalAds
	}

	// ensure timeForEachSlot is multipleof given number
	if cfg.internal.slotDurationComputed && !isMultipleOf(timeForEachSlot, multipleOf) {
		// get close to value of multiple
		// here we muse get either cfg.SlotMinDuration or cfg.SlotMaxDuration
		// these values are already pre-computed in multiples of given number
		timeForEachSlot = getClosestFactor(timeForEachSlot, multipleOf)
		util.Logf("Computed closet factor %v, in multiples of %v for timeForEachSlot\n", timeForEachSlot, multipleOf)
	}
	util.Logf("Computed Final timeForEachSlot = %v  [%v <= %v <= %v]\n", timeForEachSlot, cfg.requested.slotMinDuration, timeForEachSlot, cfg.requested.slotMaxDuration)
	return timeForEachSlot
}

// Checks if multipleOf can be used as least time value
// this will ensure eack slot to maximize its time if possible
// if multipleOf can not be used as least value then default input value is returned as is
// accepts time containing, which least value to be computed.
// leastTimeRequiredByEachSlot - indicates the mimimum time that any slot can accept (UOE-5268)
// Returns the least value based on multiple of X
func computeTimeLeastValue(time int64, leastTimeRequiredByEachSlot int64) int64 {
	// time if Testcase#6
	// 1. multiple of x - get smallest factor N of multiple of x for time
	// 2. not multiple of x - try to obtain smallet no N multipe of x
	// ensure N <= timeForEachSlot
	leastFactor := multipleOf
	if leastFactor < time {
		time = leastFactor
	}

	// case:  check if slots are looking for time < leastFactor
	// UOE-5268
	if leastTimeRequiredByEachSlot > 0 && leastTimeRequiredByEachSlot < time {
		time = leastTimeRequiredByEachSlot
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
func (config *generator) validateSlots() {

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
			util.Logf("WARNING:Slot[%v][%v] is having 0 duration\n", index, slot)
			emptySlotCount++
			continue
		}

		// check slot boundaries
		if slot[1] < config.requested.slotMinDuration || slot[1] > config.requested.slotMaxDuration {
			util.Logf("ERROR: Slot%v Duration %v sec is out of either requested.slotMinDuration (%v) or requested.slotMaxDuration (%v)\n", index, slot[1], config.requested.slotMinDuration, config.requested.slotMaxDuration)
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
		util.Logf("Removed %v empty slots\n", emptySlotCount)
	}

	if int64(len(config.Slots)) < config.requested.minAds || int64(len(config.Slots)) > config.requested.maxAds {
		util.Logf("ERROR: slotSize %v is either less than Min Ads (%v) or greater than Max Ads (%v)\n", len(config.Slots), config.requested.minAds, config.requested.maxAds)
		returnEmptySlots = true
	}

	// ensure if min pod duration = max pod duration
	// config.TotalSlotTime = pod duration
	if config.requested.podMinDuration == config.requested.podMaxDuration && *config.totalSlotTime != config.requested.podMaxDuration {
		util.Logf("ERROR: Total Slot Duration %v sec is not matching with Total Pod Duration %v sec\n", *config.totalSlotTime, config.requested.podMaxDuration)
		returnEmptySlots = true
	}

	// ensure slot duration lies between requested min pod duration and  requested max pod duration
	// Testcase #15
	if *config.totalSlotTime < config.requested.podMinDuration || *config.totalSlotTime > config.requested.podMaxDuration {
		util.Logf("ERROR: Total Slot Duration %v sec is either less than Requested Pod Min Duration (%v sec) or greater than Requested  Pod Max Duration (%v sec)\n", *config.totalSlotTime, config.requested.podMinDuration, config.requested.podMaxDuration)
		returnEmptySlots = true
	}

	if returnEmptySlots {
		config.Slots = emptySlots
		config.freeTime = config.requested.podMaxDuration
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
func (config generator) addTime(timeForEachSlot int64, fillZeroSlotsOnPriority bool) (int64, bool) {
	time := int64(0)

	// iterate over each ad
	slotCountFullWithCapacity := 0
	for ad := int64(0); ad < int64(len(config.Slots)); ad++ {

		slot := &config.Slots[ad]
		// check
		// 1. time(slot(0)) <= config.SlotMaxDuration
		// 2. if adding new time  to slot0 not exeeding config.SlotMaxDuration
		// 3. if sum(slot time) +  timeForEachSlot  <= config.RequestedPodMaxDuration
		canAdjustTime := (slot[1]+timeForEachSlot) <= config.requested.slotMaxDuration && (slot[1]+timeForEachSlot) >= config.requested.slotMinDuration
		totalSlotTimeWithNewTimeLessThanRequestedPodMaxDuration := *config.totalSlotTime+timeForEachSlot <= config.requested.podMaxDuration

		// if fillZeroSlotsOnPriority= true ensure current slot value =  0
		allowCurrentSlot := !fillZeroSlotsOnPriority || (fillZeroSlotsOnPriority && slot[1] == 0)
		if slot[1] <= config.internal.slotMaxDuration && canAdjustTime && totalSlotTimeWithNewTimeLessThanRequestedPodMaxDuration && allowCurrentSlot {
			slot[0] += timeForEachSlot

			// if we are adjusting the free time which will match up with config.RequestedPodMaxDuration
			// then set config.SlotMinDuration as min value for this slot
			// TestCase #16
			//if timeForEachSlot == maxPodDurationMatchUpTime {
			if timeForEachSlot < multipleOf {
				// override existing value of slot[0] here
				slot[0] = config.requested.slotMinDuration
			}

			// check if this slot duration was zero
			if slot[1] == 0 {
				// decrememt config.slotsWithZeroTime as we added some time for this slot
				*config.slotsWithZeroTime--
			}

			slot[1] += timeForEachSlot
			*config.totalSlotTime += timeForEachSlot
			time += timeForEachSlot
			util.Logf("Slot %v = Added %v sec (New Time = %v)\n", ad, timeForEachSlot, slot[1])
		}
		// check slot capabity
		// !canAdjustTime - TestCase18
		// UOE-5268 - Check with Requested Slot Max Duration
		if slot[1] == config.requested.slotMaxDuration || !canAdjustTime {
			// slot is full
			slotCountFullWithCapacity++
		}
	}
	util.Logf("adjustedTime = %v\n ", time)
	return time, slotCountFullWithCapacity == len(config.Slots)
}

//shouldAdjustSlotWithZeroDuration - returns if slot with zero durations should be filled
// Currently it will return true in following condition
// cfg.minAds = cfg.maxads (i.e. Exact number of ads are required)
func (config generator) shouldAdjustSlotWithZeroDuration() bool {
	if config.requested.minAds == config.requested.maxAds {
		return true
	}
	return false
}
