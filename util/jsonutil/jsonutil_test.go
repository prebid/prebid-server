package jsonutil

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestDropElement(t *testing.T) {
	tests := []struct {
		description     string
		input           []byte
		elementToRemove string
		output          []byte
		errorExpected   bool
		errorContains   string
	}{
		{
			description:     "Drop single element after another element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1,"consented_providers": [1608,765,492]}}`),
			elementToRemove: "consented_providers",
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop single element before another element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492],"test": 1}}`),
			elementToRemove: "consented_providers",
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop single element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1545,2563,1411]}}`),
			elementToRemove: "consented_providers",
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop single element string",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": "test"}}`),
			elementToRemove: "consented_providers",
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop parent element between two elements",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToRemove: "consented_providers_settings",
			output:          []byte(`{"consent": "TESTCONSENT","test": 123}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop parent element before element",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToRemove: "consented_providers_settings",
			output:          []byte(`{"test": 123}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop parent element after element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: "consented_providers_settings",
			output:          []byte(`{"consent": "TESTCONSENT"}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop parent element only",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: "consented_providers_settings",
			output:          []byte(`{}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop element that doesn't exist",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: "test2",
			output:          []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		//Errors
		{
			description:     "Error decode",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": ["123",1,,1365,5678,1545,2563,1411], "test": 1}}`),
			elementToRemove: "consented_providers",
			output:          []byte(``),
			errorExpected:   true,
			errorContains:   "looking for beginning of value",
		},
		{
			description:     "Error malformed",
			input:           []byte(`{consented_providers_settings: {"consented_providers": [1365,5678,1545,2563,1411], "test": 1}}`),
			elementToRemove: "consented_providers",
			output:          []byte(``),
			errorExpected:   true,
			errorContains:   "invalid character",
		},
	}

	for _, tt := range tests {
		res, err := DropElement(tt.input, tt.elementToRemove)

		if tt.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			assert.True(t, strings.Contains(err.Error(), tt.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.Equal(t, tt.output, res, "Result is incorrect")
		}

	}
}

func TestFindElement(t *testing.T) {
	tests := []struct {
		description   string
		input         []byte
		elementToFind string
		output        []byte
		elementFound  bool
	}{
		{
			description:   "Find array element",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1,"consented_providers":[1608,765,492]}}`),
			elementToFind: "consented_providers",
			output:        []byte(`[1608,765,492]`),
			elementFound:  true,
		},
		{
			description:   "Find object element",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings":{"test": 1,"consented_providers": [1608,765,492]}}`),
			elementToFind: "consented_providers_settings",
			output:        []byte(`{"test": 1,"consented_providers": [1608,765,492]}`),
			elementFound:  true,
		},
		{
			description:   "Find element that doesn't exist",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings":{"test": 1,"consented_providers": [1608,765,492]}}`),
			elementToFind: "test_element",
			output:        []byte(nil),
			elementFound:  false,
		},
	}

	for _, tt := range tests {
		exists, res, err := FindElement(tt.input, tt.elementToFind)
		assert.NoError(t, err, "Error should be nil")
		assert.Equal(t, tt.output, res, "Result is incorrect")

		if tt.elementFound {
			assert.True(t, exists, "Element must be found")
		} else {
			assert.False(t, exists, "Element must not be found")
		}
	}
}

func TestFindElementIndexes(t *testing.T) {
	tests := []struct {
		description   string
		input         []byte
		elementToFind string
		startIndex    int64
		endIndex      int64
		found         bool
		errorExpected bool
		errorContains string
	}{
		{
			description:   "Find single element after another element",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1,"consented_providers": [1608,765,492]}}`),
			elementToFind: "consented_providers",
			startIndex:    68,
			endIndex:      106,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find single element before another element",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492],"test": 1}}`),
			elementToFind: "consented_providers",
			startIndex:    59,
			endIndex:      97,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find single element",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1545,2563,1411]}}`),
			elementToFind: "consented_providers",
			startIndex:    59,
			endIndex:      98,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find single element string",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": "test"}}`),
			elementToFind: "consented_providers",
			startIndex:    59,
			endIndex:      88,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find parent element between two elements",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToFind: "consented_providers_settings",
			startIndex:    26,
			endIndex:      109,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find parent element before element",
			input:         []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToFind: "consented_providers_settings",
			startIndex:    1,
			endIndex:      84,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find parent element after element",
			input:         []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToFind: "consented_providers_settings",
			startIndex:    25,
			endIndex:      108,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find parent element only",
			input:         []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToFind: "consented_providers_settings",
			startIndex:    1,
			endIndex:      83,
			found:         true,
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Find element that doesn't exist",
			input:         []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToFind: "test2",
			startIndex:    -1,
			endIndex:      -1,
			found:         false,
			errorExpected: false,
			errorContains: "",
		},
		//Errors
		{
			description:   "Error decode",
			input:         []byte(`{"consented_providers_settings": {"consented_providers": ["123",1,,1365,5678,1545,2563,1411], "test": 1}}`),
			elementToFind: "consented_providers",
			startIndex:    -1,
			endIndex:      -1,
			found:         false,
			errorExpected: true,
			errorContains: "looking for beginning of value",
		},
		{
			description:   "Error malformed",
			input:         []byte(`{consented_providers_settings: {"consented_providers": [1365,5678,1545,2563,1411], "test": 1}}`),
			elementToFind: "consented_providers",
			startIndex:    -1,
			endIndex:      -1,
			found:         false,
			errorExpected: true,
			errorContains: "invalid character",
		},
	}

	for _, tt := range tests {
		found, start, end, err := findElementIndexes(tt.input, tt.elementToFind)

		assert.Equal(t, tt.found, found, "Incorrect value of element existence")

		if tt.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			assert.True(t, strings.Contains(err.Error(), tt.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.Equal(t, tt.startIndex, start, "Result is incorrect")
			assert.Equal(t, tt.endIndex, end, "Result is incorrect")
		}

	}
}
