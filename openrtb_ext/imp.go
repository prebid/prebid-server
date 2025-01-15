package openrtb_ext

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// AuctionEnvironmentType is a Google Privacy Sandbox flag indicating where the auction may take place
type AuctionEnvironmentType int8

const (
	// 0 Standard server-side auction
	ServerSideAuction AuctionEnvironmentType = 0
	// 1 On-device interest group auction (FLEDGE)
	OnDeviceIGAuctionFledge AuctionEnvironmentType = 1
	// 2 Server-side with interest group simulation
	ServerSideWithIGSimulation AuctionEnvironmentType = 2
)

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

	AdUnitCode string `json:"adunitcode,omitempty"`

	Passthrough json.RawMessage `json:"passthrough,omitempty"`

	Floors *ExtImpPrebidFloors `json:"floors,omitempty"`

	// Imp specifies any imp bidder-specific first party data
	Imp map[string]json.RawMessage `json:"imp,omitempty"`
}

type ExtImpDataAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

type ExtImpData struct {
	PbAdslot string              `json:"pbadslot,omitempty"`
	AdServer *ExtImpDataAdServer `json:"adserver,omitempty"`
}

type ExtImpPrebidFloors struct {
	FloorRule      string  `json:"floorrule,omitempty"`
	FloorRuleValue float64 `json:"floorrulevalue,omitempty"`
	FloorValue     float64 `json:"floorvalue,omitempty"`
	FloorMin       float64 `json:"floormin,omitempty"`
	FloorMinCur    string  `json:"floorminCur,omitempty"`
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

// GetImpIDs returns slice of all impression Ids from impList
func GetImpIDs(imps []openrtb2.Imp) []string {
	impIDs := make([]string, len(imps))
	for i := range imps {
		impIDs[i] = imps[i].ID
	}
	return impIDs
}
