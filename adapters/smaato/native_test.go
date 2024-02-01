package smaato

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractAdmNative(t *testing.T) {
	tests := []struct {
		testName         string
		adMarkup         string
		expectedAdMarkup string
		expectedError    string
	}{
		{
			testName:         "extract native",
			adMarkup:         "{\"native\":{\"assets\":[]}}",
			expectedAdMarkup: `{"assets":[]}`,
			expectedError:    "",
		},
		{
			testName:         "invalid adMarkup",
			adMarkup:         "{",
			expectedAdMarkup: "",
			expectedError:    "Invalid ad markup {.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			adMarkup, err := extractAdmNative(tt.adMarkup)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedAdMarkup, adMarkup)
		})
	}
}
