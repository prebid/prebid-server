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
