package ctv

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var testBidResponseMaxDurations = []struct {
	scenario             string
	responseMaxDurations [][2]uint64
	podMinDuration       int // Pod Minimum duration value present in origin Video Ad Pod Request
	podMaxDuration       int // Pod Maximum duration value present in origin Video Ad Pod Request
	minAds               int // Minimum Ads value present in origin Video Ad Pod Request
	maxAds               int // Maximum Ads value present in origin Video Ad Pod Request
}{
	{
		scenario:             "TC1-Single_Value",
		responseMaxDurations: [][2]uint64{{14, 1}, {4, 3}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 1, maxAds: 2,
	}, {
		scenario:             "TC2-Multi_Value",
		responseMaxDurations: [][2]uint64{{1, 2}, {2, 2}, {3, 2}, {4, 2}, {5, 2}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 1, maxAds: 2,
	}, {
		scenario:             "TC3-max_ads = input_bid_durations",
		responseMaxDurations: [][2]uint64{{4, 2}, {5, 2}, {8, 2}, {7, 2}},
		podMinDuration:       10, podMaxDuration: 50, minAds: 2, maxAds: 5,
	}, {
		scenario:             "TC4-max_ads < input_bid_durations (test 1)",
		responseMaxDurations: [][2]uint64{{4, 2}, {5, 2}, {8, 2}, {7, 2}},
		podMinDuration:       10, podMaxDuration: 17, minAds: 3, maxAds: 3,
	}, {
		scenario:             "TC5-max_ads  (1) < input_bid_durations (test 1)",
		responseMaxDurations: [][2]uint64{{4, 2}, {5, 2}, {8, 2}, {7, 2}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 1,
	}, {
		scenario:             "TC6-max_ads < input_bid_durations (test 2)",
		responseMaxDurations: [][2]uint64{{4, 2}, {5, 2}, {8, 2}, {7, 2}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 2,
	}, {
		scenario:             "TC7-max_ads > input_bid_durations (test 1)",
		responseMaxDurations: [][2]uint64{{4, 2}, {5, 1}, {8, 2}, {7, 2}},
		podMinDuration:       10, podMaxDuration: 50, minAds: 4, maxAds: 4,
	},
	// {

	// 	// 4 - c1, c2,    :  5 - c3 : 6 - c4, c5,  8 : c7
	// 	scenario:             "TC8-max_ads (20 ads) > input_bid_durations (test 2)",
	// 	responseMaxDurations: []uint64{4, 5, 8, 7},
	// 	podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 20,
	// 	combinations: [][]int64{{14}}},
	{

		// 4 - c1, c2,    :  5 - c3 : 6 - c4, c5,  8 : c7
		scenario:             "TC6-max_ads (20 ads) > input_bid_durations-repeatation_not_allowed",
		responseMaxDurations: [][2]uint64{{4, 2}, {5, 2}, {8, 2}, {7, 2}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 2,
	},
	// {

	// 	// 4 - c1, c2,    :  5 - c3 : 6 - c4, c5,  8 : c7
	// 	scenario:             "TC8-max_ads (20 ads) > input_bid_durations (no repitations)",
	// 	responseMaxDurations: []uint64{4, 5, 8, 7},
	// 	podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 20,
	// 	combinations:                           [][]int64{{14}},
	// 	allowRepetitationsForEligibleDurations: "true", // no repeitations
	// },

	// {

	// 	// 4 - c1, c2,    :  5 - c3 : 6 - c4, c5,  8 : c7
	// 	scenario:             "TC9-max_ads = input_bid_durations = 4",
	// 	responseMaxDurations: []uint64{4, 4, 4, 4},
	// 	podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 4,
	// 	combinations: [][]int64{{14}}, allowRepetitationsForEligibleDurations: "true"},
	{
		scenario:             "TC10-max_ads 0",
		responseMaxDurations: [][2]uint64{{4, 2}, {4, 2}, {4, 2}, {4, 2}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 0,
	}, {
		scenario:             "TC11-max_ads =5-input-empty",
		responseMaxDurations: [][2]uint64{},
		podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 0,
	}, {
		scenario:             "TC12-max_ads =5-input-empty-no-repeatation",
		responseMaxDurations: [][2]uint64{{25, 2}, {30, 2}, {76, 2}, {10, 2}, {88, 2}},
		podMinDuration:       10, podMaxDuration: 229, minAds: 1, maxAds: 4,
	}, {
		scenario:             "TC13-max_ads = input = 10-without-repeatation",
		responseMaxDurations: [][2]uint64{{25, 2}, {30, 2}, {76, 2}, {10, 2}, {88, 2}, {34, 2}, {37, 2}, {67, 2}, {89, 2}, {45, 2}},
		podMinDuration:       10, podMaxDuration: 14, minAds: 3, maxAds: 10,
	}, {
		scenario:             "TC14-single duration: single ad",
		responseMaxDurations: [][2]uint64{{15, 1}},
		podMinDuration:       10, podMaxDuration: 15, minAds: 1, maxAds: 5,
	}, {
		scenario:             "TC15-exact-pod-duration",
		responseMaxDurations: [][2]uint64{{25, 2}, {30, 2}, {76, 2}, {10, 2}, {88, 2}},
		podMinDuration:       200, podMaxDuration: 200, minAds: 8, maxAds: 10,
	}, {
		scenario:             "TC16-50ads",
		responseMaxDurations: [][2]uint64{{25, 2}, {30, 2}, {76, 2}, {10, 2}, {88, 2}},
		podMinDuration:       200, podMaxDuration: 200, minAds: 10, maxAds: 10, /*50*/
	},
}

func BenchmarkPodDurationCombinationGenerator(b *testing.B) {
	for _, test := range testBidResponseMaxDurations {
		b.Run(test.scenario, func(b *testing.B) {
			c := new(PodDurationCombination)
			config := new(openrtb_ext.VideoAdPod)
			config.MinAds = &test.minAds
			config.MaxAds = &test.maxAds
			config.MinDuration = &test.podMinDuration
			config.MaxDuration = &test.podMaxDuration

			for n := 0; n < b.N; n++ {
				for true {
					comb := c.Next()
					if nil == comb || len(comb) == 0 {
						break
					}
				}
			}
		})
	}
}

func TestPodDurationCombinationGenerator(t *testing.T) {
	for _, test := range testBidResponseMaxDurations {

		t.Run(test.scenario, func(t *testing.T) {
			c := new(PodDurationCombination)
			//log.Printf("Input = %v", test.responseMaxDurations)

			config := new(openrtb_ext.VideoAdPod)
			config.MinAds = &test.minAds
			config.MaxAds = &test.maxAds

			c.Init(uint64(test.podMinDuration), uint64(test.podMaxDuration), config, test.responseMaxDurations)
			expectedOutput := c.searchAll()
			// determine expected size of expected output
			// subtract invalid combinations size

			actualOutput := make([][]uint64, len(expectedOutput))

			cnt := 0
			for true {
				comb := c.Next()
				if comb == nil || len(comb) == 0 {
					break
				}
				print("%v", comb)
				//fmt.Print("count = ", c.currentCombinationCount, " :: ", comb, "\n")
				//fmt.Println("e = ", (expectedOutput)[cnt], "\t : a = ", comb)
				val := make([]uint64, len(comb))
				copy(val, comb)
				actualOutput[cnt] = val
				cnt++
			}

			if expectedOutput != nil {
				// compare results
				for i := uint64(0); i < uint64(len(expectedOutput)); i++ {
					if expectedOutput[i] == nil {
						continue
					}
					for j := uint64(0); j < uint64(len(expectedOutput[i])); j++ {
						if expectedOutput[i][j] == actualOutput[i][j] {
						} else {

							assert.Fail(t, "expectedOutput[", i, "][", j, "] != actualOutput[", i, "][", j, "] ", expectedOutput[i][j], " !=", actualOutput[i][j])

						}
					}

				}
			}

			assert.Equal(t, expectedOutput, actualOutput)
			assert.ElementsMatch(t, expectedOutput, actualOutput)

			print("config = %v", test)
			print("Total combinations generated = %v", c.stats.currentCombinationCount)
			print("Total valid combinations  = %v", c.stats.validCombinationCount)
			print("Total repeated combinations  = %v", c.stats.repeatationsCount)
			print("Total outofrange combinations  = %v", c.stats.outOfRangeCount)
			print("Total combinations expected = %v", c.stats.totalExpectedCombinations)
		})
	}
}
