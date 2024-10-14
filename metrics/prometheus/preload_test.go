package prometheusmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prometheus/client_golang/prometheus"
)

func TestRegisterLabelPermutations(t *testing.T) {
	testCases := []struct {
		description      string
		labelsWithValues map[string][]string
		expectedLabels   []prometheus.Labels
	}{
		{
			description:      "Empty set.",
			labelsWithValues: map[string][]string{},
			expectedLabels:   []prometheus.Labels{},
		},
		{
			description: "Set of 1 label and 1 value.",
			labelsWithValues: map[string][]string{
				"1": {"A"},
			},
			expectedLabels: []prometheus.Labels{
				{"1": "A"},
			},
		},
		{
			description: "Set of 1 label and 2 values.",
			labelsWithValues: map[string][]string{
				"1": {"A", "B"},
			},
			expectedLabels: []prometheus.Labels{
				{"1": "A"},
				{"1": "B"},
			},
		},
		{
			description: "Set of 2 labels and 2 values.",
			labelsWithValues: map[string][]string{
				"1": {"A", "B"},
				"2": {"C", "D"},
			},
			expectedLabels: []prometheus.Labels{
				{"1": "A", "2": "C"},
				{"1": "A", "2": "D"},
				{"1": "B", "2": "C"},
				{"1": "B", "2": "D"},
			},
		},
	}

	for _, test := range testCases {
		resultLabels := []prometheus.Labels{}
		registerLabelPermutations(test.labelsWithValues, func(label prometheus.Labels) {
			resultLabels = append(resultLabels, label)
		})

		assert.ElementsMatch(t, test.expectedLabels, resultLabels)
	}
}
