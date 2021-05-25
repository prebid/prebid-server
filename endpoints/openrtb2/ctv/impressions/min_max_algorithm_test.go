package impressions

import (
	"sort"
	"testing"

	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/impressions/testdata"
	"github.com/stretchr/testify/assert"
)

type expectedOutputA2 struct {
	step1 [][2]int64 // input passed as is
	step2 [][2]int64 // pod duration = pod max duration, no of ads = maxads
	step3 [][2]int64 // pod duration = pod max duration, no of ads = minads
	step4 [][2]int64 // pod duration = pod min duration, no of ads = maxads
	step5 [][2]int64 // pod duration = pod min duration, no of ads = minads
}

var impressionsTestsA2 = []struct {
	scenario string // Testcase scenario
	//in       []int            // Testcase input
	out expectedOutputA2 // Testcase execpted output
}{
	{scenario: "TC2", out: expectedOutputA2{
		step1: [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}},
		step2: [][2]int64{{11, 13}, {11, 11}, {11, 11}, {11, 11}, {11, 11}, {11, 11}, {11, 11}, {11, 11}},
		step3: [][2]int64{}, // 90 90 15 15 2 2
		step4: [][2]int64{}, // 1,1, 15,15, 8 8
		step5: [][2]int64{}, // 1,1, 15,15, 2 2
	}},
	{scenario: "TC3", out: expectedOutputA2{
		step1: [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}},
		step2: [][2]int64{}, // 90 90 15 15 4 4
		step3: [][2]int64{}, // 90 90 15 15 2 2
		step4: [][2]int64{}, // 1 1 15 15 4 4
		step5: [][2]int64{}, // 1 1 15 15 2 2
	}},
	{scenario: "TC4", out: expectedOutputA2{
		step1: [][2]int64{{15, 15}},
		step2: [][2]int64{{15, 15}}, // 15 15 5 15 1 1
		step3: [][2]int64{{15, 15}}, // 15 15 5 15 1 1
		step4: [][2]int64{{1, 1}},   //  1  1 5 15 1 1
		step5: [][2]int64{{1, 1}},   //  1  1 5 15 1 1
	}},
	{scenario: "TC5", out: expectedOutputA2{
		step1: [][2]int64{{10, 10}, {5, 5}},
		step2: [][2]int64{{10, 10}, {5, 5}}, // 15, 15, 5, 15, 2, 2
		step3: [][2]int64{{15, 15}},         // 15, 15, 5, 15, 1, 1
		step4: [][2]int64{},                 //  1,  1, 5, 15, 2, 2
		step5: [][2]int64{{1, 1}},           //  1,  1, 5, 15, 1, 1
	}},
	{scenario: "TC6", out: expectedOutputA2{
		// 5, 90, 5, 15, 1, 8
		step1: [][2]int64{{15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		// 90, 90, 5, 15, 8, 8
		step2: [][2]int64{{15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		// 90, 90, 5, 15, 1, 1
		step3: [][2]int64{},
		// 1, 1, 5, 15, 8, 8
		step4: [][2]int64{},
		// 1, 1, 5, 15, 1, 1
		step5: [][2]int64{{1, 1}},
	}},
	{scenario: "TC7", out: expectedOutputA2{
		// 15, 30, 10, 15, 1, 1
		step1: [][2]int64{{15, 15}},
		// 30, 30, 10, 15, 1, 1
		step2: [][2]int64{},
		// 30, 30, 10, 15, 1, 1
		step3: [][2]int64{},
		// 15, 15, 10, 15, 1, 1
		step4: [][2]int64{{15, 15}},
		// 15, 15, 10, 15, 1, 1
		step5: [][2]int64{{15, 15}},
	}},
	{scenario: "TC8", out: expectedOutputA2{
		// 35, 35, 10, 35, 3, 40
		step1: [][2]int64{{15, 15}, {10, 10}, {10, 10}},
		// 35, 35, 10, 35, 40, 40
		step2: [][2]int64{},
		// 35, 35, 10, 35, 3, 3
		step3: [][2]int64{{15, 15}, {10, 10}, {10, 10}},
		// 35, 35, 10, 35, 40, 40
		step4: [][2]int64{},
		// 35, 35, 10, 35, 3, 3
		step5: [][2]int64{{15, 15}, {10, 10}, {10, 10}},
	}},
	{scenario: "TC9", out: expectedOutputA2{
		// 35, 35, 10, 35, 6, 40
		step1: [][2]int64{},
		// 35, 35, 10, 35, 40, 40
		step2: [][2]int64{},
		// 35, 35, 10, 35, 6, 6
		step3: [][2]int64{},
		// 35, 35, 10, 35, 40, 40
		step4: [][2]int64{},
		// 35, 35, 10, 35, 6, 6
		step5: [][2]int64{},
	}},
	{scenario: "TC10", out: expectedOutputA2{
		// 35, 65, 10, 35, 6, 40
		step1: [][2]int64{{15, 15}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		// 65, 65, 10, 35, 40, 40
		step2: [][2]int64{},
		// 65, 65, 10, 35, 6, 6
		step3: [][2]int64{{15, 15}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		// 35, 35, 10, 35, 40, 40
		step4: [][2]int64{},
		// 35, 35, 10, 35, 6, 6
		step5: [][2]int64{},
	}},
	{scenario: "TC11", out: expectedOutputA2{
		// 35, 65, 10, 35, 7, 40
		step1: [][2]int64{{9, 11}, {9, 9}, {9, 9}, {9, 9}, {9, 9}, {9, 9}, {9, 9}},
		// 65, 65, 10, 35, 40, 40
		step2: [][2]int64{},
		// 65, 65, 10, 35, 7, 7
		step3: [][2]int64{{9, 11}, {9, 9}, {9, 9}, {9, 9}, {9, 9}, {9, 9}, {9, 9}},
		// 35, 35, 10, 35, 40, 40
		step4: [][2]int64{},
		// 35, 35, 10, 35, 7, 7
		step5: [][2]int64{},
	}},
	{scenario: "TC12", out: expectedOutputA2{
		step1: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		step2: [][2]int64{},
		step3: [][2]int64{{20, 20}, {20, 20}, {15, 15}, {15, 15}, {15, 15}, {15, 15}},
		step4: [][2]int64{},
		step5: [][2]int64{{20, 20}, {20, 20}, {15, 15}, {15, 15}, {15, 15}, {15, 15}},
	}},
	{scenario: "TC13", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC14", out: expectedOutputA2{
		step1: [][2]int64{{5, 9}, {5, 9}, {5, 9}, {5, 9}, {5, 9}, {5, 9}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{{5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}},
		step5: [][2]int64{},
	}},
	{scenario: "TC15", out: expectedOutputA2{
		step1: [][2]int64{{5, 9}, {5, 9}, {5, 9}, {5, 9}, {5, 9}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{{5, 9}, {5, 6}, {5, 5}, {5, 5}, {5, 5}},
		step5: [][2]int64{},
	}},
	{scenario: "TC27", out: expectedOutputA2{
		step1: [][2]int64{{30, 30}, {30, 30}, {30, 30}},
		step2: [][2]int64{{30, 30}, {30, 30}, {30, 30}},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{2, 3}, {2, 2}},
	}},
	{scenario: "TC16", out: expectedOutputA2{
		step1: [][2]int64{{1, 12}, {1, 12}, {1, 12}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		step2: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {1, 6}},
		step3: [][2]int64{},
		step4: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {1, 6}},
		step5: [][2]int64{},
	}},
	{scenario: "TC17", out: expectedOutputA2{
		step1: [][2]int64{{1, 12}, {1, 12}, {1, 12}, {1, 12}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		step2: [][2]int64{{1, 11}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {6, 7}},
		step3: [][2]int64{},
		step4: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {6, 7}},
		step5: [][2]int64{},
	}},
	{scenario: "TC18", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC19", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC20", out: expectedOutputA2{
		step1: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC21", out: expectedOutputA2{
		step1: [][2]int64{{3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}, {3, 9}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC23", out: expectedOutputA2{
		step1: [][2]int64{{4, 14}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{{4, 13}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}},
		step5: [][2]int64{},
	}},
	{scenario: "TC24", out: expectedOutputA2{
		step1: [][2]int64{{60, 69}, {65, 65}},
		step2: [][2]int64{},
		step3: [][2]int64{{60, 69}, {65, 65}},
		step4: [][2]int64{},
		step5: [][2]int64{{60, 69}, {65, 65}},
	}},
	{scenario: "TC25", out: expectedOutputA2{
		step1: [][2]int64{{1, 68}, {20, 20}},
		step2: [][2]int64{{1, 68}, {20, 20}},
		step3: [][2]int64{{1, 68}, {20, 20}},
		step4: [][2]int64{{1, 68}, {20, 20}},
		step5: [][2]int64{{1, 68}, {20, 20}},
	}},
	{scenario: "TC26", out: expectedOutputA2{
		step1: [][2]int64{{45, 45}, {45, 45}},
		step2: [][2]int64{},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{45, 45}, {45, 45}},
	}},
	{scenario: "TC27", out: expectedOutputA2{
		step1: [][2]int64{{30, 30}, {30, 30}, {30, 30}},
		step2: [][2]int64{{30, 30}, {30, 30}, {30, 30}},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{2, 3}, {2, 2}},
	}},
	{scenario: "TC28", out: expectedOutputA2{
		step1: [][2]int64{{30, 30}, {30, 30}, {30, 30}, {30, 30}, {30, 30}, {30, 30}},
		step2: [][2]int64{{30, 30}, {30, 30}, {30, 30}, {30, 30}, {30, 30}, {30, 30}},
		step3: [][2]int64{{90, 90}, {90, 90}},
		step4: [][2]int64{},
		step5: [][2]int64{{2, 3}, {2, 2}},
	}},
	{scenario: "TC29", out: expectedOutputA2{
		step1: [][2]int64{{25, 25}, {20, 20}, {20, 20}},
		step2: [][2]int64{{25, 25}, {20, 20}, {20, 20}},
		step3: [][2]int64{{35, 35}, {30, 30}},
		step4: [][2]int64{},
		step5: [][2]int64{{2, 3}, {2, 2}},
	}},
	{scenario: "TC30", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC31", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC32", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC33", out: expectedOutputA2{
		step1: [][2]int64{{30, 42}, {35, 35}, {35, 35}, {35, 35}},
		step2: [][2]int64{},
		step3: [][2]int64{{30, 42}, {35, 35}, {35, 35}, {35, 35}},
		step4: [][2]int64{},
		step5: [][2]int64{{30, 42}, {35, 35}, {35, 35}, {35, 35}},
	}},
	{scenario: "TC34", out: expectedOutputA2{
		step1: [][2]int64{{30, 30}, {30, 30}, {30, 30}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC35", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC36", out: expectedOutputA2{
		step1: [][2]int64{{45, 45}, {45, 45}},
		step2: [][2]int64{},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{45, 45}, {45, 45}},
	}},
	{scenario: "TC37", out: expectedOutputA2{
		step1: [][2]int64{{25, 25}, {20, 20}},
		step2: [][2]int64{},
		step3: [][2]int64{{25, 25}, {20, 20}},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC38", out: expectedOutputA2{
		step1: [][2]int64{{25, 25}, {25, 25}, {20, 20}, {20, 20}},
		step2: [][2]int64{},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{45, 45}, {45, 45}},
	}},
	{scenario: "TC39", out: expectedOutputA2{
		step1: [][2]int64{{25, 25}, {25, 25}, {20, 20}, {20, 20}},
		step2: [][2]int64{},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{30, 30}, {30, 30}},
	}},
	{scenario: "TC40", out: expectedOutputA2{
		step1: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}},
		step2: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}},
		step3: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}},
		step4: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}},
		step5: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}},
	}},
	{scenario: "TC41", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}},
		step5: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}, {5, 5}},
	}},
	{scenario: "TC42", out: expectedOutputA2{
		step1: [][2]int64{{1, 1}},
		step2: [][2]int64{{1, 1}},
		step3: [][2]int64{{1, 1}},
		step4: [][2]int64{{1, 1}},
		step5: [][2]int64{{1, 1}},
	}},
	{scenario: "TC43", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC44", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC45", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC46", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC47", out: expectedOutputA2{
		step1: [][2]int64{{6, 6}},
		step2: [][2]int64{{6, 6}},
		step3: [][2]int64{{6, 6}},
		step4: [][2]int64{{6, 6}},
		step5: [][2]int64{{6, 6}},
	}},
	{scenario: "TC48", out: expectedOutputA2{
		step1: [][2]int64{{6, 6}, {6, 6}},
		step2: [][2]int64{{6, 6}, {6, 6}},
		step3: [][2]int64{},
		step4: [][2]int64{{6, 6}, {6, 6}},
		step5: [][2]int64{},
	}},
	{scenario: "TC49", out: expectedOutputA2{
		step1: [][2]int64{},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC50", out: expectedOutputA2{
		step1: [][2]int64{{1, 1}},
		step2: [][2]int64{{1, 1}},
		step3: [][2]int64{{1, 1}},
		step4: [][2]int64{{1, 1}},
		step5: [][2]int64{{1, 1}},
	}},
	{scenario: "TC51", out: expectedOutputA2{
		step1: [][2]int64{{13, 13}, {13, 13}, {13, 13}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC52", out: expectedOutputA2{
		step1: [][2]int64{{12, 18}, {12, 18}, {12, 18}, {12, 18}},
		step2: [][2]int64{{12, 18}, {12, 18}, {12, 18}, {12, 18}},
		step3: [][2]int64{},
		step4: [][2]int64{{12, 18}, {12, 18}, {12, 17}, {15, 15}},
		step5: [][2]int64{},
	}},
	{scenario: "TC53", out: expectedOutputA2{
		step1: [][2]int64{{20, 20}, {20, 20}, {20, 20}, {20, 20}, {20, 20}, {20, 20}, {1, 6}},
		step2: [][2]int64{{20, 20}, {20, 20}, {20, 20}, {20, 20}, {20, 20}, {20, 20}, {1, 6}},
		step3: [][2]int64{},
		step4: [][2]int64{{20, 20}, {20, 20}, {20, 20}, {20, 20}, {20, 20}, {20, 20}, {1, 6}},
		step5: [][2]int64{},
	}},
	// {1, 74, 12, 12, 1, 6}
	{scenario: "TC55", out: expectedOutputA2{
		step1: [][2]int64{{12, 12}, {12, 12}, {12, 12}, {12, 12}, {12, 12}, {12, 12}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{},
		step5: [][2]int64{},
	}},
	{scenario: "TC56", out: expectedOutputA2{
		step1: [][2]int64{{126, 126}},
		step2: [][2]int64{{126, 126}},
		step3: [][2]int64{{126, 126}},
		step4: [][2]int64{{126, 126}},
		step5: [][2]int64{{126, 126}},
	}},
	{scenario: "TC57", out: expectedOutputA2{
		step1: [][2]int64{{126, 126}},
		step2: [][2]int64{},
		step3: [][2]int64{{126, 126}},
		step4: [][2]int64{},
		step5: [][2]int64{{126, 126}},
	}},
	{scenario: "TC58", out: expectedOutputA2{
		step1: [][2]int64{{25, 25}, {25, 25}, {20, 20}, {20, 20}},
		step2: [][2]int64{{25, 25}, {25, 25}, {20, 20}, {20, 20}},
		step3: [][2]int64{{45, 45}, {45, 45}},
		step4: [][2]int64{},
		step5: [][2]int64{{15, 15}, {15, 15}},
	}},
	{scenario: "TC59", out: expectedOutputA2{
		step1: [][2]int64{{45, 45}},
		step2: [][2]int64{},
		step3: [][2]int64{},
		step4: [][2]int64{{30, 30}},
		step5: [][2]int64{{30, 30}},
	}},
	// {scenario: "TC1" , out: expectedOutputA2{
	// 	step1: [][2]int64{},
	// 	step2: [][2]int64{},
	// 	step3: [][2]int64{},
	// 	step4: [][2]int64{},
	// 	step5: [][2]int64{},
	// }},
	// Testcases with realistic scenarios

	// {scenario: "TC_3_to_4_Ads_Of_5_to_10_Sec" /*in: []int{15, 40, 5, 10, 3, 4},*/, out: expectedOutputA2{
	// 	// 15, 40, 5, 10, 3, 4
	// 	step1: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}},
	// 	// 40, 40, 5, 10, 4, 4
	// 	step2: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}},
	// 	// 40, 40, 5, 10, 3, 3
	// 	step3: [][2]int64{},
	// 	// 15, 15, 5, 10, 4, 4
	// 	step4: [][2]int64{},
	// 	// 15, 15, 5, 10, 3, 3
	// 	step5: [][2]int64{{5, 5}, {5, 5}, {5, 5}},
	// }},
	// {scenario: "TC_4_to_6_Ads_Of_2_to_25_Sec" /*in: []int{60, 77, 2, 25, 4, 6}, */, out: expectedOutputA2{
	// 	// 60, 77, 2, 25, 4, 6
	// 	step1: [][2]int64{{2, 17}, {15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}},
	// 	// 77, 77, 5, 25, 6, 6
	// 	step2: [][2]int64{{2, 17}, {15, 15}, {15, 15}, {10, 10}, {10, 10}, {10, 10}},
	// 	// 77, 77, 5, 25, 4, 4
	// 	step3: [][2]int64{{25, 25}, {25, 25}, {2, 22}, {5, 5}},
	// 	// 60, 60, 5, 25, 6, 6
	// 	step4: [][2]int64{{10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}, {10, 10}},
	// 	// 60, 60, 5, 25, 4, 4
	// 	step5: [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}},
	// }},

	// {scenario: "TC_2_to_6_ads_of_15_to_45_sec" /*in: []int{60, 90, 15, 45, 2, 6},*/, out: expectedOutputA2{
	// 	// 60, 90, 15, 45, 2, 6
	// 	step1: [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}},
	// 	// 90, 90, 15, 45, 6, 6
	// 	step2: [][2]int64{{15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}, {15, 15}},
	// 	// 90, 90, 15, 45, 2, 2
	// 	step3: [][2]int64{{45, 45}, {45, 45}},
	// 	// 60, 60, 15, 45, 6, 6
	// 	step4: [][2]int64{},
	// 	// 60, 60, 15, 45, 2, 2
	// 	step5: [][2]int64{{30, 30}, {30, 30}},
	// }},

}

func TestGetImpressionsA2(t *testing.T) {
	for _, impTest := range impressionsTestsA2 {
		t.Run(impTest.scenario, func(t *testing.T) {
			in := testdata.Input[impTest.scenario]
			p := newTestPod(int64(in[0]), int64(in[1]), in[2], in[3], in[4], in[5])
			a2 := newMinMaxAlgorithm(p.podMinDuration, p.podMaxDuration, p.vPod)
			expectedMergedOutput := make([][2]int64, 0)
			// explictly looping in order to check result of individual generator
			for step, gen := range a2.generator {
				switch step {
				case 0: // algo1 equaivalent
					assert.Equal(t, impTest.out.step1, gen.Get())
					expectedMergedOutput = appendOptimized(expectedMergedOutput, impTest.out.step1)
					break
				case 1: // pod duration = pod max duration, no of ads = maxads
					assert.Equal(t, impTest.out.step2, gen.Get())
					expectedMergedOutput = appendOptimized(expectedMergedOutput, impTest.out.step2)
					break
				case 2: // pod duration = pod max duration, no of ads = minads
					assert.Equal(t, impTest.out.step3, gen.Get())
					expectedMergedOutput = appendOptimized(expectedMergedOutput, impTest.out.step3)
					break
				case 3: // pod duration = pod min duration, no of ads = maxads
					assert.Equal(t, impTest.out.step4, gen.Get())
					expectedMergedOutput = appendOptimized(expectedMergedOutput, impTest.out.step4)
					break
				case 4: // pod duration = pod min duration, no of ads = minads
					assert.Equal(t, impTest.out.step5, gen.Get())
					expectedMergedOutput = appendOptimized(expectedMergedOutput, impTest.out.step5)
					break
				}

			}
			// also verify merged output
			expectedMergedOutput = testdata.Scenario[impTest.scenario].MinMaxAlgorithm
			out := sortOutput(a2.Get())
			//fmt.Println(out)
			assert.Equal(t, sortOutput(expectedMergedOutput), out)
		})
	}
}

func BenchmarkGetImpressionsA2(b *testing.B) {
	for _, impTest := range impressionsTestsA2 {
		for i := 0; i < b.N; i++ {
			in := testdata.Input[impTest.scenario]
			p := newTestPod(int64(in[0]), int64(in[1]), in[2], in[3], in[4], in[5])
			a2 := newMinMaxAlgorithm(p.podMinDuration, p.podMaxDuration, p.vPod)
			a2.Get()
		}
	}
}

func sortOutput(imps [][2]int64) [][2]int64 {
	sort.Slice(imps, func(i, j int) bool {
		return imps[i][1] < imps[j][1]
	})
	return imps
}

func appendOptimized(slice [][2]int64, elems [][2]int64) [][2]int64 {
	m := make(map[string]int, 0)
	keys := make([]string, 0)
	for _, sel := range slice {
		k := getKey(sel)
		m[k]++
		keys = append(keys, k)
	}
	elemsmap := make(map[string]int, 0)
	for _, ele := range elems {
		elemsmap[getKey(ele)]++
	}

	for k := range elemsmap {
		if elemsmap[k] > m[k] {
			m[k] = elemsmap[k]
		}

		keyPresent := false
		for _, kl := range keys {
			if kl == k {
				keyPresent = true
				break
			}
		}

		if !keyPresent {
			keys = append(keys, k)
		}
	}

	optimized := make([][2]int64, 0)
	for k, v := range m {
		for i := 1; i <= v; i++ {
			optimized = append(optimized, getImpression(k))
		}
	}
	return optimized
}
