package adservertargeting

import (
	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplitAndGet(t *testing.T) {

	testCases := []struct {
		description   string
		inputPath     string
		inputData     []byte
		inputDelim    string
		expectedValue string
		expectedError bool
	}{
		{
			description:   "get value from valid input",
			inputData:     []byte(`{"site": {"page": "test.com"}}`),
			inputPath:     "site.page",
			inputDelim:    ".",
			expectedValue: "test.com",
			expectedError: false,
		},
		{
			description:   "get value from invalid input",
			inputData:     nil,
			inputPath:     "site.page",
			inputDelim:    ".",
			expectedValue: "",
			expectedError: true,
		},
	}

	for _, test := range testCases {
		res, err := splitAndGet(test.inputPath, test.inputData, test.inputDelim)

		assert.Equal(t, test.expectedValue, res, "incorrect result")

		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestVerifyPrefixAndTrimFound(t *testing.T) {

	testCases := []struct {
		description   string
		inputPath     string
		inputPrefix   string
		expectedValue string
		valueFound    bool
	}{
		{
			description:   "get existing value ",
			inputPath:     "site.page.id",
			inputPrefix:   "site.page.",
			expectedValue: "id",
			valueFound:    true,
		},
		{
			description:   "get non-existing value",
			inputPath:     "site.page.id",
			inputPrefix:   "incorrect",
			expectedValue: "",
			valueFound:    false,
		},
	}

	for _, test := range testCases {
		res, found := verifyPrefixAndTrim(test.inputPath, test.inputPrefix)
		assert.Equal(t, test.valueFound, found, "found value incorrect")
		assert.Equal(t, test.expectedValue, res, "incorrect returned result")
	}
}

func TestTypedLookup(t *testing.T) {

	testCases := []struct {
		description   string
		inputData     []byte
		inputPath     string
		inputKeys     []string
		expectedValue []byte
		expectedError bool
	}{
		{
			description:   "lookup existing value",
			inputData:     []byte(`{"site": {"page": "test.com"}}`),
			inputPath:     "site.page",
			inputKeys:     []string{"site", "page"},
			expectedValue: []byte(`test.com`),
			expectedError: false,
		},
		{
			description:   "lookup non-existing value",
			inputData:     []byte(`{"site": {"page": "test.com"}}`),
			inputPath:     "site.page",
			inputKeys:     []string{"site", "id"},
			expectedValue: []byte(nil),
			expectedError: true,
		},
		{
			description:   "lookup value in incorrect json",
			inputData:     []byte(`[`),
			inputPath:     "site.page",
			inputKeys:     []string{"site", "page"},
			expectedValue: []byte(nil),
			expectedError: true,
		},
		{
			description:   "lookup object value",
			inputData:     []byte(`{"site": {"page": "test.com"}}`),
			inputPath:     "site",
			inputKeys:     []string{"site"},
			expectedValue: []byte(nil),
			expectedError: true,
		},
	}

	for _, test := range testCases {

		res, err := typedLookup(test.inputData, test.inputPath, test.inputKeys...)

		assert.Equal(t, test.expectedValue, res, "incorrect returned result")

		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestVerifyType(t *testing.T) {

	testCases := []struct {
		description string
		inputValue  jsonparser.ValueType
		expectedRes bool
	}{
		{
			description: "verify correct value",
			inputValue:  jsonparser.String,
			expectedRes: true,
		},
		{
			description: "verify correct value",
			inputValue:  jsonparser.Object,
			expectedRes: false,
		},
	}

	for _, test := range testCases {
		correctType := verifyType(test.inputValue)
		assert.Equal(t, test.expectedRes, correctType, "incorrect verified type result")

	}
}
