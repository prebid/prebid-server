package appnexus

import (
	"encoding/json"
)

// impExt defines the outgoing data contract.
type impExt struct {
	Appnexus impExtAppnexus `json:"appnexus"`
	GPID     string         `json:"gpid,omitempty"`
}

type impExtAppnexus struct {
	PlacementID       int             `json:"placement_id,omitempty"`
	Keywords          string          `json:"keywords,omitempty"`
	TrafficSourceCode string          `json:"traffic_source_code,omitempty"`
	UsePmtRule        *bool           `json:"use_pmt_rule,omitempty"`
	PrivateSizes      json.RawMessage `json:"private_sizes,omitempty"`
	ExtInvCode        string          `json:"ext_inv_code,omitempty"`
	ExternalImpID     string          `json:"external_imp_id,omitempty"`
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

type bidReqExtAppnexus struct {
	IncludeBrandCategory    *bool  `json:"include_brand_category,omitempty"`
	BrandCategoryUniqueness *bool  `json:"brand_category_uniqueness,omitempty"`
	IsAMP                   int    `json:"is_amp,omitempty"`
	HeaderBiddingSource     int    `json:"hb_source,omitempty"`
	AdPodID                 string `json:"adpod_id,omitempty"`
}
