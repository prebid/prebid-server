package openrtb_ext

// ExtImpAdswag defines the contract for bidrequest.imp[i].ext.prebid.bidder.adswag.
// Params are deliberately minimal (publisherId + optional placementId): floors,
// video parameters, schain, first-party data and consent all arrive in the
// OpenRTB request itself, per PBS param hygiene.
type ExtImpAdswag struct {
	PublisherID string `json:"publisherId"`
	PlacementID string `json:"placementId,omitempty"`
}
