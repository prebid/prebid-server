package openrtb_ext

// ImpExtCwire defines the contract for MakeRequests `request.imp[i].ext.bidder`
type ImpExtCWire struct {
	DomainID    int      `json:"domainId,omitempty"`
	PlacementID int      `json:"placementId,omitempty"`
	PageID      int      `json:"pageId,omitempty"`
	CwCreative  string   `json:"cwcreative,omitempty"`
	CwDebug     bool     `json:"cwdebug,omitempty"`
	CwFeatures  []string `json:"cwfeatures,omitempty"`
}
