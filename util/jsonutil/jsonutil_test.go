package jsonutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDropElement(t *testing.T) {
	tests := []struct {
		description     string
		input           []byte
		elementToRemove []string
		output          []byte
		errorExpected   bool
		errorContains   string
	}{
		{
			description:     "Drop Single Element After Another Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1,"consented_providers": [1608,765,492]}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Single Element Before Another Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492],"test": 1}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Single Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1545,2563,1411]}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Single Element string",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": "test"}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element Between Two Elements",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{"consent": "TESTCONSENT","test": 123}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element Before Element",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{"test": 123}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element After Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{"consent": "TESTCONSENT"}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element Only",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element List",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1},"data": [{"test1":5},{"test2": [1,2,3]}]}`),
			elementToRemove: []string{"data"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Element That Doesn't Exist",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: []string{"test2"},
			output:          []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Element Single Occurrence",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1},"data": [{"test1":5},{"test2": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers_settings", "test"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492]},"data": [{"test1":5},{"test2": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Element Multiple Occurrence",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers_settings", "test"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492]},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Element Multiple Occurrence Skip Path",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"data": {"amp":1, "test": 25}}}`),
			elementToRemove: []string{"consented_providers_settings", "test"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"data": {"amp":1}}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Single Occurrence",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492]},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Single Occurrence Deep Nested",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true, "nested2": {"test6": 123}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test6"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true, "nested2": {}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Single Occurrence Deep Nested Full Path",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true,"nested2": {"test6": 123}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test", "nested"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested2": {"test6": 123}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Doesn't Exist",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492]},"test":{"nested":true}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test2"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492]},"test":{"nested":true}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		//Errors
		{
			description:     "Error Decode",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": ["123",1,,1365,5678,1545,2563,1411], "test": 1}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(``),
			errorExpected:   true,
			errorContains:   "looking for beginning of value",
		},
		{
			description:     "Error Malformed",
			input:           []byte(`{consented_providers_settings: {"consented_providers": [1365,5678,1545,2563,1411], "test": 1}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(``),
			errorExpected:   true,
			errorContains:   "invalid character",
		},
	}

	for _, tt := range tests {
		res, err := DropElement(tt.input, tt.elementToRemove...)

		if tt.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			assert.True(t, strings.Contains(err.Error(), tt.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.Equal(t, tt.output, res, "Result is incorrect")
		}
	}
}

func TestTryExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		givenErr    string
		expectedMsg string
	}{
		{
			name:        "level-1",
			givenErr:    "readObjectStart: expect { or n, but found m, error found in #1 byte of ...|malformed|..., bigger context ...|malformed|..",
			expectedMsg: "expect { or n, but found m",
		},
		{
			name:        "level-2",
			givenErr:    "openrtb_ext.ExtRequestPrebidCache.Bids: readObjectStart: expect { or n, but found t, error found in #10 byte of ...|:{\"bids\":true}}|..., bigger context ...|{\"cache\":{\"bids\":true}}|...",
			expectedMsg: "cannot unmarshal openrtb_ext.ExtRequestPrebidCache.Bids: expect { or n, but found t",
		},
		{
			name:        "level-3+",
			givenErr:    "openrtb_ext.ExtRequestPrebid.Cache: openrtb_ext.ExtRequestPrebidCache.Bids: readObjectStart: expect { or n, but found t, error found in #10 byte of ...|:{\"bids\":true}}|..., bigger context ...|{\"cache\":{\"bids\":true}}|...",
			expectedMsg: "cannot unmarshal openrtb_ext.ExtRequestPrebidCache.Bids: expect { or n, but found t",
		},
		{
			name:        "error-msg",
			givenErr:    "Skip: do not know how to skip: 109, error found in #10 byte of ...|prebid\": malformed}|..., bigger context ...|{\"prebid\": malformed}|...",
			expectedMsg: "do not know how to skip: 109",
		},
		{
			name:        "specific",
			givenErr:    "openrtb_ext.ExtDevicePrebid.Interstitial: unmarshalerDecoder: request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100, error found in #10 byte of ...|         }\n        }|..., bigger context ...|: 120,\n            \"minheightperc\": 60\n          }\n        }|...",
			expectedMsg: "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100",
		},
		{
			name:        "normal",
			givenErr:    "normal error message",
			expectedMsg: "normal error message",
		},
		{
			name:        "norma-false-start",
			givenErr:    "false: normal error message",
			expectedMsg: "false: normal error message",
		},
		{
			name:        "norma-false-end",
			givenErr:    "normal error message, error found in #10 but doesn't follow format",
			expectedMsg: "normal error message, error found in #10 but doesn't follow format",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := tryExtractErrorMessage(errors.New(test.givenErr))
			assert.Equal(t, test.expectedMsg, result)
		})
	}
}
