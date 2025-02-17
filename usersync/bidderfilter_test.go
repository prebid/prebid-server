package usersync

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpecificBidderFilter(t *testing.T) {
	bidder := "a"

	testCases := []struct {
		description string
		bidders     []string
		mode        BidderFilterMode
		expected    bool
	}{
		{
			description: "Include - None",
			bidders:     []string{},
			mode:        BidderFilterModeInclude,
			expected:    false,
		},
		{
			description: "Include - One",
			bidders:     []string{bidder},
			mode:        BidderFilterModeInclude,
			expected:    true,
		},
		{
			description: "Include - Many",
			bidders:     []string{"other", bidder},
			mode:        BidderFilterModeInclude,
			expected:    true,
		},
		{
			description: "Include - Other",
			bidders:     []string{"other"},
			mode:        BidderFilterModeInclude,
			expected:    false,
		},
		{
			description: "Exclude - None",
			bidders:     []string{},
			mode:        BidderFilterModeExclude,
			expected:    true,
		},
		{
			description: "Exclude - One",
			bidders:     []string{bidder},
			mode:        BidderFilterModeExclude,
			expected:    false,
		},
		{
			description: "Exclude - Many",
			bidders:     []string{"other", bidder},
			mode:        BidderFilterModeExclude,
			expected:    false,
		},
		{
			description: "Exclude - Other",
			bidders:     []string{"other"},
			mode:        BidderFilterModeExclude,
			expected:    true,
		},
		{
			description: "Invalid Mode",
			bidders:     []string{bidder},
			mode:        BidderFilterMode(-1),
			expected:    false,
		},
		{
			description: "Case Insensitive Include - One",
			bidders:     []string{strings.ToUpper(bidder)},
			mode:        BidderFilterModeInclude,
			expected:    true,
		},
	}

	for _, test := range testCases {
		filter := NewSpecificBidderFilter(test.bidders, test.mode)
		assert.Equal(t, test.expected, filter.Allowed(bidder), test.description)
	}
}

func TestUniformBidderFilter(t *testing.T) {
	bidder := "a"

	testCases := []struct {
		description string
		mode        BidderFilterMode
		expected    bool
	}{
		{
			description: "Include",
			mode:        BidderFilterModeInclude,
			expected:    true,
		},
		{
			description: "Exclude",
			mode:        BidderFilterModeExclude,
			expected:    false,
		},
	}

	for _, test := range testCases {
		filter := NewUniformBidderFilter(test.mode)
		assert.Equal(t, test.expected, filter.Allowed(bidder), test.description)
	}
}
