package impressions

import (
	"testing"

	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/impressions/testdata"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

type TestAdPod struct {
	vPod           openrtb_ext.VideoAdPod
	podMinDuration int64
	podMaxDuration int64
}

type expected struct {
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
}

var impressionsTests = []struct {
	scenario string   // Testcase scenario
	out      expected // Testcase execpted output
}{
	{scenario: "TC2", out: expected{
		impressionCount:       6,
		freeTime:              0.0,
		closedMinDuration:     5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC3", out: expected{
		impressionCount: 4,
		freeTime:        30.0, closedMinDuration: 5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC4", out: expected{
		impressionCount: 1,
		freeTime:        0.0, closedMinDuration: 5,
		closedMaxDuration:     15,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC5", out: expected{
		impressionCount: 2,
		freeTime:        0.0, closedMinDuration: 5,
		closedMaxDuration:     15,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC6", out: expected{
		impressionCount: 8,
		freeTime:        0.0, closedMinDuration: 5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC7", out: expected{
		impressionCount: 1,
		freeTime:        15.0, closedMinDuration: 15,
		closedMaxDuration:     30,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC8", out: expected{
		impressionCount: 3,
		freeTime:        0.0, closedMinDuration: 35,
		closedMaxDuration:     35,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC9", out: expected{
		impressionCount: 0,
		freeTime:        35, closedMinDuration: 35,
		closedMaxDuration:     35,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC10", out: expected{
		impressionCount: 6,
		freeTime:        0.0, closedMinDuration: 35,
		closedMaxDuration:     65,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC11", out: expected{
		impressionCount: 0, //7,
		freeTime:        0, closedMinDuration: 35,
		closedMaxDuration:     65,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC12", out: expected{
		impressionCount: 10,
		freeTime:        0.0, closedMinDuration: 100,
		closedMaxDuration:     100,
		closedSlotMinDuration: 10,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC13", out: expected{
		impressionCount: 0,
		freeTime:        60, closedMinDuration: 60,
		closedMaxDuration:     60,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC14", out: expected{
		impressionCount: 6,
		freeTime:        6, closedMinDuration: 30,
		closedMaxDuration:     60,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC15", out: expected{
		impressionCount: 5,
		freeTime:        15, closedMinDuration: 30,
		closedMaxDuration:     60,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC16", out: expected{
		impressionCount: 13,
		freeTime:        0, closedMinDuration: 126,
		closedMaxDuration:     126,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC17", out: expected{
		impressionCount: 13,
		freeTime:        0, closedMinDuration: 130,
		closedMaxDuration:     125,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC18", out: expected{
		impressionCount: 0,
		freeTime:        125, closedMinDuration: 125,
		closedMaxDuration:     125,
		closedSlotMinDuration: 4,
		closedSlotMaxDuration: 4,
	}},
	{scenario: "TC19", out: expected{
		impressionCount: 0,
		freeTime:        90, closedMinDuration: 90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 7, // overlapping case. Hence as is
		closedSlotMaxDuration: 9,
	}},
	{scenario: "TC20", out: expected{
		impressionCount: 9,
		freeTime:        0, closedMinDuration: 90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 10,
	}},
	{scenario: "TC21", out: expected{
		impressionCount: 9,
		freeTime:        89, closedMinDuration: 5,
		closedMaxDuration:     170,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 5,
	}},
	{scenario: "TC23", out: expected{
		impressionCount: 12,
		freeTime:        0, closedMinDuration: 120,
		closedMaxDuration:     120,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 15,
	}},
	{scenario: "TC24", out: expected{
		impressionCount: 2,
		freeTime:        0, closedMinDuration: 134,
		closedMaxDuration:     134,
		closedSlotMinDuration: 60,
		closedSlotMaxDuration: 90,
	}},
	{scenario: "TC25", out: expected{
		impressionCount: 2,
		freeTime:        0,

		closedMinDuration:     88,
		closedMaxDuration:     88,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 80,
	}},
	{scenario: "TC26", out: expected{
		impressionCount: 2,
		freeTime:        0,

		closedMinDuration:     90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 45,
		closedSlotMaxDuration: 45,
	}},
	{scenario: "TC27", out: expected{
		impressionCount: 3,
		freeTime:        0,

		closedMinDuration:     5,
		closedMaxDuration:     90,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 45,
	}},
	{scenario: "TC28", out: expected{
		impressionCount: 6,
		freeTime:        0,

		closedMinDuration:     5,
		closedMaxDuration:     180,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 90,
	}},
	{scenario: "TC29", out: expected{
		impressionCount: 3,
		freeTime:        0, closedMinDuration: 5,
		closedMaxDuration:     65,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 35,
	}},
	{scenario: "TC30", out: expected{
		impressionCount: 3,
		freeTime:        123, closedMinDuration: 123,
		closedMaxDuration:     123,
		closedSlotMinDuration: 34,
		closedSlotMaxDuration: 34,
	}},
	{scenario: "TC31", out: expected{
		impressionCount: 3,
		freeTime:        123, closedMinDuration: 123,
		closedMaxDuration:     123,
		closedSlotMinDuration: 31,
		closedSlotMaxDuration: 31,
	}}, {scenario: "TC32", out: expected{
		impressionCount: 0,
		freeTime:        134, closedMinDuration: 134,
		closedMaxDuration:     134,
		closedSlotMinDuration: 63,
		closedSlotMaxDuration: 63,
	}},
	{scenario: "TC33", out: expected{
		impressionCount: 4,
		freeTime:        0, closedMinDuration: 147,
		closedMaxDuration:     147,
		closedSlotMinDuration: 30,
		closedSlotMaxDuration: 60,
	}},
	{scenario: "TC34", out: expected{
		impressionCount: 3,
		freeTime:        12, closedMinDuration: 90,
		closedMaxDuration:     100,
		closedSlotMinDuration: 30,
		closedSlotMaxDuration: 30,
	}}, {scenario: "TC35", out: expected{
		impressionCount: 0,
		freeTime:        102, closedMinDuration: 90,
		closedMaxDuration:     100,
		closedSlotMinDuration: 30,
		closedSlotMaxDuration: 40,
	}}, {scenario: "TC36", out: expected{
		impressionCount: 2,
		freeTime:        0, closedMinDuration: 90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 45,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC37", out: expected{
		impressionCount: 2,
		freeTime:        0, closedMinDuration: 10,
		closedMaxDuration:     45,
		closedSlotMinDuration: 20,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC38", out: expected{
		impressionCount: 0,
		freeTime:        0, closedMinDuration: 90,
		closedMaxDuration:     90,
		closedSlotMinDuration: 20,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC39", out: expected{
		impressionCount: 4,
		freeTime:        0, closedMinDuration: 60,
		closedMaxDuration:     90,
		closedSlotMinDuration: 20,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC40", out: expected{
		impressionCount: 10,
		freeTime:        0, closedMinDuration: 95,
		closedMaxDuration:     95,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC41", out: expected{
		impressionCount: 0,
		freeTime:        123, closedMinDuration: 95,
		closedMaxDuration:     120,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 45,
	}}, {scenario: "TC42", out: expected{
		impressionCount: 1,
		freeTime:        0, closedMinDuration: 1,
		closedMaxDuration:     1,
		closedSlotMinDuration: 1,
		closedSlotMaxDuration: 1,
	}}, {scenario: "TC43", out: expected{
		impressionCount: 0,
		freeTime:        2, closedMinDuration: 2,
		closedMaxDuration:     2,
		closedSlotMinDuration: 2,
		closedSlotMaxDuration: 2,
	}}, {scenario: "TC44", out: expected{
		impressionCount: 0,
		freeTime:        0, closedMinDuration: 0,
		closedMaxDuration:     0,
		closedSlotMinDuration: 0,
		closedSlotMaxDuration: 0,
	}}, {scenario: "TC45", out: expected{
		impressionCount: 0,
		freeTime:        0, closedMinDuration: 5,
		closedMaxDuration:     -5,
		closedSlotMinDuration: -3, // overlapping hence will as is
		closedSlotMaxDuration: -4,
	}}, {scenario: "TC46", out: expected{
		impressionCount: 0,
		freeTime:        0, closedMinDuration: -1,
		closedMaxDuration:     -1,
		closedSlotMinDuration: -1,
		closedSlotMaxDuration: -1,
	}}, {scenario: "TC47", out: expected{
		impressionCount: 1,
		freeTime:        0, closedMinDuration: 6,
		closedMaxDuration:     6,
		closedSlotMinDuration: 6,
		closedSlotMaxDuration: 6,
	}}, {scenario: "TC48", out: expected{
		impressionCount: 2,
		freeTime:        0, closedMinDuration: 12,
		closedMaxDuration:     12,
		closedSlotMinDuration: 6,
		closedSlotMaxDuration: 6,
	}}, {scenario: "TC49", out: expected{
		impressionCount: 0,
		freeTime:        12, closedMinDuration: 12,
		closedMaxDuration:     12,
		closedSlotMinDuration: 7,
		closedSlotMaxDuration: 7,
	}}, {scenario: "TC50", out: expected{
		impressionCount: 0,
		freeTime:        0, closedMinDuration: 1,
		closedMaxDuration:     1,
		closedSlotMinDuration: 1,
		closedSlotMaxDuration: 1,
	}}, {scenario: "TC51", out: expected{
		impressionCount: 3,
		freeTime:        4, closedMinDuration: 35,
		closedMaxDuration:     40,
		closedSlotMinDuration: 11,
		closedSlotMaxDuration: 13,
	}},
	{scenario: "TC52", out: expected{
		impressionCount: 3,
		freeTime:        0, closedMinDuration: 70,
		closedMaxDuration:     70,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 15,
	}}, {scenario: "TC53", out: expected{
		impressionCount: 3,
		freeTime:        0, closedMinDuration: 126,
		closedMaxDuration:     126,
		closedSlotMinDuration: 5,
		closedSlotMaxDuration: 20,
	}}, {scenario: "TC55", out: expected{
		impressionCount: 6,
		freeTime:        2, closedMinDuration: 1,
		closedMaxDuration:     74,
		closedSlotMinDuration: 12,
		closedSlotMaxDuration: 12,
	}}, {scenario: "TC56", out: expected{
		impressionCount: 1,
		freeTime:        0, closedMinDuration: 126,
		closedMaxDuration:     126,
		closedSlotMinDuration: 126,
		closedSlotMaxDuration: 126,
	}}, {scenario: "TC57", out: expected{
		impressionCount: 1,
		freeTime:        0, closedMinDuration: 126,
		closedMaxDuration:     126,
		closedSlotMinDuration: 126,
		closedSlotMaxDuration: 126,
	}}, {scenario: "TC58", out: expected{
		impressionCount: 4,
		freeTime:        0, closedMinDuration: 30,
		closedMaxDuration:     90,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 45,
	}},
	{scenario: "TC59", out: expected{
		impressionCount: 1,
		freeTime:        45, closedMinDuration: 30,
		closedMaxDuration:     90,
		closedSlotMinDuration: 15,
		closedSlotMaxDuration: 45,
	}},
}

func TestGetImpressionsA1(t *testing.T) {
	for _, impTest := range impressionsTests {
		t.Run(impTest.scenario, func(t *testing.T) {
			in := testdata.Input[impTest.scenario]
			p := newTestPod(int64(in[0]), int64(in[1]), in[2], in[3], in[4], in[5])

			cfg := newMaximizeForDuration(p.podMinDuration, p.podMaxDuration, p.vPod)
			imps := cfg.Get()
			expected := impTest.out
			expectedImpressionBreak := testdata.Scenario[impTest.scenario].MaximizeForDuration
			// assert.Equal(t, expected.impressionCount, len(pod.Slots), "expected impression count = %v . But Found %v", expectedImpressionCount, len(pod.Slots))
			assert.Equal(t, expected.freeTime, cfg.freeTime, "expected Free Time = %v . But Found %v", expected.freeTime, cfg.freeTime)
			// assert.Equal(t, expected.closedMinDuration, cfg.requested.podMinDuration, "expected closedMinDuration= %v . But Found %v", expected.closedMinDuration, cfg.requested.podMinDuration)
			// assert.Equal(t, expected.closedMaxDuration, cfg.requested.podMaxDuration, "expected closedMinDuration= %v . But Found %v", expected.closedMaxDuration, cfg.requested.podMaxDuration)
			assert.Equal(t, expected.closedSlotMinDuration, cfg.internal.slotMinDuration, "expected closedSlotMinDuration= %v . But Found %v", expected.closedSlotMinDuration, cfg.internal.slotMinDuration)
			assert.Equal(t, expected.closedSlotMaxDuration, cfg.internal.slotMaxDuration, "expected closedSlotMinDuration= %v . But Found %v", expected.closedSlotMaxDuration, cfg.internal.slotMaxDuration)
			assert.Equal(t, expectedImpressionBreak, imps, "2darray mismatch")
			assert.Equal(t, MaximizeForDuration, cfg.Algorithm())
		})
	}
}

/* Benchmarking Tests */
func BenchmarkGetImpressions(b *testing.B) {
	for _, impTest := range impressionsTests {
		b.Run(impTest.scenario, func(b *testing.B) {
			in := testdata.Input[impTest.scenario]
			p := newTestPod(int64(in[0]), int64(in[1]), in[2], in[3], in[4], in[5])
			for n := 0; n < b.N; n++ {
				cfg := newMaximizeForDuration(p.podMinDuration, p.podMaxDuration, p.vPod)
				cfg.Get()
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
