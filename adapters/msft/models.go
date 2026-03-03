package msft

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type extraAdapterInfo struct {
	HBSource      int `json:"hb_source"`
	HBSourceVideo int `json:"hb_source_video"`
}

// impExtIncoming defines the incoming data contract to Prebid Server.
type impExtIncoming struct {
	Bidder openrtb_ext.ImpExtMsft `json:"bidder"`
	GPID   string                 `json:"gpid"`
}

// impExtOutgoing defines the outgoing data contract from Prebid Server to Microsoft.
type impExtOutgoing struct {
	Appnexus impExtOutgoingAppnexus `json:"appnexus"`
	GPID     string                 `json:"gpid,omitempty"`
}

type impExtOutgoingAppnexus struct {
	PlacementID       int    `json:"placement_id,omitempty"`
	AllowSmallerSizes *bool  `json:"allow_smaller_sizes,omitempty"`
	UsePmtRule        *bool  `json:"use_pmt_rule,omitempty"`
	Keywords          string `json:"keywords,omitempty"`
	TrafficSourceCode string `json:"traffic_source_code,omitempty"`
	PubClick          string `json:"pub_click,omitempty"`
	ExtInvCode        string `json:"ext_inv_code,omitempty"`
	ExtImpID          string `json:"ext_imp_id,omitempty"`
}

type bidExtVideo struct {
	Duration int `json:"duration"`
}

type bidExtCreative struct {
	Video bidExtVideo `json:"video"`
}

type bidExtAppnexus struct {
	BidType       int            `json:"bid_ad_type"`
	BrandId       int            `json:"brand_id"`
	BrandCategory int            `json:"brand_category_id"`
	CreativeInfo  bidExtCreative `json:"creative_info"`
	DealPriority  int            `json:"deal_priority"`
}

type bidExt struct {
	Appnexus bidExtAppnexus `json:"appnexus"`
}

type requestExAppnexus struct {
	IncludeBrandCategory    *bool `json:"include_brand_category,omitempty"`
	BrandCategoryUniqueness *bool `json:"brand_category_uniqueness,omitempty"`
	IsAMP                   int   `json:"is_amp,omitempty"`
	HeaderBiddingSource     int   `json:"hb_source,omitempty"`
}
