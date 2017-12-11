package openrtb_ext

import "github.com/mxmCherry/openrtb"

// ExtBid defines the contract for bidresponse.seatbid.bid[i].ext
type ExtBid struct {
	Prebid *ExtBidPrebid   `json:"prebid,omitempty"`
	Bidder openrtb.RawJSON `json:"bidder,omitempty"`
}

// ExtBidPrebid defines the contract for bidresponse.seatbid.bid[i].ext.prebid
type ExtBidPrebid struct {
	Cache              *ExtResponseCache `json:"cache,omitempty"`
	Targeting          map[string]string `json:"targeting,omitempty"`
	Type               BidType           `json:"type"`
}

// ExtResponseCache defines the contract for  bidresponse.seatbid.bid[i].ext.prebid.cache
type ExtResponseCache struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

// BidType describes the allowed values for bidresponse.seatbid.bid[i].ext.prebid.type
type BidType string

const (
	BidTypeBanner BidType = "banner"
	BidTypeVideo          = "video"
	BidTypeAudio          = "audio"
	BidTypeNative         = "native"
)

// This also duplicates code in pbs_light, which should be moved to /pbs/targeting. But that is beyond the current
// scope, and likely moot if the non-openrtb endpoint goes away.
type TargetingKey string

const (
	HbpbConstantKey TargetingKey = "hb_pb"
	HbBidderConstantKey TargetingKey = "hb_bidder"
	HbSizeConstantKey TargetingKey = "hb_size"
	HbCreativeLoadMethodConstantKey TargetingKey = "hb_creative_loadtype"
	HbCacheIdConstantKey TargetingKey = "hb_cache_id"
	HbDealIdConstantKey TargetingKey = "hb_deal"
	// These are not keys, but values used by hbCreativeLoadMethodConstantKey
	HbCreativeLoadMethodHTML string = "html"
	HbCreativeLoadMethodDemandSDK string = "demand_sdk"
)

func (key TargetingKey) BidderKey(bidder BidderName, maxLength int) string {
	s := string(key) + "_" + string(bidder)
	if maxLength != 0 {
		s = s[:min(len(s), maxLength)]
	}
	return s
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
