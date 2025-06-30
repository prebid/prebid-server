package openrtb_ext

import (
	"encoding/json"
	"fmt"
)

// ExtBid defines the contract for bidresponse.seatbid.bid[i].ext
type ExtBid struct {
	DSA    *ExtBidDSA    `json:"dsa,omitempty"`
	Prebid *ExtBidPrebid `json:"prebid,omitempty"`
}

// ExtBidPrebid defines the contract for bidresponse.seatbid.bid[i].ext.prebid
// DealPriority represents priority of deal bid. If its non deal bid then value will be 0
// DealTierSatisfied true represents corresponding bid has satisfied the deal tier
type ExtBidPrebid struct {
	Cache             *ExtBidPrebidCache  `json:"cache,omitempty"`
	DealPriority      int                 `json:"dealpriority,omitempty"`
	DealTierSatisfied bool                `json:"dealtiersatisfied,omitempty"`
	Meta              *ExtBidPrebidMeta   `json:"meta,omitempty"`
	Targeting         map[string]string   `json:"targeting,omitempty"`
	TargetBidderCode  string              `json:"targetbiddercode,omitempty"`
	Type              BidType             `json:"type,omitempty"`
	Video             *ExtBidPrebidVideo  `json:"video,omitempty"`
	Events            *ExtBidPrebidEvents `json:"events,omitempty"`
	BidId             string              `json:"bidid,omitempty"`
	Passthrough       json.RawMessage     `json:"passthrough,omitempty"`
	Floors            *ExtBidPrebidFloors `json:"floors,omitempty"`
}

// ExtBidPrebidFloors defines the contract for bidresponse.seatbid.bid[i].ext.prebid.floors
type ExtBidPrebidFloors struct {
	FloorRule      string  `json:"floorRule,omitempty"`
	FloorRuleValue float64 `json:"floorRuleValue,omitempty"`
	FloorValue     float64 `json:"floorValue,omitempty"`
	FloorCurrency  string  `json:"floorCurrency,omitempty"`
}

// ExtBidPrebidCache defines the contract for  bidresponse.seatbid.bid[i].ext.prebid.cache
type ExtBidPrebidCache struct {
	Key  string                 `json:"key"`
	Url  string                 `json:"url"`
	Bids *ExtBidPrebidCacheBids `json:"bids,omitempty"`
}

type ExtBidPrebidCacheBids struct {
	Url     string `json:"url"`
	CacheId string `json:"cacheId"`
}

// ExtBidPrebidMeta defines the contract for bidresponse.seatbid.bid[i].ext.prebid.meta
type ExtBidPrebidMeta struct {
	AdapterCode          string          `json:"adaptercode,omitempty"`
	AdvertiserDomains    []string        `json:"advertiserDomains,omitempty"`
	AdvertiserID         int             `json:"advertiserId,omitempty"`
	AdvertiserName       string          `json:"advertiserName,omitempty"`
	AgencyID             int             `json:"agencyId,omitempty"`
	AgencyName           string          `json:"agencyName,omitempty"`
	BrandID              int             `json:"brandId,omitempty"`
	BrandName            string          `json:"brandName,omitempty"`
	DChain               json.RawMessage `json:"dchain,omitempty"`
	DemandSource         string          `json:"demandSource,omitempty"`
	MediaType            string          `json:"mediaType,omitempty"`
	NetworkID            int             `json:"networkId,omitempty"`
	NetworkName          string          `json:"networkName,omitempty"`
	PrimaryCategoryID    string          `json:"primaryCatId,omitempty"`
	RendererName         string          `json:"rendererName,omitempty"`
	RendererVersion      string          `json:"rendererVersion,omitempty"`
	RendererData         json.RawMessage `json:"rendererData,omitempty"`
	RendererUrl          string          `json:"rendererUrl,omitempty"`
	SecondaryCategoryIDs []string        `json:"secondaryCatIds,omitempty"`
	Seat                 string          `json:"seat,omitempty"`
}

// ExtBidPrebidVideo defines the contract for bidresponse.seatbid.bid[i].ext.prebid.video
type ExtBidPrebidVideo struct {
	Duration        int    `json:"duration"`
	PrimaryCategory string `json:"primary_category"`
}

// ExtBidPrebidEvents defines the contract for bidresponse.seatbid.bid[i].ext.prebid.events
type ExtBidPrebidEvents struct {
	Win string `json:"win,omitempty"`
	Imp string `json:"imp,omitempty"`
}

// ExtBidDSA defines the contract for bidresponse.seatbid.bid[i].ext.dsa
type ExtBidDSA struct {
	AdRender *int8  `json:"adrender,omitempty"`
	Behalf   string `json:"behalf,omitempty"`
	Paid     string `json:"paid,omitempty"`
}

// BidType describes the allowed values for bidresponse.seatbid.bid[i].ext.prebid.type
type BidType string

const (
	BidTypeBanner BidType = "banner"
	BidTypeVideo  BidType = "video"
	BidTypeAudio  BidType = "audio"
	BidTypeNative BidType = "native"
)

func BidTypes() []BidType {
	return []BidType{
		BidTypeBanner,
		BidTypeVideo,
		BidTypeAudio,
		BidTypeNative,
	}
}

func ParseBidType(bidType string) (BidType, error) {
	switch bidType {
	case "banner":
		return BidTypeBanner, nil
	case "video":
		return BidTypeVideo, nil
	case "audio":
		return BidTypeAudio, nil
	case "native":
		return BidTypeNative, nil
	default:
		return "", fmt.Errorf("invalid BidType: %s", bidType)
	}
}

// TargetingKeys are used throughout Prebid as keys which can be used in an ad server like DFP.
// Clients set the values we assign on the request to the ad server, where they can be substituted like macros into
// Creatives.
//
// Removing one of these, or changing the semantics of what we store there, will probably break the
// line item setups for many publishers.
//
// These are especially important to Prebid Mobile. It's much more cumbersome for a Mobile App to update code
// than it is for a website. As a result, they rely heavily on these targeting keys so that any changes can
// be made on Prebid Server and the Ad Server's line items.
type TargetingKey string

const (
	PbKey TargetingKey = "_pb"

	// EnvKey exists to support the Prebid Universal Creative. If it exists, the only legal value is mobile-app.
	// It will exist only if the incoming bidRequest defined request.app instead of request.site.
	EnvKey TargetingKey = "_env"

	// CacheHostKey and CachePathKey exist to supply cache host and path as targeting parameters
	CacheHostKey TargetingKey = "_cache_host"
	CachePathKey TargetingKey = "_cache_path"

	// BidderKey is the name of the Bidder. For example, "appnexus" or "rubicon".
	BidderKey TargetingKey = "_bidder"
	SizeKey   TargetingKey = "_size"
	DealKey   TargetingKey = "_deal"

	// FormatKey is the format of the bid. For example, "video", "banner"
	FormatKey TargetingKey = "_format"

	// CacheKey and VastCacheKey store UUIDs which can be used to fetch things from prebid cache.
	// Callers should *never* assume that either of these exist, since the call to the cache may always fail.
	//
	// VastCacheKey's UUID will fetch the entire bid JSON, while VastCacheKey will fetch just the VAST XML.
	// VastCacheKey will only ever exist for Video bids.
	CacheKey     TargetingKey = "_cache_id"
	VastCacheKey TargetingKey = "_uuid"

	// EnvAppValue used as a value for EnvKey
	EnvAppValue string = "mobile-app"

	CategoryDurationKey TargetingKey = "_pb_cat_dur"
)

func (key TargetingKey) BidderKey(prefix string, bidder BidderName, maxLength int) string {
	s := prefix + string(key) + "_" + string(bidder)
	if maxLength != 0 {
		return s[:min(len(s), maxLength)]
	}
	return s
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (key TargetingKey) TruncateKey(prefix string, maxLength int) string {
	result := prefix + string(key)
	if maxLength > 0 {
		return result[:min(len(result), maxLength)]
	}
	return result
}

const (
	StoredRequestAttributes = "storedrequestattributes"
	OriginalBidCpmKey       = "origbidcpm"
	OriginalBidCurKey       = "origbidcur"
	Passthrough             = "passthrough"
)
