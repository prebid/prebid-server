package geolocation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimezoneToUTCOffset(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		offset   int
		ok       bool
	}{
		{
			name:     "Valid timezone Asia/Shanghai: UTC+8",
			timezone: "Asia/Shanghai",
			offset:   8 * 60,
			ok:       true,
		},
		{
			name:     "Valid timezone Asia/Tokyo: UTC+9",
			timezone: "Asia/Tokyo",
			offset:   9 * 60,
			ok:       true,
		},
		{
			name:     "Valid timezone UTC",
			timezone: "UTC",
			offset:   0,
			ok:       true,
		},
		{
			name:     "Invalid timezone Unknown",
			timezone: "Unknown",
			offset:   0,
			ok:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			offset, err := TimezoneToUTCOffset(test.timezone)
			if test.ok {
				assert.NoError(t, err, "timezone %s should be valid", test.timezone)
				assert.Equal(t, test.offset, offset, "timezone %s should have offset minutes %d", test.timezone, test.offset)
			} else {
				assert.Error(t, err, "timezone %s should be invalid", test.timezone)
			}
		})
	}
}

func TestNilGeoLocation(t *testing.T) {
	loc := NewNilGeoLocation()
	geo, err := loc.Lookup(context.Background(), "")
	assert.NoError(t, err, "nil geolocation should not return error")
	assert.NotNil(t, geo, "nil geolocation should return empty geo info")
}
