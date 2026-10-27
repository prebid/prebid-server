package openrtb_ext

// ExtImpEzoic defines the contract for bidrequest.imp[i].ext.prebid.bidder.ezoic.
// Ezoic eligibility is resolved server-side from site.domain, so no params are
// required; placementId is an optional onboarding-assigned identifier.
type ExtImpEzoic struct {
	PlacementID string `json:"placementId,omitempty"`
}
