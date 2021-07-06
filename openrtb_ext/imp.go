package openrtb_ext

import (
	"encoding/json"
)

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	// StoredRequest specifies which stored impression to use, if any.
	StoredRequest *ExtStoredRequest `json:"storedrequest"`

	// IsRewardedInventory is a signal intended for video impressions. Must be 0 or 1.
	IsRewardedInventory int8 `json:"is_rewarded_inventory"`

	SKADN SKADN `json:"skadn"`

	// Bidder is the preferred approach for providing paramters to be interepreted by the bidder's adapter.
	Bidder map[string]json.RawMessage `json:"bidder"`
}

// ExtStoredRequest defines the contract for bidrequest.imp[i].ext.prebid.storedrequest
type ExtStoredRequest struct {
	ID string `json:"id"`
}

// SKADN ..
type SKADN struct {
	Version    string   `json:"version,omitempty"`
	Versions   []string `json:"versions,omitempty"`
	SourceApp  string   `json:"sourceapp,omitempty"`
	SKADNetIDs []string `json:"skadnetids,omitempty"`
}
