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

func TestReadEmbeddedString(t *testing.T) {
	testCases := []struct {
		description    string
		value          map[string]interface{}
		key            string
		expectedString string
		expectedOK     bool
	}{
		{
			description:    "Nil",
			value:          nil,
			key:            "",
			expectedString: "",
			expectedOK:     false,
		},
		{
			description:    "Empty",
			value:          map[string]interface{}{},
			key:            "foo",
			expectedString: "",
			expectedOK:     false,
		},
		{
			description:    "Success",
			value:          map[string]interface{}{"foo": "stringValue"},
			key:            "foo",
			expectedString: "stringValue",
			expectedOK:     true,
		},
		{
			description:    "Not Found",
			value:          map[string]interface{}{"foo": "stringValue"},
			key:            "notFound",
			expectedString: "",
			expectedOK:     false,
		},
		{
			description:    "Wrong Type",
			value:          map[string]interface{}{"foo": []interface{}{42}},
			key:            "foo",
			expectedString: "",
			expectedOK:     false,
		},
	}

	for _, test := range testCases {
		resultString, resultOK := ReadEmbeddedString(test.value, test.key)

		assert.Equal(t, test.expectedString, resultString, test.description+":string")
		assert.Equal(t, test.expectedOK, resultOK, test.description+":ok")
	}
}

func TestHasElement(t *testing.T) {
	testCases := []struct {
		description string
		value       map[string]interface{}
		keys        []string
		expected    bool
	}{
		{
			description: "Level 1 - Exists",
			value:       map[string]interface{}{"foo": "exists"},
			keys:        []string{"foo"},
			expected:    true,
		},
		{
			description: "Level 1 - Does Not Exist",
			value:       map[string]interface{}{"foo": "exists"},
			keys:        []string{"doesnotexist"},
			expected:    false,
		},
		{
			description: "Level 2 - Exists",
			value:       map[string]interface{}{"foo": map[string]interface{}{"bar": "exists"}},
			keys:        []string{"foo", "bar"},
			expected:    true,
		},
		{
			description: "Level 2 - Top Level Does Not Exist",
			value:       map[string]interface{}{"foo": map[string]interface{}{"bar": "exists"}},
			keys:        []string{"doesnotexist", "bar"},
			expected:    false,
		},
		{
			description: "Level 2 - Lower Level Does Not Exist",
			value:       map[string]interface{}{"foo": map[string]interface{}{"bar": "exists"}},
			keys:        []string{"foo", "doesnotexist"},
			expected:    false,
		},
		{
			description: "Level 2 - Does Not Exist At All",
			value:       map[string]interface{}{"foo": map[string]interface{}{"bar": "exists"}},
			keys:        []string{"doesnotexist", "doesnotexist"},
			expected:    false,
		},
		{
			description: "Keys Nil",
			value:       map[string]interface{}{"foo": "exists"},
			keys:        nil,
			expected:    false,
		},
		{
			description: "Keys Empty",
			value:       map[string]interface{}{"foo": "exists"},
			keys:        []string{},
			expected:    false,
		},
		{
			description: "Map Nil",
			value:       nil,
			keys:        []string{"foo"},
			expected:    false,
		},
		{
			description: "Map Empty",
			value:       map[string]interface{}{},
			keys:        []string{"foo"},
			expected:    false,
		},
		{
			description: "Nil",
			value:       nil,
			keys:        nil,
			expected:    false,
		},
	}

	for _, test := range testCases {
		result := HasElement(test.value, test.keys...)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestCloneMap(t *testing.T) {
	// Test we handle nils properly
	t.Run("NilMap", func(t *testing.T) {
		var testMap, copyMap map[string]string = nil, nil // copyMap is a manual copy of testMap
		clone := Clone(testMap)
		testMap = map[string]string{"foo": "bar"}
		assert.Equal(t, copyMap, clone)
	})
	// Test a simple string map
	t.Run("StringMap", func(t *testing.T) {
		var testMap, copyMap map[string]string = map[string]string{"foo": "bar", "first": "one"}, map[string]string{"foo": "bar", "first": "one"}
		clone := Clone(testMap)
		testMap["foo"] = "baz"
		testMap["bozo"] = "the clown"
		assert.Equal(t, copyMap, clone)
	})
	// Test a simple map[string]int
	t.Run("StringInt", func(t *testing.T) {
		var testMap, copyMap map[string]int = map[string]int{"foo": 1, "first": 2}, map[string]int{"foo": 1, "first": 2}
		clone := Clone(testMap)
		testMap["foo"] = 7
		testMap["bozo"] = 13
		assert.Equal(t, copyMap, clone)
	})
}
