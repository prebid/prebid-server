package countrycode

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
	assert.Equal(t, "AND", ToAlpha3("AD"), "map AD to AND")
	assert.Equal(t, "AE", ToAlpha2("ARE"), "map ARE to AE")
}

func TestCountryCodeToAlpha3(t *testing.T) {
	c := New()
	c.Load(`
AD,AND
AE,ARE
AF,AFG
`)
	tests := []struct {
		input    string
		expected string
	}{
		{"AD", "AND"},
		{"AE", "ARE"},
		{"XX", ""},
		{"", ""},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, c.ToAlpha3(test.input), "map %s to alpha3", test.input)
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
		input    string
		expected string
	}{
		{"AND", "AD"},
		{"ARE", "AE"},
		{"", ""},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, c.ToAlpha2(test.input), "map %s to alpha2", test.input)
	}
}
