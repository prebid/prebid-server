package openrtb_ext

import "encoding/json"

// ExtImpTriplelift defines the contract for bidrequest.imp[i].ext.triplelift
type ExtImpTriplelift struct {
    InvCode                 string                  `json:"inventoryCode"`
}
