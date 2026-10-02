package openrtb_ext

// ExtImpSmartadserver defines the contract for bidrequest.imp[i].ext.prebid.bidder.smartadserver.
// ExtImpSmartadserverIn carries the publisher-facing param names; ExtImpSmartadserverOut carries the
// wire names sent to Equativ.
type ExtImpSmartadserverIn struct {
	SiteID                 int    `json:"siteId"`
	PageID                 int    `json:"pageId"`
	FormatID               int    `json:"formatId"`
	NetworkID              int    `json:"networkId"`
	PlacementUUID          string `json:"placementuuid"`
	ProgrammaticGuaranteed bool   `json:"programmaticGuaranteed"`
}

// ExtImpSmartadserverOut is the imp.ext.<bidder> payload forwarded to Equativ (wire names).
type ExtImpSmartadserverOut struct {
	SiteID                 int    `json:"siteId"`
	PageID                 int    `json:"pageId"`
	FormatID               int    `json:"formatId"`
	NetworkID              int    `json:"networkId"`
	PlacementUUID          string `json:"plcmtuuid,omitempty"` // Equativ's wire name for placementuuid.
	ProgrammaticGuaranteed bool   `json:"-"`
}
