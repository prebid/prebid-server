package openrtb_ext

import (
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// ExtImpPulsePoint defines the json spec for bidrequest.imp[i].ext.prebid.bidder.pulsepoint
// PubId/TagId are mandatory params

type ExtImpPulsePoint struct {
	PubID jsonutil.StringInt `json:"cp"`
	TagID jsonutil.StringInt `json:"ct"`
}
