package openrtb_ext

import "github.com/prebid/prebid-server/v3/util/jsonutil"

// ExtImpTargetVideo defines the contract for bidrequest.imp[i].ext.prebid.bidder.targetVideo
type ExtImpTargetVideo struct {
	PlacementId jsonutil.StringInt `json:"placementId,omitempty"`
}
