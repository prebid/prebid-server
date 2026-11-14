package openrtb_ext

// ExtImpBiddigi defines the contract for bidrequest.imp[i].ext.prebid.bidder.biddigi.
//
// seatKey is the per-integration credential BidDigi issues when an integration is onboarded. The
// adapter sends it as an Authorization header and removes it from the outgoing bid request, so it
// never appears in the request body.
//
// There is no publisher or site parameter: BidDigi resolves the publisher from the authenticated
// seat together with site.domain / app.bundle, server-side.
type ExtImpBiddigi struct {
	SeatKey     string `json:"seatKey"`
	PlacementID string `json:"placementId,omitempty"`
}
