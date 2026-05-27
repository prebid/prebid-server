package tmp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountryAlpha3ToAlpha2(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"USA", "US"},
		{"GBR", "GB"},
		{"DEU", "DE"},
		{"FRA", "FR"},
		{"JPN", "JP"},
		{"CAN", "CA"},
		{"AUS", "AU"},
		{"BRA", "BR"},
		{"IND", "IN"},
		{"CHN", "CN"},
		{"unknown", ""},
		{"", ""},
		{"US", ""},  // already alpha-2 — function only accepts alpha-3
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, countryAlpha3ToAlpha2(tc.in))
		})
	}
}
