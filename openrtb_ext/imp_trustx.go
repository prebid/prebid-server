package openrtb_ext

import "encoding/json"

// ExtImpTrustX defines the contract for bidrequest.imp[i].ext.prebid.bidder.trustx
type ExtImpTrustX struct {
	Uid      int             `json:"uid"`
	Keywords json.RawMessage `json:"keywords,omitempty"`
}
