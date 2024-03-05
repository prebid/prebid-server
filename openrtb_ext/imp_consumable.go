package openrtb_ext

// ExtImpConsumable defines the contract for bidrequest.imp[i].ext.prebid.bidder.consumable
type ExtImpConsumable struct {
	NetworkId int `json:"networkId,omitempty"`
	SiteId    int `json:"siteId,omitempty"`
	UnitId    int `json:"unitId,omitempty"`
	/* UnitName gets used as a classname and in the URL when building the ad markup */
	UnitName    string `json:"unitName,omitempty"`
	PlacementId string `json:"placementid,omitempty"`
}
