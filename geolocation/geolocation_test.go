package geolocation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimezoneToUTCOffset(t *testing.T) {
	tests := []struct {
		timezone string
		offset   int
		failed   bool
	}{
		{"Asia/Shanghai", 8 * 60, false},
		{"Asia/Tokyo", 9 * 60, false},
		{"UTC", 0, false},
		{"Unknown", 0, true},
	}

	for _, test := range tests {
		offset, err := TimezoneToUTCOffset(test.timezone)
		if test.failed {
			assert.Error(t, err, "timezone %s should be invalid", test.timezone)
		} else {
			assert.NoError(t, err, "timezone %s should be valid", test.timezone)
			assert.Equal(t, test.offset, offset, "timezone %s should have offset minutes %d", test.timezone, test.offset)
		}
	}
}

func TestNilGeoLocation(t *testing.T) {
	loc := NewNilGeoLocation()
	geo, err := loc.Lookup(context.Background(), "")
	assert.NoError(t, err, "nil geolocation should not return error")
	assert.NotNil(t, geo, "nil geolocation should return empty geo info")
}
