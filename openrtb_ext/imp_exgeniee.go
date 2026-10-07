package openrtb_ext

// ExtImpExGeniee defines the contract for bidrequest.imp[i].ext.prebid.bidder.ex_geniee.
type ExtImpExGeniee struct {
	// PartnerID identifies the supply partner integration.
	PartnerID int `json:"partnerId"`
	// PlacementID names the ad slot. Geniee SSP validates the value itself, so
	// the adapter and its JSON schema do not restrict it.
	PlacementID string `json:"placementId"`
}
