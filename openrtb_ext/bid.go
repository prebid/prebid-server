package openrtb_ext

import (
	"encoding/json"
	"fmt"
)

// ExtBid defines the contract for bidresponse.seatbid.bid[i].ext
type ExtBid struct {
	Prebid *ExtBidPrebid   `json:"prebid,omitempty"`
	Bidder json.RawMessage `json:"bidder,omitempty"`
}

// ExtBidPrebid defines the contract for bidresponse.seatbid.bid[i].ext.prebid
type ExtBidPrebid struct {
	Cache     *ExtBidPrebidCache `json:"cache,omitempty"`
	Targeting map[string]string  `json:"targeting,omitempty"`
	Type      BidType            `json:"type"`
	Video     *ExtBidPrebidVideo `json:"video,omitempty"`
}

// ExtBidPrebidCache defines the contract for  bidresponse.seatbid.bid[i].ext.prebid.cache
type ExtBidPrebidCache struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

// ExtBidPrebidVideo defines the contract for bidresponse.seatbid.bid[i].ext.prebid.video
type ExtBidPrebidVideo struct {
	Duration        int    `json:"duration"`
	PrimaryCategory string `json:"primary_category"`
}

type BidderExt struct {
	CreativeInfo CreativeInfo `json:"creative_info"`
}

type CreativeInfo struct {
	Video Video `json:"video"`
}

type Video struct {
	Duration int `json:"duration"`
}

// BidType describes the allowed values for bidresponse.seatbid.bid[i].ext.prebid.type
type BidType string

const (
	BidTypeBanner BidType = "banner"
	BidTypeVideo          = "video"
	BidTypeAudio          = "audio"
	BidTypeNative         = "native"
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
	HbpbConstantKey TargetingKey = "hb_pb"

	// HbEnvKey exists to support the Prebid Universal Creative. If it exists, the only legal value is mobile-app.
	// It will exist only if the incoming bidRequest defiend request.app instead of request.site.
	HbEnvKey TargetingKey = "hb_env"

	// HbBidderConstantKey is the name of the Bidder. For example, "appnexus" or "rubicon".
	HbBidderConstantKey TargetingKey = "hb_bidder"
	HbSizeConstantKey   TargetingKey = "hb_size"
	HbDealIdConstantKey TargetingKey = "hb_deal"

	// HbCacheKey and HbVastCacheKey store UUIDs which can be used to fetch things from prebid cache.
	// Callers should *never* assume that either of these exist, since the call to the cache may always fail.
	//
	// HbVastCacheKey's UUID will fetch the entire bid JSON, while HbVastCacheKey will fetch just the VAST XML.
	// HbVastCacheKey will only ever exist for Video bids.
	HbCacheKey     TargetingKey = "hb_cache_id"
	HbVastCacheKey TargetingKey = "hb_uuid"

	// This is not a key, but values used by the HbEnvKey
	HbEnvKeyApp string = "mobile-app"

	HbCategoryDurationKey TargetingKey = "hb_pb_cat_dur"
)

func (key TargetingKey) BidderKey(bidder BidderName, maxLength int) string {
	s := string(key) + "_" + string(bidder)
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
