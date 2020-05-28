package ctv

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

type TestAdPod struct {
	vPod           openrtb_ext.VideoAdPod
	podMinDuration int64
	podMaxDuration int64
}

type Expected struct {
	impressionCount int
	// Time remaining after ad breaking is done
	// if no ad breaking i.e. 0 then freeTime = pod.maxduration
	freeTime        int64
	adSlotTimeInSec []int64

	// close bounds
	closedMinDuration     int64 // pod
	closedMaxDuration     int64 // pod
	closedSlotMinDuration int64 // ad slot
	closedSlotMaxDuration int64 // ad slot

	output [][2]int64
}

var impressionsTests = []struct {
	scenario string   // Testcase scenario
	in       []int    // Testcase input
	out      Expected // Testcase execpted output
}{
	{scenario: "TC2", in: []int{1, 90, 11, 15, 2, 8}, out: Expected{
		impressionCount:       6,
		freeTime:              0.0,
		output:                [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}},
		closedMinDuration:     5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC3", in: []int{1, 90, 11, 15, 2, 4}, out: Expected{
		impressionCount: 4,
		freeTime:        30.0,
		output:          [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}},

		closedMinDuration:     5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC4", in: []int{1, 15, 1, 15, 1, 1}, out: Expected{
		impressionCount: 1,
		freeTime:        0.0,
		output:          [][2]int64{{15, 15}},

		closedMinDuration:     5,
		closedMaxDuration:     15,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC5", in: []int{1, 15, 1, 15, 1, 2}, out: Expected{
		impressionCount: 2,
		freeTime:        0.0,
		output:          [][2]int64{{10, 10}, {5, 5}},

		closedMinDuration:     5,
		closedMaxDuration:     15,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC6", in: []int{1, 90, 1, 15, 1, 8}, out: Expected{
		impressionCount: 8,
		freeTime:        0.0,
		output:          [][2]int64{{15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC7", in: []int{15, 30, 8, 15, 1, 1}, out: Expected{
		impressionCount: 1,
		freeTime:        15.0,
		output:          [][2]int64{{15, 15}},

		closedMinDuration:     15,
		closedMaxDuration:     30,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC8", in: []int{35, 35, 10, 35, 3, 40}, out: Expected{
		impressionCount: 3,
		freeTime:        0.0,
		output:          [][2]int64{{15, 15}, {10, 10}, {10, 10}},

		closedMinDuration:     35,
		closedMaxDuration:     35,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC9", in: []int{35, 35, 10, 35, 6, 40}, out: Expected{
		impressionCount: 0,
		freeTime:        35,
		output:          [][2]int64{},

		closedMinDuration:     35,
		closedMaxDuration:     35,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC10", in: []int{35, 65, 10, 35, 6, 40}, out: Expected{
		impressionCount: 6,
		freeTime:        0.0,
		output:          [][2]int64{{15, 15}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     35,
		closedMaxDuration:     65,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC11", in: []int{35, 65, 9, 35, 7, 40}, out: Expected{
		impressionCount: 0, //7,
		freeTime:        65,
		output:          [][2]int64{},

		closedMinDuration:     35,
		closedMaxDuration:     65,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC12", in: []int{100, 100, 10, 35, 6, 40}, out: Expected{
		impressionCount: 10,
		freeTime:        0.0,
		output:          [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     100,
		closedMaxDuration:     100,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC13", in: []int{60, 60, 5, 9, 1, 6}, out: Expected{
		impressionCount: 0,
		freeTime:        60,
		output:          [][2]int64{},

		closedMinDuration:     60,
		closedMaxDuration:     60,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC14", in: []int{30, 60, 5, 9, 1, 6}, out: Expected{
		impressionCount: 6,
		freeTime:        6,
		output:          [][2]int64{{5, 9}, {5, 9}, {5, 9}, {5, 9}, {5, 9}, {5, 9}},

		closedMinDuration:     30,
		closedMaxDuration:     60,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC15", in: []int{30, 60, 5, 9, 1, 5}, out: Expected{
		impressionCount: 5,
		freeTime:        15,
		output:          [][2]int64{{5, 9}, {5, 9}, {5, 9}, {5, 9}, {5, 9}},

		closedMinDuration:     30,
		closedMaxDuration:     60,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC16", in: []int{126, 126, 1, 12, 7, 13}, out: Expected{
		impressionCount: 13,
		freeTime:        0,
		output:          [][2]int64{{1, 12}, {1, 12}, {1, 12}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     126,
		closedMaxDuration:     126,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC17", in: []int{127, 128, 1, 12, 7, 13}, out: Expected{
		impressionCount: 13,
		freeTime:        0,
		output:          [][2]int64{{1, 12}, {1, 12}, {1, 12}, {1, 12}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     130,
		closedMaxDuration:     125,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC18", in: []int{125, 125, 4, 4, 1, 1}, out: Expected{
		impressionCount: 0,
		freeTime:        125,
		output:          [][2]int64{},

		closedMinDuration:     125,
		closedMaxDuration:     125,
		closedSlotMinDuration: 4,
		closedSlotMaxDuration: 4,
	}},
	{scenario: "TC19", in: []int{90, 90, 7, 9, 3, 5}, out: Expected{
		impressionCount: 0,
		freeTime:        90,
		output:          [][2]int64{},

		closedMinDuration:     90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC20", in: []int{90, 90, 5, 10, 1, 11}, out: Expected{
		impressionCount: 9,
		freeTime:        0,
		output:          [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC21", in: []int{2, 170, 3, 9, 4, 9}, out: Expected{
		impressionCount: 9,
		freeTime:        89,
		output:          [][2]int64{{3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}},

		closedMinDuration:     5,
		closedMaxDuration:     170,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC23", in: []int{118, 124, 4, 17, 6, 15}, out: Expected{
		impressionCount: 12,
		freeTime:        0,
		output:          [][2]int64{{4, 14}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},

		closedMinDuration:     120,
		closedMaxDuration:     120,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC24", in: []int{134, 134, 60, 90, 2, 3}, out: Expected{
		impressionCount: 2,
		freeTime:        0,
		output:          [][2]int64{{60, 69}, {65, 65}},

		closedMinDuration:     134,
		closedMaxDuration:     134,
		closedSlotMinDuration: 60,
		closedSlotMaxDuration: 90,
	}},
	{scenario: "TC25", in: []int{88, 88, 1, 80, 2, 2}, out: Expected{
		impressionCount:       2,
		freeTime:              0,
		output:                [][2]int64{{1, 68}, {20, 20}},
		closedMinDuration:     88,
		closedMaxDuration:     88,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 80,
	}},
	{scenario: "TC26", in: []int{90, 90, 45, 45, 2, 3}, out: Expected{
		impressionCount:       2,
		freeTime:              0,
		output:                [][2]int64{{45, 45}, {45, 45}},
		closedMinDuration:     90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 45,
		closedSlotMaxDuration: 45,
	}},
	{scenario: "TC27", in: []int{5, 90, 2, 45, 2, 3}, out: Expected{
		impressionCount:       3,
		freeTime:              0,
		output:                [][2]int64{{30, 30}, {30, 30}, {30, 30}},
		closedMinDuration:     5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 45,
	}},
	{scenario: "TC28", in: []int{5, 180, 2, 90, 2, 6}, out: Expected{
		impressionCount:       6,
		freeTime:              0,
		output:                [][2]int64{{30, 30}, {30, 30}, {30, 30}, {30, 30}, {30, 30}, {30, 30}},
		closedMinDuration:     5,
		closedMaxDuration:     180,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 90,
	}},
	{scenario: "TC29", in: []int{5, 65, 2, 35, 2, 3}, out: Expected{
		impressionCount: 3,
		freeTime:        0,
		output:          [][2]int64{{25, 25}, {20, 20}, {20, 20}},

		closedMinDuration:     5,
		closedMaxDuration:     65,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC30", in: []int{123, 123, 34, 34, 3, 3}, out: Expected{
		impressionCount: 3,
		freeTime:        123,
		output:          [][2]int64{},

		closedMinDuration:     123,
		closedMaxDuration:     123,
		closedSlotMinDuration: 34,
		closedSlotMaxDuration: 34,
	}},
	{scenario: "TC31", in: []int{123, 123, 31, 31, 3, 3}, out: Expected{
		impressionCount: 3,
		freeTime:        123,
		output:          [][2]int64{},

		closedMinDuration:     123,
		closedMaxDuration:     123,
		closedSlotMinDuration: 31,
		closedSlotMaxDuration: 31,
	}}, {scenario: "TC32", in: []int{134, 134, 63, 63, 2, 3}, out: Expected{
		impressionCount: 0,
		freeTime:        134,
		output:          [][2]int64{},

		closedMinDuration:     134,
		closedMaxDuration:     134,
		closedSlotMinDuration: 63,
		closedSlotMaxDuration: 63,
	}},
	{scenario: "TC33", in: []int{147, 147, 30, 60, 4, 6}, out: Expected{
		impressionCount: 4,
		freeTime:        0,
		output:          [][2]int64{{30, 42}, {35, 35}, {35, 35}, {35, 35}},

		closedMinDuration:     147,
		closedMaxDuration:     147,
		closedSlotMinDuration: 30,
		closedSlotMaxDuration: 60,
	}},
	{scenario: "TC34", in: []int{88, 102, 30, 30, 3, 3}, out: Expected{
		impressionCount: 3,
		freeTime:        12,
		output:          [][2]int64{{30, 30}, {30, 30}, {30, 30}},

		closedMinDuration:     90,
		closedMaxDuration:     100,
		closedSlotMinDuration: 30,
		closedSlotMaxDuration: 30,
	}}, {scenario: "TC35", in: []int{88, 102, 30, 42, 3, 3}, out: Expected{
		impressionCount: 0,
		freeTime:        102,
		output:          [][2]int64{},

		closedMinDuration:     90,
		closedMaxDuration:     100,
		closedSlotMinDuration: 30,
		closedSlotMaxDuration: 40,
	}}, {scenario: "TC36", in: []int{90, 90, 45, 45, 2, 5}, out: Expected{
		impressionCount: 2,
		freeTime:        0,
		output:          [][2]int64{{45, 45}, {45, 45}},

		closedMinDuration:     90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 45,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC37", in: []int{10, 45, 20, 45, 2, 5}, out: Expected{
		impressionCount: 2,
		freeTime:        0,
		output:          [][2]int64{{25, 25}, {20, 20}},

		closedMinDuration:     10,
		closedMaxDuration:     45,
		closedSlotMinDuration: 20,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC38", in: []int{90, 90, 20, 45, 2, 5}, out: Expected{
		impressionCount: 0,
		freeTime:        0,
		output:          [][2]int64{{25, 25}, {25, 25}, {20, 20}, {20, 20}},

		closedMinDuration:     90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 20,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC39", in: []int{60, 90, 20, 45, 2, 5}, out: Expected{
		impressionCount: 4,
		freeTime:        0,
		output:          [][2]int64{{25, 25}, {25, 25}, {20, 20}, {20, 20}},

		closedMinDuration:     60,
		closedMaxDuration:     90,
		closedSlotMinDuration: 20,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC40", in: []int{95, 95, 5, 45, 10, 10}, out: Expected{
		impressionCount: 10,
		freeTime:        0,
		output:          [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}},

		closedMinDuration:     95,
		closedMaxDuration:     95,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC41", in: []int{95, 123, 5, 45, 13, 13}, out: Expected{
		impressionCount: 0,
		freeTime:        123,
		output:          [][2]int64{},

		closedMinDuration:     95,
		closedMaxDuration:     120,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC42", in: []int{1, 1, 1, 1, 1, 1}, out: Expected{
		impressionCount: 1,
		freeTime:        0,
		output:          [][2]int64{{1, 1}},

		closedMinDuration:     1,
		closedMaxDuration:     1,
		closedSlotMinDuration: 1,
		closedSlotMaxDuration: 1,
	}}, {scenario: "TC43", in: []int{2, 2, 2, 2, 2, 2}, out: Expected{
		impressionCount: 0,
		freeTime:        2,
		output:          [][2]int64{},

		closedMinDuration:     2,
		closedMaxDuration:     2,
		closedSlotMinDuration: 2,
		closedSlotMaxDuration: 2,
	}}, {scenario: "TC44", in: []int{0, 0, 0, 0, 0, 0}, out: Expected{
		impressionCount: 0,
		freeTime:        0,
		output:          [][2]int64{},

		closedMinDuration:     0,
		closedMaxDuration:     0,
		closedSlotMinDuration: 0,
		closedSlotMaxDuration: 0,
	}}, {scenario: "TC45", in: []int{-1, -2, -3, -4, -5, -6}, out: Expected{
		impressionCount: 0,
		freeTime:        0,
		output:          [][2]int64{},

		closedMinDuration:     5,
		closedMaxDuration:     -5,
		closedSlotMinDuration: 0,
		closedSlotMaxDuration: -5,
	}}, {scenario: "TC46", in: []int{-1, -1, -1, -1, -1, -1}, out: Expected{
		impressionCount: 0,
		freeTime:        0,
		output:          [][2]int64{},

		closedMinDuration:     -1,
		closedMaxDuration:     -1,
		closedSlotMinDuration: -1,
		closedSlotMaxDuration: -1,
	}}, {scenario: "TC47", in: []int{6, 6, 6, 6, 1, 1}, out: Expected{
		impressionCount: 1,
		freeTime:        0,
		output:          [][2]int64{{6, 6}},

		closedMinDuration:     6,
		closedMaxDuration:     6,
		closedSlotMinDuration: 6,
		closedSlotMaxDuration: 6,
	}}, {scenario: "TC48", in: []int{12, 12, 6, 6, 1, 2}, out: Expected{
		impressionCount: 2,
		freeTime:        0,
		output:          [][2]int64{{6, 6}, {6, 6}},

		closedMinDuration:     12,
		closedMaxDuration:     12,
		closedSlotMinDuration: 6,
		closedSlotMaxDuration: 6,
	}}, {scenario: "TC49", in: []int{12, 12, 7, 7, 1, 2}, out: Expected{
		impressionCount: 0,
		freeTime:        12,
		output:          [][2]int64{},

		closedMinDuration:     12,
		closedMaxDuration:     12,
		closedSlotMinDuration: 7,
		closedSlotMaxDuration: 7,
	}}, {scenario: "TC50", in: []int{1, 1, 1, 1, 1, 1}, out: Expected{
		impressionCount: 0,
		freeTime:        0,
		output:          [][2]int64{{1, 1}},

		closedMinDuration:     1,
		closedMaxDuration:     1,
		closedSlotMinDuration: 1,
		closedSlotMaxDuration: 1,
	}}, {scenario: "TC51", in: []int{31, 43, 11, 13, 2, 3}, out: Expected{
		impressionCount: 3,
		freeTime:        4,
		output:          [][2]int64{{13, 13}, {13, 13}, {13, 13}},

		closedMinDuration:     35,
		closedMaxDuration:     40,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC52", in: []int{68, 72, 12, 18, 2, 4}, out: Expected{
		impressionCount: 3,
		freeTime:        0,
		output:          [][2]int64{{12, 18}, {12, 18}, {12, 18}, {12, 18}},

		closedMinDuration:     70,
		closedMaxDuration:     70,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 15,
	}},
}

func TestGetImpressions(t *testing.T) {
	for _, impTest := range impressionsTests {
		t.Run(impTest.scenario, func(t *testing.T) {
			p := newTestPod(int64(impTest.in[0]), int64(impTest.in[1]), impTest.in[2], impTest.in[3], impTest.in[4], impTest.in[5])
			cfg, _ := getImpressions(p.podMinDuration, p.podMaxDuration, p.vPod)
			expected := impTest.out

			// assert.Equal(t, expected.impressionCount, len(pod.Slots), "Expected impression count = %v . But Found %v", expectedImpressionCount, len(pod.Slots))
			assert.Equal(t, expected.freeTime, cfg.freeTime, "Expected Free Time = %v . But Found %v", expected.freeTime, cfg.freeTime)
			assert.Equal(t, expected.closedMinDuration, cfg.podMinDuration, "Expected closedMinDuration= %v . But Found %v", expected.closedMinDuration, cfg.podMinDuration)
			assert.Equal(t, expected.closedMaxDuration, cfg.podMaxDuration, "Expected closedMinDuration= %v . But Found %v", expected.closedMaxDuration, cfg.podMaxDuration)
			assert.Equal(t, expected.closedSlotMinDuration, cfg.slotMinDuration, "Expected closedSlotMinDuration= %v . But Found %v", expected.closedSlotMinDuration, cfg.slotMinDuration)
			assert.Equal(t, expected.closedSlotMaxDuration, cfg.slotMaxDuration, "Expected closedSlotMinDuration= %v . But Found %v", expected.closedSlotMaxDuration, cfg.slotMaxDuration)
			assert.Equal(t, expected.output, cfg.Slots, "2darray mismatch")
		})
	}
}

/* Benchmarking Tests */
func BenchmarkGetImpressions(b *testing.B) {
	for _, impTest := range impressionsTests {
		b.Run(impTest.scenario, func(b *testing.B) {
			p := newTestPod(int64(impTest.in[0]), int64(impTest.in[1]), impTest.in[2], impTest.in[3], impTest.in[4], impTest.in[5])
			for n := 0; n < b.N; n++ {
				getImpressions(p.podMinDuration, p.podMaxDuration, p.vPod)
			}
		})
	}
}

func newTestPod(podMinDuration, podMaxDuration int64, slotMinDuration, slotMaxDuration, minAds, maxAds int) *TestAdPod {
	testPod := TestAdPod{}

	pod := openrtb_ext.VideoAdPod{}

	pod.MinDuration = &slotMinDuration
	pod.MaxDuration = &slotMaxDuration
	pod.MinAds = &minAds
	pod.MaxAds = &maxAds

	testPod.vPod = pod
	testPod.podMinDuration = podMinDuration
	testPod.podMaxDuration = podMaxDuration
	return &testPod
}
