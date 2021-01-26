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

	// GDPRVendorID returns the ID in the IAB Global Vendor List which refers to this Bidder.
	//
	// The Global Vendor list can be found here: https://vendor-list.consensu.org/vendorlist.json
	// Bidders can register for the list here: https://register.consensu.org/
	//
	// If you're not on the list, this should return 0. If cookie sync requests have GDPR consent info,
	// or the Prebid Server host company configures its deploy to be "cautious" when no GDPR info exists
	// in the request, it will _not_ sync user IDs with you.
	GDPRVendorID() uint16
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
