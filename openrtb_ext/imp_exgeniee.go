package openrtb_ext

// ExtImpExGeniee defines the contract for bidrequest.imp[i].ext.prebid.bidder.ex_geniee.
type ExtImpExGeniee struct {
	// PartnerID identifies the supply partner integration.
	PartnerID int `json:"partnerId"`
	// PlacementID names the ad slot. Geniee Exchange validates the value itself, so
	// the JSON schema only refuses an empty string.
	PlacementID string `json:"placementId"`
}
