package iosutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		description     string
		given           string
		expectedVersion Version
		expectedError   string
	}{
		{
			description:     "Valid - major.minor format",
			given:           "14.2",
			expectedVersion: Version{Major: 14, Minor: 2},
		},
		{
			description:     "Valid - major.minor.patch format",
			given:           "14.2.1",
			expectedVersion: Version{Major: 14, Minor: 2},
		},
		{
			description:   "Invalid Parts - Empty",
			given:         "",
			expectedError: "expected either major.minor or major.minor.patch format",
		},
		{
			description:   "Invalid Parts - Too Few",
			given:         "14",
			expectedError: "expected either major.minor or major.minor.patch format",
		},
		{
			description:   "Invalid Parts - Too Many",
			given:         "14.2.1.3",
			expectedError: "expected either major.minor or major.minor.patch format",
		},
		{
			description:   "Invalid Parts - Too Few",
			given:         "14",
			expectedError: "expected either major.minor or major.minor.patch format",
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
		version, err := ParseVersion(test.given)

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
		givenVersion Version
		expected     bool
	}{
		{
			description:  "Less Than By Major + Minor",
			givenVersion: Version{Major: 13, Minor: 1},
			expected:     false,
		},
		{
			description:  "Less Than By Major",
			givenVersion: Version{Major: 13, Minor: 2},
			expected:     false,
		},
		{
			description:  "Less Than By Minor",
			givenVersion: Version{Major: 14, Minor: 1},
			expected:     false,
		},
		{
			description:  "Equal",
			givenVersion: Version{Major: 14, Minor: 2},
			expected:     true,
		},
		{
			description:  "Greater By Major + Minor",
			givenVersion: Version{Major: 15, Minor: 3},
			expected:     true,
		},
		{
			description:  "Greater By Major",
			givenVersion: Version{Major: 15, Minor: 2},
			expected:     true,
		},
		{
			description:  "Greater By Minor",
			givenVersion: Version{Major: 14, Minor: 3},
			expected:     true,
		},
	}

	for _, test := range tests {
		result := test.givenVersion.EqualOrGreater(givenMajor, givenMinor)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestEqual(t *testing.T) {
	givenMajor := 14
	givenMinor := 2

	tests := []struct {
		description  string
		givenVersion Version
		expected     bool
	}{
		{
			description:  "Less Than By Major + Minor",
			givenVersion: Version{Major: 13, Minor: 1},
			expected:     false,
		},
		{
			description:  "Less Than By Major",
			givenVersion: Version{Major: 13, Minor: 2},
			expected:     false,
		},
		{
			description:  "Less Than By Minor",
			givenVersion: Version{Major: 14, Minor: 1},
			expected:     false,
		},
		{
			description:  "Equal",
			givenVersion: Version{Major: 14, Minor: 2},
			expected:     true,
		},
		{
			description:  "Greater By Major + Minor",
			givenVersion: Version{Major: 15, Minor: 3},
			expected:     false,
		},
		{
			description:  "Greater By Major",
			givenVersion: Version{Major: 15, Minor: 2},
			expected:     false,
		},
		{
			description:  "Greater By Minor",
			givenVersion: Version{Major: 14, Minor: 3},
			expected:     false,
		},
	}

	for _, test := range tests {
		result := test.givenVersion.Equal(givenMajor, givenMinor)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestDetectVersionClassification(t *testing.T) {

	tests := []struct {
		given    string
		expected VersionClassification
	}{
		{
			given:    "13.0",
			expected: VersionUnknown,
		},
		{
			given:    "14.0",
			expected: Version140,
		},
		{
			given:    "14.0.1",
			expected: Version140,
		},
		{
			given:    "14.1",
			expected: Version141,
		},
		{
			given:    "14.1.2",
			expected: Version141,
		},
		{
			given:    "14.2",
			expected: Version142OrGreater,
		},
		{
			given:    "14.2.3",
			expected: Version142OrGreater,
		},
		{
			given:    "14.3",
			expected: Version142OrGreater,
		},
		{
			given:    "14.3.2",
			expected: Version142OrGreater,
		},
		{
			given:    "15.0",
			expected: Version142OrGreater,
		},
		{
			given:    "15.0.1",
			expected: Version142OrGreater,
		},
	}

	for _, test := range tests {
		result := DetectVersionClassification(test.given)
		assert.Equal(t, test.expected, result, test.given)
	}
}
