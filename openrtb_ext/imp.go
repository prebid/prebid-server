package openrtb_ext

import (
	"encoding/json"
)

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	// StoredRequest specifies which stored impression to use, if any.
	StoredRequest *ExtStoredRequest `json:"storedrequest"`

	// StoredResponse specifies which stored impression to use, if any.
	StoredAuctionResponse *ExtStoredAuctionResponse `json:"storedauctionresponse,omitempty"`

	// Stored bid response determines if imp has stored bid response for bidder
	StoredBidResponse []ExtStoredBidResponse `json:"storedbidresponse,omitempty"`

	// IsRewardedInventory is a signal intended for video impressions. Must be 0 or 1.
	IsRewardedInventory int8 `json:"is_rewarded_inventory"`

	// Bidder is the preferred approach for providing paramters to be interepreted by the bidder's adapter.
	Bidder map[string]json.RawMessage `json:"bidder"`

	Options *Options `json:"options,omitempty"`
}

// ExtStoredRequest defines the contract for bidrequest.imp[i].ext.prebid.storedrequest
type ExtStoredRequest struct {
	ID string `json:"id"`
}

// ExtStoredAuctionResponse defines the contract for bidrequest.imp[i].ext.prebid.storedauctionresponse
type ExtStoredAuctionResponse struct {
	ID string `json:"id"`
}

// ExtStoredBidResponse defines the contract for bidrequest.imp[i].ext.prebid.storedbidresponse
type ExtStoredBidResponse struct {
	ID     string `json:"id"`
	Bidder string `json:"bidder"`
}

type Options struct {
	EchoVideoAttrs bool `json:"echovideoattrs"`
}
