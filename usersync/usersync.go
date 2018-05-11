package usersync

type Usersyncer interface {
	// GetUsersyncInfo returns basic info the browser needs in order to run a user sync.
	// The returned UsersyncInfo object must not be mutated by callers.
	//
	// For more information about user syncs, see http://clearcode.cc/2015/12/cookie-syncing/
	GetUsersyncInfo() *UsersyncInfo
	// FamilyName identifies the space of cookies for this usersyncer.
	// For example, if this Usersyncer syncs with adnxs.com, then this
	// should return "adnxs".
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
