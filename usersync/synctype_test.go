package usersync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncTypeFilter(t *testing.T) {
	bidder := "foo"

	bidderFilterAllowed := NewUniformBidderFilter(BidderFilterModeInclude)
	bidderFilterNotAllowed := NewUniformBidderFilter(BidderFilterModeExclude)

	testCases := []struct {
		description         string
		givenIFrameFilter   BidderFilter
		givenRedirectFilter BidderFilter
		expectedSyncTypes   []SyncType
	}{
		{
			description:         "None",
			givenIFrameFilter:   bidderFilterNotAllowed,
			givenRedirectFilter: bidderFilterNotAllowed,
			expectedSyncTypes:   []SyncType{},
		},
		{
			description:         "IFrame Only",
			givenIFrameFilter:   bidderFilterAllowed,
			givenRedirectFilter: bidderFilterNotAllowed,
			expectedSyncTypes:   []SyncType{SyncTypeIFrame},
		},
		{
			description:         "Redirect Only",
			givenIFrameFilter:   bidderFilterNotAllowed,
			givenRedirectFilter: bidderFilterAllowed,
			expectedSyncTypes:   []SyncType{SyncTypeRedirect},
		},
		{
			description:         "All",
			givenIFrameFilter:   bidderFilterAllowed,
			givenRedirectFilter: bidderFilterAllowed,
			expectedSyncTypes:   []SyncType{SyncTypeIFrame, SyncTypeRedirect},
		},
	}

	for _, test := range testCases {
		syncTypeFilter := SyncTypeFilter{IFrame: test.givenIFrameFilter, Redirect: test.givenRedirectFilter}
		syncTypes := syncTypeFilter.ForBidder(bidder)
		assert.ElementsMatch(t, test.expectedSyncTypes, syncTypes, test.description)
	}
}

func TestSyncTypeParse(t *testing.T) {
	testCases := []struct {
		description   string
		given         string
		expected      SyncType
		expectedError string
	}{
		{
			description: "IFrame",
			given:       "iframe",
			expected:    SyncTypeIFrame,
		},
		{
			description: "IFrame - Case Insensitive",
			given:       "iFrAmE",
			expected:    SyncTypeIFrame,
		},
		{
			description: "Redirect",
			given:       "redirect",
			expected:    SyncTypeRedirect,
		},
		{
			description: "Redirect - Case Insensitive",
			given:       "ReDiReCt",
			expected:    SyncTypeRedirect,
		},
		{
			description:   "Invalid",
			given:         "invalid",
			expectedError: "invalid sync type `invalid`",
		},
	}

	for _, test := range testCases {
		result, err := SyncTypeParse(test.given)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expected, result, test.description+":result")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}
