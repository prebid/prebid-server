package privacy

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestUserInProhibitedGeoState(t *testing.T) {
	tests := []struct {
		name             string
		userRegion       string
		deviceRegion     string
		prohibitedStates []string
		expected         bool
	}{
		{
			name:             "empty prohibited list returns false",
			userRegion:       "MD",
			prohibitedStates: []string{},
			expected:         false,
		},
		{
			name:             "nil prohibited list returns false",
			userRegion:       "MD",
			prohibitedStates: nil,
			expected:         false,
		},
		{
			name:             "user geo region matches prohibited state",
			userRegion:       "MD",
			prohibitedStates: []string{"MD", "NJ", "CT", "VA", "OR"},
			expected:         true,
		},
		{
			name:             "device geo region matches prohibited state",
			deviceRegion:     "OR",
			prohibitedStates: []string{"MD", "NJ", "CT", "VA", "OR"},
			expected:         true,
		},
		{
			name:             "case insensitive match",
			userRegion:       "md",
			prohibitedStates: []string{"MD"},
			expected:         true,
		},
		{
			name:             "state not in prohibited list",
			userRegion:       "CA",
			prohibitedStates: []string{"MD", "NJ", "CT", "VA", "OR"},
			expected:         false,
		},
		{
			name:             "no geo data returns false",
			prohibitedStates: []string{"MD", "NJ"},
			expected:         false,
		},
		{
			name:             "user geo takes precedence over non-matching device geo",
			userRegion:       "MD",
			deviceRegion:     "CA",
			prohibitedStates: []string{"MD"},
			expected:         true,
		},
		{
			name:             "device geo matches when user geo does not",
			userRegion:       "CA",
			deviceRegion:     "NJ",
			prohibitedStates: []string{"NJ"},
			expected:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{}

			if tc.userRegion != "" {
				req.User = &openrtb2.User{
					Geo: &openrtb2.Geo{Region: tc.userRegion},
				}
			}

			if tc.deviceRegion != "" {
				req.Device = &openrtb2.Device{
					Geo: &openrtb2.Geo{Region: tc.deviceRegion},
				}
			}

			wrapper := &openrtb_ext.RequestWrapper{BidRequest: req}
			result := UserInProhibitedGeoState(wrapper, tc.prohibitedStates)
			assert.Equal(t, tc.expected, result)
		})
	}
}
