package appnexus

import (
	"encoding/json"

	"github.com/prebid/prebid-server/openrtb_ext"
)

type bidderImpExtAppnexus struct {
	PlacementID       int             `json:"placement_id,omitempty"`
	Keywords          string          `json:"keywords,omitempty"`
	TrafficSourceCode string          `json:"traffic_source_code,omitempty"`
	UsePmtRule        *bool           `json:"use_pmt_rule,omitempty"`
	PrivateSizes      json.RawMessage `json:"private_sizes,omitempty"`
	ExtInvCode        string          `json:"ext_inv_code,omitempty"`
	ExternalImpId     string          `json:"external_imp_id,omitempty"`
}

type bidderImpExt struct {
	Appnexus bidderImpExtAppnexus `json:"appnexus"`
}

type bidderExtVideo struct {
	Duration int `json:"duration"`
}

type bidderExtCreative struct {
	Video bidderExtVideo `json:"video"`
}

type bidderExtAppnexus struct {
	BidType       int               `json:"bid_ad_type"`
	BrandId       int               `json:"brand_id"`
	BrandCategory int               `json:"brand_category_id"`
	CreativeInfo  bidderExtCreative `json:"creative_info"`
	DealPriority  int               `json:"deal_priority"`
}

type bidderExt struct {
	Appnexus bidderExtAppnexus `json:"appnexus"`
}

type bidderReqExtAppnexus struct {
	IncludeBrandCategory    *bool  `json:"include_brand_category,omitempty"`
	BrandCategoryUniqueness *bool  `json:"brand_category_uniqueness,omitempty"`
	IsAMP                   int    `json:"is_amp,omitempty"`
	HeaderBiddingSource     int    `json:"hb_source,omitempty"`
	AdPodId                 string `json:"adpod_id,omitempty"`
}

// Full request extension including appnexus extension object
type bidderReqExt struct {
	openrtb_ext.ExtRequest
	Appnexus *bidderReqExtAppnexus `json:"appnexus,omitempty"`
}
