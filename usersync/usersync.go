package usersync

import "github.com/prebid/prebid-server/privacy"

type Usersyncer interface {
	// GetUsersyncInfo returns basic info the browser needs in order to run a user sync.
	// The returned UsersyncInfo object must not be mutated by callers.
	//
	// For more information about user syncs, see http://clearcode.cc/2015/12/cookie-syncing/
	GetUsersyncInfo(privacyPolicies privacy.Policies) (*UsersyncInfo, error)

	// FamilyName should be the same as the `BidderName` for this Usersyncer.
	// This function only exists for legacy reasons.
	// TODO #362: when the appnexus usersyncer is consistent, delete this and use the key
	// of NewSyncerMap() here instead.
	FamilyName() string
}

type UsersyncInfo struct {
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SupportCORS bool   `json:"supportCORS,omitempty"`
}

type CookieSyncBidders struct {
	BidderCode   string        `json:"bidder"`
	NoCookie     bool          `json:"no_cookie,omitempty"`
	UsersyncInfo *UsersyncInfo `json:"usersync,omitempty"`
}
