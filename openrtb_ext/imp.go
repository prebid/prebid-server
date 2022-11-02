package openrtb_ext

import (
	"encoding/json"
)

// AuctionEnvironmentType is a Google Privacy Sandbox flag indicating where the auction may take place
type AuctionEnvironmentType int8

const (
	ServerSideAuction          AuctionEnvironmentType = 0
	OnDeviceIGAuctionFledge    AuctionEnvironmentType = 1
	ServerSideWithIGSimulation AuctionEnvironmentType = 2
)

// AuctionEnvironmentKey is the json key under imp[].ext.prebid for ExtImpPrebid.AuctionEnvironment
const AuctionEnvironmentKey = "ae"

// IsRewardedInventoryKey is the json key for ExtImpPrebid.IsRewardedInventory
const IsRewardedInventoryKey = "is_rewarded_inventory"

// OptionsKey is the json key for ExtImpPrebid.Options
const OptionsKey = "options"

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	// StoredRequest specifies which stored impression to use, if any.
	StoredRequest *ExtStoredRequest `json:"storedrequest,omitempty"`

	// StoredResponse specifies which stored impression to use, if any.
	StoredAuctionResponse *ExtStoredAuctionResponse `json:"storedauctionresponse,omitempty"`

	// Stored bid response determines if imp has stored bid response for bidder
	StoredBidResponse []ExtStoredBidResponse `json:"storedbidresponse,omitempty"`

	// IsRewardedInventory is a signal intended for video impressions. Must be 0 or 1.
	IsRewardedInventory *int8 `json:"is_rewarded_inventory,omitempty"`

	// Bidder is the preferred approach for providing parameters to be interpreted by the bidder's adapter.
	Bidder map[string]json.RawMessage `json:"bidder,omitempty"`

	Options *Options `json:"options,omitempty"`

	Passthrough json.RawMessage `json:"passthrough,omitempty"`

	// AuctionEnvironment can be 0 (server-side, default), 1 (on-device interest group auction FLEDGE), 2 (server-side with interest group simulation)
	AuctionEnvironment AuctionEnvironmentType `json:"ae,omitempty"`
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
	ID           string `json:"id"`
	Bidder       string `json:"bidder"`
	ReplaceImpId *bool  `json:"replaceimpid"`
}

type Options struct {
	EchoVideoAttrs bool `json:"echovideoattrs"`
}
