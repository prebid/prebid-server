package impressions

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/util"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// keyDelim used as separator in forming key of maxExpectedDurationMap
var keyDelim = ","

type config struct {
	IImpressions
	generator []generator
	// maxExpectedDurationMap contains key = min , max duration, value = 0 -no of impressions, 1
	// this map avoids the unwanted repeatations of impressions generated
	//   Example,
	//   Step 1 : {{2, 17}, {15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}}
	//   Step 2 : {{2, 17}, {15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}}
	//   Step 3 : {{25, 25}, {25, 25}, {2, 22}, {5, 5}}
	//   Step 4 : {{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}}
	//   Step 5 : {{15, 15}, {15, 15}, {15, 15}, {15, 15}}
	//   Optimized Output : {{2, 17}, {15, 15},{15, 15},{15, 15},{15, 15},{10, 10},{10, 10},{10, 10},{10, 10},{10, 10},{10, 10},{25, 25}, {25, 25},{2, 22}, {5, 5}}
	//   This map will contains : {2, 17} = 1, {15, 15} = 4, {10, 10} = 6, {25, 25} = 2, {2, 22} = 1, {5, 5} =1
	maxExpectedDurationMap map[string][2]int
	requested              pod
}

// newMinMaxAlgorithm constructs instance of MinMaxAlgorithm
// It computes durations for Ad Slot and Ad Pod in multiple of X
// it also considers minimum configurations present in the request
func newMinMaxAlgorithm(podMinDuration, podMaxDuration int64, p openrtb_ext.VideoAdPod) config {
	generator := make([]generator, 0)
	// step 1 - same as Algorithm1
	generator = append(generator, initGenerator(podMinDuration, podMaxDuration, p, *p.MinAds, *p.MaxAds))
	// step 2 - pod duration = pod max, no of ads = max ads
	generator = append(generator, initGenerator(podMaxDuration, podMaxDuration, p, *p.MaxAds, *p.MaxAds))
	// step 3 - pod duration = pod max, no of ads = min ads
	generator = append(generator, initGenerator(podMaxDuration, podMaxDuration, p, *p.MinAds, *p.MinAds))
	// step 4 - pod duration = pod min, no of ads = max  ads
	generator = append(generator, initGenerator(podMinDuration, podMinDuration, p, *p.MaxAds, *p.MaxAds))
	// step 5 - pod duration = pod min, no of ads = min  ads
	generator = append(generator, initGenerator(podMinDuration, podMinDuration, p, *p.MinAds, *p.MinAds))

	return config{generator: generator, requested: generator[0].requested}
}

func initGenerator(podMinDuration, podMaxDuration int64, p openrtb_ext.VideoAdPod, minAds, maxAds int) generator {
	config := newConfigWithMultipleOf(podMinDuration, podMaxDuration, newVideoAdPod(p, minAds, maxAds), multipleOf)
	return config
}

func newVideoAdPod(p openrtb_ext.VideoAdPod, minAds, maxAds int) openrtb_ext.VideoAdPod {
	return openrtb_ext.VideoAdPod{MinDuration: p.MinDuration,
		MaxDuration: p.MaxDuration,
		MinAds:      &minAds,
		MaxAds:      &maxAds}
}

// Get ...
func (c *config) Get() [][2]int64 {
	imps := make([][2]int64, 0)
	wg := new(sync.WaitGroup) // ensures each step generating impressions is finished
	impsChan := make(chan [][2]int64, len(c.generator))
	for i := 0; i < len(c.generator); i++ {
		wg.Add(1)
		go get(c.generator[i], impsChan, wg)
	}

	// ensure impressions channel is closed
	// when all go routines are executed
	func() {
		defer close(impsChan)
		wg.Wait()
	}()

	c.maxExpectedDurationMap = make(map[string][2]int, 0)
	util.Logf("Step wise breakup ")
	for impressions := range impsChan {
		for index, impression := range impressions {
			impKey := getKey(impression)
			setMaximumRepeatations(c, impKey, index+1 == len(impressions))
		}
		util.Logf("%v", impressions)
	}

	// for impressions array
	indexOffset := 0
	for impKey := range c.maxExpectedDurationMap {
		totalRepeations := c.getRepeations(impKey)
		for repeation := 1; repeation <= totalRepeations; repeation++ {
			imps = append(imps, getImpression(impKey))
		}
		// if exact pod duration is provided then do not compute
		// min duration. Instead expect min duration same as max duration
		// It must be set by underneath algorithm
		if c.requested.podMinDuration != c.requested.podMaxDuration {
			computeMinDuration(*c, imps[:], indexOffset, indexOffset+totalRepeations)
		}
		indexOffset += totalRepeations
	}
	return imps
}

// getImpression constructs the impression object with min and max duration
// from input impression key
func getImpression(key string) [2]int64 {
	decodedKey := strings.Split(key, keyDelim)
	minDuration, _ := strconv.Atoi(decodedKey[MinDuration])
	maxDuration, _ := strconv.Atoi(decodedKey[MaxDuration])
	return [2]int64{int64(minDuration), int64(maxDuration)}
}

// setMaximumRepeatations avoids unwanted repeatations of impression object. Using following logic
// maxExpectedDurationMap value contains 2 types of storage
//  1. value[0] - represents current counter where final repeataions are stored
//  2. value[1] - local storage used by each impression object to add more repeatations if required
// impKey - key used to obtained already added repeatations for given impression
// updateCurrentCounter - if true and if current local storage value  > repeatations then repeations will be
// updated as current counter
func setMaximumRepeatations(c *config, impKey string, updateCurrentCounter bool) {
	// update maxCounter of each impression
	value := c.maxExpectedDurationMap[impKey]
	value[1]++ // increment max counter (contains no of repeatations for given iteration)
	c.maxExpectedDurationMap[impKey] = value
	// if val(maxCounter)  > actual store then consider temporary value as actual value
	if updateCurrentCounter {
		for k := range c.maxExpectedDurationMap {
			val := c.maxExpectedDurationMap[k]
			if val[1] > val[0] {
				val[0] = val[1]
			}
			// clear maxCounter
			val[1] = 0
			c.maxExpectedDurationMap[k] = val // reassign
		}
	}

}

// getKey returns the key used for refering values of maxExpectedDurationMap
// key is computed based on input impression object having min and max durations
func getKey(impression [2]int64) string {
	return fmt.Sprintf("%v%v%v", impression[MinDuration], keyDelim, impression[MaxDuration])
}

// getRepeations returns number of repeatations at that time that this algorithm will
// return w.r.t. input impressionKey
func (c config) getRepeations(impressionKey string) int {
	return c.maxExpectedDurationMap[impressionKey][0]
}

// get is internal function that actually computes the number of impressions
// based on configrations present in c
func get(c generator, ch chan [][2]int64, wg *sync.WaitGroup) {
	defer wg.Done()
	imps := c.Get()
	util.Logf("A2 Impressions = %v\n", imps)
	ch <- imps
}

// Algorithm returns MinMaxAlgorithm
func (c config) Algorithm() Algorithm {
	return MinMaxAlgorithm
}

func computeMinDuration(c config, impressions [][2]int64, start int, end int) {
	r := c.requested
	// 5/2 => q = 2 , r = 1 =>  2.5 => 3
	minDuration := int64(math.Round(float64(r.podMinDuration) / float64(r.minAds)))
	for i := start; i < end; i++ {
		impression := &impressions[i]
		// ensure imp duration boundaries
		// if boundaries are not honoured keep min duration which is computed as is
		if minDuration >= r.slotMinDuration && minDuration <= impression[MaxDuration] {
			// override previous value
			impression[MinDuration] = minDuration
		} else {
			// boundaries are not matching keep min value as is
			util.Logf("False : minDuration (%v) >= r.slotMinDuration (%v)  &&  minDuration (%v)  <= impression[MaxDuration] (%v)", minDuration, r.slotMinDuration, minDuration, impression[MaxDuration])
			util.Logf("Hence, setting request level slot minduration (%v) ", r.slotMinDuration)
			impression[MinDuration] = r.slotMinDuration
		}
	}
}
