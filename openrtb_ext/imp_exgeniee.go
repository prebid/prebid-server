package openrtb_ext

// ExtImpExGeniee defines the contract for bidrequest.imp[i].ext.prebid.bidder.ex_geniee.
type ExtImpExGeniee struct {
	// PartnerID identifies the supply partner integration.
	PartnerID int `json:"partnerId"`
	// PlacementID names the ad slot.
	PlacementID string `json:"placementId"`
}
