package iosutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIOSVersion(t *testing.T) {
	tests := []struct {
		description     string
		given           string
		expectedVersion IOSVersion
		expectedError   string
	}{
		{
			description:     "Valid",
			given:           "14.2",
			expectedVersion: IOSVersion{Major: 14, Minor: 2},
		},
		{
			description:   "Invalid Parts - Empty",
			given:         "",
			expectedError: "expected major.minor format",
		},
		{
			description:   "Invalid Parts - Too Few",
			given:         "14",
			expectedError: "expected major.minor format",
		},
		{
			description:   "Invalid Parts - Too Many",
			given:         "14.2.1",
			expectedError: "expected major.minor format",
		},
		{
			description:   "Invalid Major",
			given:         "xxx.2",
			expectedError: "major version is not an integer",
		},
		{
			description:   "Invalid Minor",
			given:         "14.xxx",
			expectedError: "minor version is not an integer",
		},
	}

	for _, test := range tests {
		version, err := ParseIOSVersion(test.given)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}

		assert.Equal(t, test.expectedVersion, version, test.description+":version")
	}
}

func TestEqualOrGreater(t *testing.T) {
	givenMajor := 14
	givenMinor := 2

	tests := []struct {
		description  string
		givenVersion IOSVersion
		expected     bool
	}{
		{
			description:  "Less Than By Major + Minor",
			givenVersion: IOSVersion{Major: 13, Minor: 1},
			expected:     false,
		},
		{
			description:  "Less Than By Major",
			givenVersion: IOSVersion{Major: 13, Minor: 2},
			expected:     false,
		},
		{
			description:  "Less Than By Minor",
			givenVersion: IOSVersion{Major: 14, Minor: 1},
			expected:     false,
		},
		{
			description:  "Equal",
			givenVersion: IOSVersion{Major: 14, Minor: 2},
			expected:     true,
		},
		{
			description:  "Greater By Major + Minor",
			givenVersion: IOSVersion{Major: 15, Minor: 3},
			expected:     true,
		},
		{
			description:  "Greater By Major",
			givenVersion: IOSVersion{Major: 15, Minor: 2},
			expected:     true,
		},
		{
			description:  "Greater By Minor",
			givenVersion: IOSVersion{Major: 14, Minor: 3},
			expected:     true,
		},
	}

	for _, test := range tests {
		result := test.givenVersion.EqualOrGreater(givenMajor, givenMinor)
		assert.Equal(t, test.expected, result, test.description)
	}
}
