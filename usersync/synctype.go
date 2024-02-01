package usersync

// SyncType specifies the mechanism used to perform a user sync.
type SyncType string

const (
	// SyncTypeUnknown specifies the user sync type is invalid or not specified.
	SyncTypeUnknown SyncType = ""

	// SyncTypeIFrame specifies the user sync is to be performed within an HTML iframe
	// and to expect the server to return a valid HTML page with an embedded script.
	SyncTypeIFrame SyncType = "iframe"

	// SyncTypeRedirect specifies the user sync is to be performed within an HTML image
	// and to expect the server to return a 302 redirect.
	SyncTypeRedirect SyncType = "redirect"
)

// SyncTypeFilter determines which sync types, if any, the bidder is permitted to use.
type SyncTypeFilter struct {
	IFrame   BidderFilter
	Redirect BidderFilter
}

// ForBidder returns a slice of sync types the bidder is permitted to use.
func (t SyncTypeFilter) ForBidder(bidder string) []SyncType {
	var syncTypes []SyncType

	if t.IFrame.Allowed(bidder) {
		syncTypes = append(syncTypes, SyncTypeIFrame)
	}

	if t.Redirect.Allowed(bidder) {
		syncTypes = append(syncTypes, SyncTypeRedirect)
	}

	return syncTypes
}
