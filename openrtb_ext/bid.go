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
	ResponseTimeMillis int               `json:"responsetimemillis"`
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
