package countrycodemapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountryCode(t *testing.T) {
	Load(`
AD,AND
AE,ARE
AF,AFG
`)
	assert.Equal(t, "AND", MapToAlpha3("AD"), "map AD to AND")
	assert.Equal(t, "AE", MapToAlpha2("ARE"), "map ARE to AE")
}

func TestCountryCodeToAlpha3(t *testing.T) {
	c := New()
	c.Load(`
AD,AND
AE,ARE
AF,AFG
`)
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid country code AD",
			input:    "AD",
			expected: "AND",
		},
		{
			name:     "invalid country code XX",
			input:    "XX",
			expected: "",
		},
		{
			name:     "empty country code",
			input:    "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, c.MapToAlpha3(test.input), "map %s to alpha3", test.input)
		})
	}
}

func TestCountryCodeToAlpha2(t *testing.T) {
	c := New()
	c.Load(`
AD,AND
AE,ARE
AF,AFG
`)
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid country code AND",
			input:    "AND",
			expected: "AD",
		},
		{
			name:     "invalid country code XXX",
			input:    "XXX",
			expected: "",
		},
		{
			name:     "empty country code",
			input:    "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, c.MapToAlpha2(test.input), "map %s to alpha2", test.input)
		})
	}
}
