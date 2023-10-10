package usersync

import "github.com/prebid/prebid-server/openrtb_ext"

// Chooser determines which syncers are eligible for a given request.
type Chooser interface {
	// Choose considers bidders to sync, filters the bidders, and returns the result of the
	// user sync selection.
	Choose(request Request, cookie *Cookie) Result
}

// NewChooser returns a new instance of the standard chooser implementation.
func NewChooser(bidderSyncerLookup map[string]Syncer) Chooser {
	bidders := make([]string, 0, len(bidderSyncerLookup))
	for k := range bidderSyncerLookup {
		bidders = append(bidders, k)
	}

	return standardChooser{
		bidderSyncerLookup:       bidderSyncerLookup,
		biddersAvailable:         bidders,
		bidderChooser:            standardBidderChooser{shuffler: randomShuffler{}},
		normalizeValidBidderName: openrtb_ext.NormalizeBidderName,
	}
}

// Request specifies a user sync request.
type Request struct {
	Bidders        []string
	Cooperative    Cooperative
	Limit          int
	Privacy        Privacy
	SyncTypeFilter SyncTypeFilter
}

// Cooperative specifies the settings for cooperative syncing for a given request, where bidders
// other than those used by the publisher are considered for syncing.
type Cooperative struct {
	Enabled        bool
	PriorityGroups [][]string
}

// Result specifies which bidders were included in the evaluation and which syncers were chosen.
type Result struct {
	BiddersEvaluated []BidderEvaluation
	Status           Status
	SyncersChosen    []SyncerChoice
}

// BidderEvaluation specifies which bidders were considered to be synced.
type BidderEvaluation struct {
	Bidder    string
	SyncerKey string
	Status    Status
}

// SyncerChoice specifies a syncer chosen.
type SyncerChoice struct {
	Bidder string
	Syncer Syncer
}

// Status specifies the result of a sync evaluation.
type Status int

const (
	// StatusOK specifies user syncing is permitted.
	StatusOK Status = iota

	// StatusBlockedByUserOptOut specifies a user's cookie explicitly signals an opt-out.
	StatusBlockedByUserOptOut

	// StatusBlockedByGDPR specifies a user's GDPR TCF consent explicitly forbids host cookies
	// or specific bidder syncing.
	StatusBlockedByGDPR

	// StatusBlockedByCCPA specifies a user's CCPA consent explicitly forbids bidder syncing.
	StatusBlockedByCCPA

	// StatusAlreadySynced specifies a user's cookie has an existing non-expired sync for a specific bidder.
	StatusAlreadySynced

	// StatusUnknownBidder specifies a requested bidder is unknown to Prebid Server.
	StatusUnknownBidder

	// StatusTypeNotSupported specifies a requested sync type is not supported by a specific bidder.
	StatusTypeNotSupported

	// StatusDuplicate specifies the bidder is a duplicate or shared a syncer key with another bidder choice.
	StatusDuplicate

	// StatusBlockedByPrivacy specifies a bidder sync url is not allowed by privacy activities
	StatusBlockedByPrivacy
)

// Privacy determines which privacy policies will be enforced for a user sync request.
type Privacy interface {
	GDPRAllowsHostCookie() bool
	GDPRAllowsBidderSync(bidder string) bool
	CCPAAllowsBidderSync(bidder string) bool
	ActivityAllowsUserSync(bidder string) bool
}

// standardChooser implements the user syncer algorithm per official Prebid specification.
type standardChooser struct {
	bidderSyncerLookup       map[string]Syncer
	biddersAvailable         []string
	bidderChooser            bidderChooser
	normalizeValidBidderName func(name string) (openrtb_ext.BidderName, bool)
}

// Choose randomly selects user syncers which are permitted by the user's privacy settings and
// which don't already have a valid user sync.
func (c standardChooser) Choose(request Request, cookie *Cookie) Result {
	if !cookie.AllowSyncs() {
		return Result{Status: StatusBlockedByUserOptOut}
	}

	if !request.Privacy.GDPRAllowsHostCookie() {
		return Result{Status: StatusBlockedByGDPR}
	}

	syncersSeen := make(map[string]struct{})
	limitDisabled := request.Limit <= 0

	biddersEvaluated := make([]BidderEvaluation, 0)
	syncersChosen := make([]SyncerChoice, 0)

	bidders := c.bidderChooser.choose(request.Bidders, c.biddersAvailable, request.Cooperative)
	for i := 0; i < len(bidders) && (limitDisabled || len(syncersChosen) < request.Limit); i++ {
		syncer, evaluation := c.evaluate(bidders[i], syncersSeen, request.SyncTypeFilter, request.Privacy, cookie)

		biddersEvaluated = append(biddersEvaluated, evaluation)
		if evaluation.Status == StatusOK {
			syncersChosen = append(syncersChosen, SyncerChoice{Bidder: bidders[i], Syncer: syncer})
		}
	}

	return Result{Status: StatusOK, BiddersEvaluated: biddersEvaluated, SyncersChosen: syncersChosen}
}

func (c standardChooser) evaluate(bidder string, syncersSeen map[string]struct{}, syncTypeFilter SyncTypeFilter, privacy Privacy, cookie *Cookie) (Syncer, BidderEvaluation) {
	bidderNormalized, exists := c.normalizeValidBidderName(bidder)
	if !exists {
		return nil, BidderEvaluation{Status: StatusUnknownBidder, Bidder: bidder}
	}

	syncer, exists := c.bidderSyncerLookup[bidderNormalized.String()]
	if !exists {
		return nil, BidderEvaluation{Status: StatusUnknownBidder, Bidder: bidder}
	}

	_, seen := syncersSeen[syncer.Key()]
	if seen {
		return nil, BidderEvaluation{Status: StatusDuplicate, Bidder: bidder, SyncerKey: syncer.Key()}
	}
	syncersSeen[syncer.Key()] = struct{}{}

	if !syncer.SupportsType(syncTypeFilter.ForBidder(bidder)) {
		return nil, BidderEvaluation{Status: StatusTypeNotSupported, Bidder: bidder, SyncerKey: syncer.Key()}
	}

	if cookie.HasLiveSync(syncer.Key()) {
		return nil, BidderEvaluation{Status: StatusAlreadySynced, Bidder: bidder, SyncerKey: syncer.Key()}
	}

	userSyncActivityAllowed := privacy.ActivityAllowsUserSync(bidder)
	if !userSyncActivityAllowed {
		return nil, BidderEvaluation{Status: StatusBlockedByPrivacy, Bidder: bidder, SyncerKey: syncer.Key()}
	}

	if !privacy.GDPRAllowsBidderSync(bidder) {
		return nil, BidderEvaluation{Status: StatusBlockedByGDPR, Bidder: bidder, SyncerKey: syncer.Key()}
	}

	if !privacy.CCPAAllowsBidderSync(bidder) {
		return nil, BidderEvaluation{Status: StatusBlockedByCCPA, Bidder: bidder, SyncerKey: syncer.Key()}
	}

	return syncer, BidderEvaluation{Status: StatusOK, Bidder: bidder, SyncerKey: syncer.Key()}
}
