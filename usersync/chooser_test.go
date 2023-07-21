package usersync

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewChooser(t *testing.T) {
	testCases := []struct {
		description              string
		bidderSyncerLookup       map[string]Syncer
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
		chooser, _ := NewChooser(test.bidderSyncerLookup).(standardChooser)
		assert.ElementsMatch(t, test.expectedBiddersAvailable, chooser.biddersAvailable, test.description)
	}
}

func TestChooserChoose(t *testing.T) {
	fakeSyncerA := fakeSyncer{key: "keyA", supportsIFrame: true}
	fakeSyncerB := fakeSyncer{key: "keyB", supportsIFrame: true}
	fakeSyncerC := fakeSyncer{key: "keyC", supportsIFrame: false}
	bidderSyncerLookup := map[string]Syncer{"a": fakeSyncerA, "b": fakeSyncerB, "c": fakeSyncerC}
	syncerChoiceA := SyncerChoice{Bidder: "a", Syncer: fakeSyncerA}
	syncerChoiceB := SyncerChoice{Bidder: "b", Syncer: fakeSyncerB}
	syncTypeFilter := SyncTypeFilter{
		IFrame:   NewUniformBidderFilter(BidderFilterModeInclude),
		Redirect: NewUniformBidderFilter(BidderFilterModeExclude)}

	cooperativeConfig := Cooperative{Enabled: true}

	testCases := []struct {
		description        string
		givenRequest       Request
		givenChosenBidders []string
		givenCookie        Cookie
		expected           Result
	}{
		{
			description: "Cookie Opt Out",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{optOut: true},
			expected: Result{
				Status:           StatusBlockedByUserOptOut,
				BiddersEvaluated: nil,
				SyncersChosen:    nil,
			},
		},
		{
			description: "GDPR Host Cookie Not Allowed",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: false, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusBlockedByGDPR,
				BiddersEvaluated: nil,
				SyncersChosen:    nil,
			},
		},
		{
			description: "No Bidders",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "One Bidder - Sync",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "One Bidder - No Sync",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"c"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "c", SyncerKey: "keyC", Status: StatusTypeNotSupported}},
				SyncersChosen:    []SyncerChoice{},
			},
		},
		{
			description: "Many Bidders - All Sync - Limit Disabled With 0",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a", "b"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "b", SyncerKey: "keyB", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA, syncerChoiceB},
			},
		},
		{
			description: "Many Bidders - All Sync - Limit Disabled With Negative Value",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   -1,
			},
			givenChosenBidders: []string{"a", "b"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "b", SyncerKey: "keyB", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA, syncerChoiceB},
			},
		},
		{
			description: "Many Bidders - Limited Sync",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   1,
			},
			givenChosenBidders: []string{"a", "b"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "Many Bidders - Limited Sync - Disqualified Syncers Don't Count Towards Limit",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   1,
			},
			givenChosenBidders: []string{"c", "a", "b"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "c", SyncerKey: "keyC", Status: StatusTypeNotSupported}, {Bidder: "a", SyncerKey: "keyA", Status: StatusOK}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
			},
		},
		{
			description: "Many Bidders - Some Sync, Some Don't",
			givenRequest: Request{
				Privacy: fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
				Limit:   0,
			},
			givenChosenBidders: []string{"a", "c"},
			givenCookie:        Cookie{},
			expected: Result{
				Status:           StatusOK,
				BiddersEvaluated: []BidderEvaluation{{Bidder: "a", SyncerKey: "keyA", Status: StatusOK}, {Bidder: "c", SyncerKey: "keyC", Status: StatusTypeNotSupported}},
				SyncersChosen:    []SyncerChoice{syncerChoiceA},
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

		chooser := standardChooser{
			bidderSyncerLookup: bidderSyncerLookup,
			biddersAvailable:   biddersAvailable,
			bidderChooser:      mockBidderChooser,
		}

		result := chooser.Choose(test.givenRequest, &test.givenCookie)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestChooserEvaluate(t *testing.T) {
	fakeSyncerA := fakeSyncer{key: "keyA", supportsIFrame: true}
	fakeSyncerB := fakeSyncer{key: "keyB", supportsIFrame: false}
	bidderSyncerLookup := map[string]Syncer{"a": fakeSyncerA, "b": fakeSyncerB}
	syncTypeFilter := SyncTypeFilter{
		IFrame:   NewUniformBidderFilter(BidderFilterModeInclude),
		Redirect: NewUniformBidderFilter(BidderFilterModeExclude)}

	cookieNeedsSync := Cookie{}
	cookieAlreadyHasSyncForA := Cookie{uids: map[string]UIDEntry{"keyA": {Expires: time.Now().Add(time.Duration(24) * time.Hour)}}}
	cookieAlreadyHasSyncForB := Cookie{uids: map[string]UIDEntry{"keyB": {Expires: time.Now().Add(time.Duration(24) * time.Hour)}}}

	testCases := []struct {
		description        string
		givenBidder        string
		givenSyncersSeen   map[string]struct{}
		givenPrivacy       Privacy
		givenCookie        Cookie
		expectedSyncer     Syncer
		expectedEvaluation BidderEvaluation
	}{
		{
			description:        "Valid",
			givenBidder:        "a",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
			givenCookie:        cookieNeedsSync,
			expectedSyncer:     fakeSyncerA,
			expectedEvaluation: BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusOK},
		},
		{
			description:        "Unknown Bidder",
			givenBidder:        "unknown",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
			givenCookie:        cookieNeedsSync,
			expectedSyncer:     nil,
			expectedEvaluation: BidderEvaluation{Bidder: "unknown", Status: StatusUnknownBidder},
		},
		{
			description:        "Duplicate Syncer",
			givenBidder:        "a",
			givenSyncersSeen:   map[string]struct{}{"keyA": {}},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
			givenCookie:        cookieNeedsSync,
			expectedSyncer:     nil,
			expectedEvaluation: BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusDuplicate},
		},
		{
			description:        "Incompatible Kind",
			givenBidder:        "b",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
			givenCookie:        cookieNeedsSync,
			expectedSyncer:     nil,
			expectedEvaluation: BidderEvaluation{Bidder: "b", SyncerKey: "keyB", Status: StatusTypeNotSupported},
		},
		{
			description:        "Already Synced",
			givenBidder:        "a",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
			givenCookie:        cookieAlreadyHasSyncForA,
			expectedSyncer:     nil,
			expectedEvaluation: BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusAlreadySynced},
		},
		{
			description:        "Different Bidder Already Synced",
			givenBidder:        "a",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: true},
			givenCookie:        cookieAlreadyHasSyncForB,
			expectedSyncer:     fakeSyncerA,
			expectedEvaluation: BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusOK},
		},
		{
			description:        "Blocked By GDPR",
			givenBidder:        "a",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: false, ccpaAllowsBidderSync: true},
			givenCookie:        cookieNeedsSync,
			expectedSyncer:     nil,
			expectedEvaluation: BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByGDPR},
		},
		{
			description:        "Blocked By CCPA",
			givenBidder:        "a",
			givenSyncersSeen:   map[string]struct{}{},
			givenPrivacy:       fakePrivacy{gdprAllowsHostCookie: true, gdprAllowsBidderSync: true, ccpaAllowsBidderSync: false},
			givenCookie:        cookieNeedsSync,
			expectedSyncer:     nil,
			expectedEvaluation: BidderEvaluation{Bidder: "a", SyncerKey: "keyA", Status: StatusBlockedByCCPA},
		},
	}

	for _, test := range testCases {
		chooser, _ := NewChooser(bidderSyncerLookup).(standardChooser)
		sync, evaluation := chooser.evaluate(test.givenBidder, test.givenSyncersSeen, syncTypeFilter, test.givenPrivacy, &test.givenCookie)

		assert.Equal(t, test.expectedSyncer, sync, test.description+":syncer")
		assert.Equal(t, test.expectedEvaluation, evaluation, test.description+":evaluation")
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
}

func (s fakeSyncer) Key() string {
	return s.key
}

func (s fakeSyncer) DefaultSyncType() SyncType {
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

func (fakeSyncer) GetSync(syncTypes []SyncType, privacyPolicies privacy.Policies) (Sync, error) {
	return Sync{}, nil
}

type fakePrivacy struct {
	gdprAllowsHostCookie bool
	gdprAllowsBidderSync bool
	ccpaAllowsBidderSync bool
}

func (p fakePrivacy) GDPRAllowsHostCookie() bool {
	return p.gdprAllowsHostCookie
}

func (p fakePrivacy) GDPRAllowsBidderSync(bidder string) bool {
	return p.gdprAllowsBidderSync
}

func (p fakePrivacy) CCPAAllowsBidderSync(bidder string) bool {
	return p.ccpaAllowsBidderSync
}
