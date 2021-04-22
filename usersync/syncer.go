package usersync

import "github.com/prebid/prebid-server/privacy"

type Syncer interface {
	Key() string
	SupportsKind(kind Kind) bool
	GetSync(kind Kind, privacyPolicies privacy.Policies) Sync
}

type Sync struct {
	URL         string
	Kind        Kind
	SupportCORS bool
}

type Kind int

const (
	KindBidderPreference Kind = iota
	KindIFrame
	KindRedirect
)

// todo: syncer from config
// - builds up the url template per bidder
