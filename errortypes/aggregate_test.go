package errortypes

import (
	"errors"
	"testing"

	"github.com/influxdata/influxdb/pkg/testing/assert"
)

func TestAggregateError(t *testing.T) {
	var testCases = []struct {
		description string
		message     string
		errors      []error
		expected    string
	}{
		{
			description: "None",
			message:     "anyMessage",
			errors:      []error{},
			expected:    "",
		},
		{
			description: "One",
			message:     "anyMessage",
			errors:      []error{errors.New("err1")},
			expected:    "anyMessage (1 error):\n  1: err1\n",
		},
		{
			description: "Many",
			message:     "anyMessage",
			errors:      []error{errors.New("err1"), errors.New("err2")},
			expected:    "anyMessage (2 errors):\n  1: err1\n  2: err2\n",
		},
	}

	for _, test := range testCases {
		err := NewAggregateError(test.message, test.errors)
		assert.Equal(t, test.expected, err.Error(), test.description)
	}
}
