package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadEmbeddedMap(t *testing.T) {
	testCases := []struct {
		description string
		value       map[string]interface{}
		key         string
		expectedMap map[string]interface{}
		expectedOK  bool
	}{
		{
			description: "Nil",
			value:       nil,
			key:         "",
			expectedMap: nil,
			expectedOK:  false,
		},
		{
			description: "Empty",
			value:       map[string]interface{}{},
			key:         "foo",
			expectedMap: nil,
			expectedOK:  false,
		},
		{
			description: "Success",
			value:       map[string]interface{}{"foo": map[string]interface{}{"bar": 42}},
			key:         "foo",
			expectedMap: map[string]interface{}{"bar": 42},
			expectedOK:  true,
		},
		{
			description: "Not Found",
			value:       map[string]interface{}{"foo": map[string]interface{}{"bar": 42}},
			key:         "notFound",
			expectedMap: nil,
			expectedOK:  false,
		},
		{
			description: "Wrong Type",
			value:       map[string]interface{}{"foo": 42},
			key:         "foo",
			expectedMap: nil,
			expectedOK:  false,
		},
	}

	for _, test := range testCases {
		resultMap, resultOK := ReadEmbeddedMap(test.value, test.key)

		assert.Equal(t, test.expectedMap, resultMap, test.description+":map")
		assert.Equal(t, test.expectedOK, resultOK, test.description+":ok")
	}
}

func TestReadEmbeddedSlice(t *testing.T) {
	testCases := []struct {
		description   string
		value         map[string]interface{}
		key           string
		expectedSlice []interface{}
		expectedOK    bool
	}{
		{
			description:   "Nil",
			value:         nil,
			key:           "",
			expectedSlice: nil,
			expectedOK:    false,
		},
		{
			description:   "Empty",
			value:         map[string]interface{}{},
			key:           "foo",
			expectedSlice: nil,
			expectedOK:    false,
		},
		{
			description:   "Success",
			value:         map[string]interface{}{"foo": []interface{}{42}},
			key:           "foo",
			expectedSlice: []interface{}{42},
			expectedOK:    true,
		},
		{
			description:   "Not Found",
			value:         map[string]interface{}{"foo": []interface{}{42}},
			key:           "notFound",
			expectedSlice: nil,
			expectedOK:    false,
		},
		{
			description:   "Wrong Type",
			value:         map[string]interface{}{"foo": 42},
			key:           "foo",
			expectedSlice: nil,
			expectedOK:    false,
		},
	}

	for _, test := range testCases {
		resultSlice, resultOK := ReadEmbeddedSlice(test.value, test.key)

		assert.Equal(t, test.expectedSlice, resultSlice, test.description+":slicd")
		assert.Equal(t, test.expectedOK, resultOK, test.description+":ok")
	}
}
