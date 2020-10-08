package exchange

import (
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/assert"
)

func TestGetActiveBidders(t *testing.T) {
	active := adapters.BidderInfo{Status: adapters.StatusActive}
	disabled := adapters.BidderInfo{Status: adapters.StatusDisabled}
	unknown := adapters.BidderInfo{Status: adapters.StatusUnknown}

	testCases := []struct {
		description string
		bidderInfos map[string]adapters.BidderInfo
		expected    map[string]struct{}
	}{
		{
			description: "Active",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": active},
			expected:    map[string]struct{}{"appnexus": {}},
		},
		{
			description: "Disabled",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": disabled},
			expected:    map[string]struct{}{},
		},
		{
			description: "Unknown",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": unknown},
			expected:    map[string]struct{}{"appnexus": {}},
		},
		{
			description: "Mixed",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": disabled, "openx": active, "rubicon": unknown},
			expected:    map[string]struct{}{"openx": {}, "rubicon": {}},
		},
	}

	for _, test := range testCases {
		result := GetActiveBidders(test.bidderInfos)
		assert.Equal(t, test.expected, result, test.description)
	}
}
