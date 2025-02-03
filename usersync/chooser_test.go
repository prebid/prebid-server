package usersync

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/prebid/prebid-server/v3/macros"
)

func TestNewChooser(t *testing.T) {
	testCases := []struct {
		description              string
		bidderSyncerLookup       map[string]Syncer
		bidderInfo               map[string]config.BidderInfo
		expectedBiddersAvailable []string
	}{
		{
			description:              "Nil",
			bidderSyncerLookup:       nil,
			expectedBiddersAvailable: []string{},
		},
		{
			description:              "Empty",
			bidderSyncerLookup:       map[string]Syncer{},
			expectedBiddersAvailable: []string{},
		},
		{
			description:              "One",
			bidderSyncerLookup:       map[string]Syncer{"a": fakeSyncer{}},
			expectedBiddersAvailable: []string{"a"},
		},
		{
			description:              "Many",
			bidderSyncerLookup:       map[string]Syncer{"a": fakeSyncer{}, "b": fakeSyncer{}},
			expectedBiddersAvailable: []string{"a", "b"},
		},
	}

	for _, test := range testCases {
		chooser, _ := NewChooser(test.bidderSyncerLookup, make(map[string]struct{}), test.bidderInfo).(standardChooser)
		assert.ElementsMatch(t, test.expectedBiddersAvailable, chooser.biddersAvailable, test.description)
	}
}

func TestChooserChoose(t *testing.T) {
	fakeSyncerA := fakeSyncer{key: "keyA", supportsIFrame: true}
	fakeSyncerB := fakeSyncer{key: "keyB", supportsIFrame: true}
	fakeSyncerC := fakeSyncer{key: "keyC", supportsIFrame: false}

	duplicateSyncer := fakeSyncer{key: "syncerForDuplicateTest", supportsIFrame: true}
	bidderSyncerLookup := map[string]Syncer{"a": fakeSyncerA, "b": fakeSyncerB, "c": fakeSyncerC, "appnexus": fakeSyncerA, "d": duplicateSyncer, "e": duplicateSyncer}
	biddersKnown := map[string]struct{}{"a": {}, "b": {}, "c": {}}

	normalizedBidderNamesLookup := func(name string) (openrtb_ext.BidderName, bool) {
		return openrtb_ext.BidderName(name), true
	}

	syncerChoiceA := SyncerChoice{Bidder: "a", Syncer: fakeSyncerA}
	syncerChoiceB := SyncerChoice{Bidder: "b", Syncer: fakeSyncerB}

	syncTypeFilter := SyncTypeFilter{
		IFrame:   NewUniformBidderFilter(BidderFilterModeInclude),
		Redirect: NewUniformBidderFilter(BidderFilterModeExclude),
	}

	cooperativeConfig := Cooperative{Enabled: true}

	usersyncDisabled := ptrutil.ToPtr(false)

	testCases := []struct {
		description        string
		givenRequest       Request
		givenChosenBidders []string
		givenCookie        Cookie
		givenBidderInfo    map[string]config.BidderInfo
		bidderNamesLookup  func(name string) (openrtb_ext.BidderName, bool)
		expected           Result
	}{
		{
			description: "Cookie Opt Out",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{optOut: true},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusBlockedByUserOptOut,
				BiddersEvaluated: nil,
				SyncersChosen:    nil,
			},
		},
		{
			description: "GDPR Host Cookie Not Allowed",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: false, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusBlockedByPrivacy,
				BiddersEvaluated: nil,
				SyncersChosen:    nil,
			},
		},
		{
			description: "No Bidders",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "One Bidder - Sync",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "One Bidder - No Sync",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"c"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "c", SyncerKey: "keyC", Status: StatusRejectedByFilter}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "One Bidder - No Sync - Unknown",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"unknown"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "unknown", Status: StatusUnknownBidder}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "Many Bidders - All Sync - Limit Disabled With 0",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a", "b"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "b", SyncerKey: "keyB", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA, syncerChoiceB},
			},
		},
		{
			description: "Many Bidders - All Sync - Limit Disabled With Negative Value",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   -1,
			},
			givenChosenBidders: []string{"a", "b"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "b", SyncerKey: "keyB", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA, syncerChoiceB},
			},
		},
		{
			description: "Many Bidders - Limited Sync",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   1,
			},
			givenChosenBidders: []string{"a", "b"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "Many Bidders - Limited Sync - Disqualified Syncers Don't Count Towards Limit",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   1,
			},
			givenChosenBidders: []string{"c", "a", "b"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "c", SyncerKey: "keyC", Status: StatusRejectedByFilter}, {Bidder: "a", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "Many Bidders - Some Sync, Some Don't",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a", "c"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "c", SyncerKey: "keyC", Status: StatusRejectedByFilter}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "Chosen bidders have duplicate syncer keys, the one that comes first should be labled OK",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"d", "e"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status: StatusOK,
				BiddersEvaluated: []BidderEvaluation{
					{Bidder: "d", SyncerKey: "syncerForDuplicateTest", Status: StatusOK},
					{Bidder: "e", SyncerKey: "syncerForDuplicateTest", Status: StatusDuplicate},
				},
				SyncersChosen: []SyncerChoice{{Bidder: "d", Syncer: duplicateSyncer}},
			},
		},
		{
			description: "Chosen bidders have duplicate syncer keys, the one that comes first should be labled OK (reverse)",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"e", "d"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status: StatusOK,
				BiddersEvaluated: []BidderEvaluation{
					{Bidder: "e", SyncerKey: "syncerForDuplicateTest", Status: StatusOK},
					{Bidder: "d", SyncerKey: "syncerForDuplicateTest", Status: StatusDuplicate},
				},
				SyncersChosen: []SyncerChoice{{Bidder: "e", Syncer: duplicateSyncer}},
			},
		},
		{
			description: "Same bidder name, no duplicate warning",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a", "a"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  normalizedBidderNamesLookup,
			expected: Result{
				Status: StatusOK,
				BiddersEvaluated: []BidderEvaluation{
					{Bidder: "a", SyncerKey: fakeSyncerA.key, Status: StatusOK},
				},
				SyncersChosen: []SyncerChoice{{Bidder: "a", Syncer: fakeSyncerA}},
			},
		},
		{
			description: "Unknown Bidder",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			bidderNamesLookup: func(name string) (openrtb_ext.BidderName, bool) {
				return openrtb_ext.BidderName(name), false
			},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", Status: StatusUnknownBidder}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "Case insensitive bidder name",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"AppNexus"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  openrtb_ext.NormalizeBidderName,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "AppNexus", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{{Bidder: "AppNexus", Syncer: fakeSyncerA}},
			},
		},
		{
			description: "Duplicate bidder name",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"AppNexus", "appNexus"},
			givenCookie:        Cookie{},
			bidderNamesLookup:  openrtb_ext.NormalizeBidderName,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "AppNexus", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "appNexus", SyncerKey: "keyA", Status: StatusDuplicate}},
				SyncersChosen:    []SyncerChoice{{Bidder: "AppNexus", Syncer: fakeSyncerA}},
			},
		},
		{
			description: "Disabled Usersync",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			givenBidderInfo: map[string]config.BidderInfo{
				"a": {
					Syncer: &config.Syncer{
						Enabled: usersyncDisabled,
					},
				},
			},
			bidderNamesLookup: normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByDisabledUsersync}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "Regulation Scope GDPR",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true, gdprInScope: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			givenBidderInfo: map[string]config.BidderInfo{
				"a": {
					Syncer: &config.Syncer{
						SkipWhen: &config.SkipWhen{
							GDPR: true,
						},
					},
				},
			},
			bidderNamesLookup: normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByRegulationScope}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "Regulation Scope GPP",
			givenRequest: Request{
				Privacy: &fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
				Limit:   0,
				GPPSID:  "2",
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			givenBidderInfo: map[string]config.BidderInfo{
				"a": {
					Syncer: &config.Syncer{
						SkipWhen: &config.SkipWhen{
							GPPSID: []string{"2", "3"},
						},
					},
				},
			},
			bidderNamesLookup: normalizedBidderNamesLookup,
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByRegulationScope}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
	}

	bidders := []string{"anyRequested"}
	biddersAvailable := []string{"anyAvailable"}
	for _, test := range testCases {
		// set request values which don't need to be specified for each test case
		test.givenRequest.Bidders = bidders
		test.givenRequest.SyncTypeFilter = syncTypeFilter
		test.givenRequest.Cooperative = cooperativeConfig

		mockBidderChooser := &mockBidderChooser{}
		mockBidderChooser.
			On("choose", test.givenRequest.Bidders, biddersAvailable, cooperativeConfig).
			Return(test.givenChosenBidders)

		if test.givenBidderInfo == nil {
			test.givenBidderInfo = map[string]config.BidderInfo{}
		}

		chooser := standardChooser{
			bidderSyncerLookup:       bidderSyncerLookup,
			biddersAvailable:         biddersAvailable,
			bidderChooser:            mockBidderChooser,
			normalizeValidBidderName: test.bidderNamesLookup,
			biddersKnown:             biddersKnown,
			bidderInfo:               test.givenBidderInfo,
		}

		result := chooser.Choose(test.givenRequest, &test.givenCookie)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestChooserEvaluate(t *testing.T) {
	fakeSyncerA := fakeSyncer{key: "keyA", supportsIFrame: true}
	fakeSyncerB := fakeSyncer{key: "keyB", supportsIFrame: false}

	biddersKnown := map[string]struct{}{"a": {}, "b": {}, "unconfigured": {}}
	bidderSyncerLookup := map[string]Syncer{"a": fakeSyncerA, "b": fakeSyncerB, "appnexus": fakeSyncerA, "seedingAlliance": fakeSyncerA}

	syncTypeFilter := SyncTypeFilter{
		IFrame:   NewUniformBidderFilter(BidderFilterModeInclude),
		Redirect: NewUniformBidderFilter(BidderFilterModeExclude),
	}
	normalizedBidderNamesLookup := func(name string) (openrtb_ext.BidderName, bool) {
		return openrtb_ext.BidderName(name), true
	}
	cookieNeedsSync := Cookie{}
	cookieAlreadyHasSyncForA := Cookie{uids: map[string]UIDEntry{"keyA": {Expires: time.Now().Add(time.Duration(24) * time.Hour)}}}
	cookieAlreadyHasSyncForB := Cookie{uids: map[string]UIDEntry{"keyB": {Expires: time.Now().Add(time.Duration(24) * time.Hour)}}}

	usersyncDisabled := ptrutil.ToPtr(false)

	testCases := []struct {
		description                 string
		givenBidder                 string
		normalisedBidderName        string
		givenSyncersSeen            map[string]struct{}
		givenPrivacy                fakePrivacy
		givenCookie                 Cookie
		givenGPPSID                 string
		givenBidderInfo             map[string]config.BidderInfo
		givenSyncTypeFilter         SyncTypeFilter
		normalizedBidderNamesLookup func(name string) (openrtb_ext.BidderName, bool)
		expectedSyncer              Syncer
		expectedEvaluation          BidderEvaluation
	}{
		{
			description:                 "Valid",
			givenBidder:                 "a",
			normalisedBidderName:        "a",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              fakeSyncerA,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusOK},
		},
		{
			description:                 "Unknown Bidder",
			givenBidder:                 "unknown",
			normalisedBidderName:        "",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "unknown", Status: StatusUnknownBidder},
		},
		{
			description:                 "Duplicate Syncer",
			givenBidder:                 "a",
			normalisedBidderName:        "",
			givenSyncersSeen:            map[string]struct{}{"keyA": {}},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusDuplicate},
		},
		{
			description:                 "Incompatible Kind",
			givenBidder:                 "b",
			normalisedBidderName:        "",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "b", SyncerKey: "keyB", Status: StatusRejectedByFilter},
		},
		{
			description:                 "Already Synced",
			givenBidder:                 "a",
			normalisedBidderName:        "",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieAlreadyHasSyncForA,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusAlreadySynced},
		},
		{
			description:                 "Different Bidder Already Synced",
			givenBidder:                 "a",
			normalisedBidderName:        "a",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieAlreadyHasSyncForB,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              fakeSyncerA,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusOK},
		},
		{
			description:                 "Blocked By GDPR",
			givenBidder:                 "a",
			normalisedBidderName:        "a",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: false, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByPrivacy},
		},
		{
			description:                 "Blocked By activity control",
			givenBidder:                 "a",
			normalisedBidderName:        "",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: false},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByPrivacy},
		},
		{
			description:                 "Case insensitive bidder name",
			givenBidder:                 "AppNexus",
			normalisedBidderName:        "appnexus",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: openrtb_ext.NormalizeBidderName,
			expectedSyncer:              fakeSyncerA,
			expectedEvaluation:          BidderEvaluation{Bidder: "AppNexus", SyncerKey: "keyA", Status: StatusOK},
		},
		{
			description:          "Case insensitivity check for sync type filter",
			givenBidder:          "SeedingAlliance",
			normalisedBidderName: "seedingAlliance",
			givenSyncersSeen:     map[string]struct{}{},
			givenPrivacy:         fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:          cookieNeedsSync,
			givenSyncTypeFilter: SyncTypeFilter{
				IFrame:   NewSpecificBidderFilter([]string{"SeedingAlliance"}, BidderFilterModeInclude),
				Redirect: NewSpecificBidderFilter([]string{"SeedingAlliance"}, BidderFilterModeExclude),
			},
			normalizedBidderNamesLookup: openrtb_ext.NormalizeBidderName,
			expectedSyncer:              fakeSyncerA,
			expectedEvaluation:          BidderEvaluation{Bidder: "SeedingAlliance", SyncerKey: "keyA", Status: StatusOK},
		},
		{
			description:                 "Case Insensitivity Check For Blocked By GDPR",
			givenBidder:                 "AppNexus",
			normalisedBidderName:        "appnexus",
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: false, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: openrtb_ext.NormalizeBidderName,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "AppNexus", SyncerKey: "keyA", Status: StatusBlockedByPrivacy},
		},
		{
			description:                 "Unconfigured Bidder",
			givenBidder:                 "unconfigured",
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			givenSyncersSeen:            map[string]struct{}{},
			givenPrivacy:                fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:                 cookieNeedsSync,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "unconfigured", Status: StatusUnconfiguredBidder},
		},
		{
			description:          "Disabled Usersync",
			givenBidder:          "a",
			normalisedBidderName: "a",
			givenSyncersSeen:     map[string]struct{}{},
			givenPrivacy:         fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:          cookieNeedsSync,
			givenBidderInfo: map[string]config.BidderInfo{
				"a": {
					Syncer: &config.Syncer{
						Enabled: usersyncDisabled,
					},
				},
			},
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByDisabledUsersync},
		},
		{
			description:      "Blocked By Regulation Scope - GDPR",
			givenBidder:      "a",
			givenSyncersSeen: map[string]struct{}{},
			givenPrivacy:     fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true, gdprInScope: true},
			givenCookie:      cookieNeedsSync,
			givenBidderInfo: map[string]config.BidderInfo{
				"a": {
					Syncer: &config.Syncer{
						SkipWhen: &config.SkipWhen{
							GDPR: true,
						},
					},
				},
			},
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			normalisedBidderName:        "a",
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByRegulationScope},
		},
		{
			description:      "Blocked By Regulation Scope - GPP",
			givenBidder:      "a",
			givenSyncersSeen: map[string]struct{}{},
			givenPrivacy:     fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true, activityAllowUserSync: true},
			givenCookie:      cookieNeedsSync,
			givenBidderInfo: map[string]config.BidderInfo{
				"a": {
					Syncer: &config.Syncer{
						SkipWhen: &config.SkipWhen{
							GPPSID: []string{"2", "3"},
						},
					},
				},
			},
			givenGPPSID:                 "2",
			givenSyncTypeFilter:         syncTypeFilter,
			normalizedBidderNamesLookup: normalizedBidderNamesLookup,
			normalisedBidderName:        "a",
			expectedSyncer:              nil,
			expectedEvaluation:          BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByRegulationScope},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			chooser, _ := NewChooser(bidderSyncerLookup, biddersKnown, test.givenBidderInfo).(standardChooser)
			chooser.normalizeValidBidderName = test.normalizedBidderNamesLookup
			sync, evaluation := chooser.evaluate(test.givenBidder, test.givenSyncersSeen, test.givenSyncTypeFilter, &test.givenPrivacy, &test.givenCookie, test.givenGPPSID)

			assert.Equal(t, test.normalisedBidderName, test.givenPrivacy.inputBidderName)
			assert.Equal(t, test.expectedSyncer, sync, test.description+":syncer")
			assert.Equal(t, test.expectedEvaluation, evaluation, test.description+":evaluation")
		})
	}
}

type mockBidderChooser struct {
	mock.Mock
}

func (m *mockBidderChooser) choose(requested, available []string, cooperative Cooperative) []string {
	args := m.Called(requested, available, cooperative)
	return args.Get(0).([]string)
}

type fakeSyncer struct {
	key              string
	supportsIFrame   bool
	supportsRedirect bool
	formatOverride   string
}

func (s fakeSyncer) Key() string {
	return s.key
}

func (s fakeSyncer) DefaultResponseFormat() SyncType {
	return SyncTypeIFrame
}

func (s fakeSyncer) SupportsType(syncTypes []SyncType) bool {
	for _, syncType := range syncTypes {
		if syncType == SyncTypeIFrame && s.supportsIFrame {
			return true
		}
		if syncType == SyncTypeRedirect && s.supportsRedirect {
			return true
		}
	}
	return false
}

func (fakeSyncer) GetSync([]SyncType, macros.UserSyncPrivacy) (Sync, error) {
	return Sync{}, nil
}

type fakePrivacy struct {
	gdprAllowsHostCookie  bool
	gdprAllowsBidderSync  bool
	ccpaAllowsBidderSync  bool
	activityAllowUserSync bool
	gdprInScope           bool
	inputBidderName       string
}

func (p *fakePrivacy) GDPRAllowsHostCookie() bool {
	return p.gdprAllowsHostCookie
}

func (p *fakePrivacy) GDPRAllowsBidderSync(bidder string) bool {
	p.inputBidderName = bidder
	return p.gdprAllowsBidderSync
}

func (p *fakePrivacy) CCPAAllowsBidderSync(bidder string) bool {
	p.inputBidderName = bidder
	return p.ccpaAllowsBidderSync
}

func (p *fakePrivacy) ActivityAllowsUserSync(bidder string) bool {
	return p.activityAllowUserSync
}

func (p *fakePrivacy) GDPRInScope() bool {
	return p.gdprInScope
}
