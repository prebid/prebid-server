package usersync

import "github.com/prebid/prebid-server/config"

type Chooser interface {
	Choose(request Request, cookie Cookie) Result
}

func NewChooser(bidderSyncerLookup map[string]Syncer) Chooser {
	bidders := make([]string, 0, len(bidderSyncerLookup))
	for k := range bidderSyncerLookup {
		bidders = append(bidders, k)
	}

	return standardChooser{
		bidderSyncerLookup: bidderSyncerLookup,
		biddersAvailable:   bidders,
		bidderChooser:      standardBidderChooser{shuffler: randomShuffler{}},
	}
}

type Request struct {
	Bidders     []string
	Kind        Kind
	Privacy     Privacy
	Cooperative config.UserSyncCooperative
	Limit       int
}

type Result struct {
	Status           Status
	BiddersEvaluated []BidderEvaluation
	SyncersChosen    []Syncer
}

type BidderEvaluation struct {
	Bidder string
	Status Status
}

type Status int

const (
	StatusOK Status = iota
	StatusBlockedByUserOptOut
	StatusBlockedByGDPR
	StatusBlockedByCCPA
	StatusAlreadySynced
	StatusUnknownBidder
	StatusIncompatibleKind
	StatusDuplicate
)

type Privacy interface {
	GDPRAllowsHostCookie() bool
	GDPRAllowsBidderSync(bidder string) bool
	CCPAAllowsBidderSync(bidder string) bool
}

type standardChooser struct {
	bidderSyncerLookup map[string]Syncer
	biddersAvailable   []string
	bidderChooser      bidderChooser
}

func (c standardChooser) Choose(request Request, cookie Cookie) Result {
	if !cookie.AllowSyncs() {
		return Result{Status: StatusBlockedByUserOptOut}
	}

	if !request.Privacy.GDPRAllowsHostCookie() {
		return Result{Status: StatusBlockedByGDPR}
	}

	syncersSeen := make(map[string]struct{})
	limitDisabled := request.Limit <= 0

	biddersEvaluated := make([]BidderEvaluation, 0)
	syncersChosen := make([]Syncer, 0)

	bidders := c.bidderChooser.choose(request.Bidders, c.biddersAvailable, request.Cooperative)
	for i := 0; i < len(bidders) && (limitDisabled || len(syncersChosen) < request.Limit); i++ {
		syncer, evaluation := c.evaluate(bidders[i], syncersSeen, request.Kind, request.Privacy, cookie)

		biddersEvaluated = append(biddersEvaluated, evaluation)
		if evaluation.Status == StatusOK {
			syncersChosen = append(syncersChosen, syncer)
		}
	}

	return Result{Status: StatusOK, BiddersEvaluated: biddersEvaluated, SyncersChosen: syncersChosen}
}

func (c standardChooser) evaluate(bidder string, syncersSeen map[string]struct{}, kind Kind, privacy Privacy, cookie Cookie) (Syncer, BidderEvaluation) {
	syncer, exists := c.bidderSyncerLookup[bidder]
	if !exists {
		return nil, BidderEvaluation{Bidder: bidder, Status: StatusUnknownBidder}
	}

	_, seen := syncersSeen[syncer.Key()]
	if seen {
		return nil, BidderEvaluation{Bidder: bidder, Status: StatusDuplicate}
	}
	syncersSeen[syncer.Key()] = struct{}{}

	if !syncer.SupportsKind(kind) {
		return nil, BidderEvaluation{Bidder: bidder, Status: StatusIncompatibleKind}
	}

	if cookie.HasLiveSync(syncer.Key()) {
		return nil, BidderEvaluation{Bidder: bidder, Status: StatusAlreadySynced}
	}

	if !privacy.GDPRAllowsBidderSync(bidder) {
		return nil, BidderEvaluation{Bidder: bidder, Status: StatusBlockedByGDPR}
	}

	if !privacy.CCPAAllowsBidderSync(bidder) {
		return nil, BidderEvaluation{Bidder: bidder, Status: StatusBlockedByCCPA}
	}

	return syncer, BidderEvaluation{Bidder: bidder, Status: StatusOK}
}
