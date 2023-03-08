package openrtb_ext

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

type granularityTestData struct {
	json   []byte
	target PriceGranularity
}

func TestGranularityUnmarshal(t *testing.T) {
	testGroups := []struct {
		desc        string
		in          []granularityTestData
		expectError bool
	}{
		{
			desc:        "Unmarshal without error",
			in:          validGranularityTests,
			expectError: false,
		},
		{
			desc: "Malformed json. Expect unmarshall error",
			in: []granularityTestData{
				{json: []byte(`[]`)},
			},
			expectError: true,
		},
	}
	for _, tg := range testGroups {
		for i, tc := range tg.in {
			var resolved PriceGranularity
			err := json.Unmarshal(tc.json, &resolved)

			// Assert validation error
			if tg.expectError && !assert.Errorf(t, err, "%s test case %d", tg.desc, i) {
				continue
			}

			// Assert Targeting.Precision
			assert.Equal(t, tc.target.Precision, resolved.Precision, "%s test case %d", tg.desc, i)

			// Assert Targeting.Ranges
			if assert.Len(t, resolved.Ranges, len(tc.target.Ranges), "%s test case %d", tg.desc, i) {
				expected := make(map[string]struct{}, len(tc.target.Ranges))
				for _, r := range tc.target.Ranges {
					expected[fmt.Sprintf("%2.2f-%2.2f-%2.2f", r.Min, r.Max, r.Increment)] = struct{}{}
				}
				for _, actualRange := range resolved.Ranges {
					targetRange := fmt.Sprintf("%2.2f-%2.2f-%2.2f", actualRange.Min, actualRange.Max, actualRange.Increment)
					_, exists := expected[targetRange]
					assert.True(t, exists, "%s test case %d target.range %s not found", tg.desc, i, targetRange)
				}
			}
		}
	}
}

var validGranularityTests []granularityTestData = []granularityTestData{
	{
		json: []byte(`{"precision": 4, "ranges": [{"min": 0, "max": 5, "increment": 0.1}, {"min": 5, "max":10, "increment":0.5}, {"min":10, "max":20, "increment":1}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(4),
			Ranges: []GranularityRange{
				{Min: 0.0, Max: 5.0, Increment: 0.1},
				{Min: 5.0, Max: 10.0, Increment: 0.5},
				{Min: 10.0, Max: 20.0, Increment: 1.0},
			},
		},
	},
	{
		json: []byte(`{"ranges":[{ "max":5, "increment": 0.05}, {"max": 10, "increment": 0.25}, {"max": 20, "increment": 0.5}]}`),
		target: PriceGranularity{
			Ranges: []GranularityRange{
				{Min: 0.0, Max: 5.0, Increment: 0.05},
				{Min: 0.0, Max: 10.0, Increment: 0.25},
				{Min: 0.0, Max: 20.0, Increment: 0.5},
			},
		},
	},
	{
		json: []byte(`"medium"`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges: []GranularityRange{{
				Min:       0,
				Max:       20,
				Increment: 0.1}},
		},
	},
	{
		json: []byte(`{ "precision": 3, "ranges": [{"max":20, "increment":0.005}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(3),
			Ranges:    []GranularityRange{{Min: 0.0, Max: 20.0, Increment: 0.005}},
		},
	},
	{
		json: []byte(`{"precision": 0, "ranges": [{"max":5, "increment": 1}, {"max": 10, "increment": 2}, {"max": 20, "increment": 5}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(0),
			Ranges: []GranularityRange{
				{Min: 0.0, Max: 5.0, Increment: 1.0},
				{Min: 0.0, Max: 10.0, Increment: 2.0},
				{Min: 0.0, Max: 20.0, Increment: 5.0},
			},
		},
	},
	{
		json: []byte(`{"precision": 2, "ranges": [{"min": 0.5, "max":5, "increment": 0.1}, {"min": 54, "max": 10, "increment": 1}, {"min": -42, "max": 20, "increment": 5}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges: []GranularityRange{
				{Min: 0.5, Max: 5.0, Increment: 0.1},
				{Min: 54.0, Max: 10.0, Increment: 1.0},
				{Min: -42.0, Max: 20.0, Increment: 5.0},
			},
		},
	},
	{
		json:   []byte(`{}`),
		target: PriceGranularity{},
	},
	{
		json: []byte(`{"precision": 2}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
		},
	},
	{
		json: []byte(`{"precision": 2, "ranges":[]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges:    []GranularityRange{},
		},
	},
}
