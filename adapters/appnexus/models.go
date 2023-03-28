package appnexus

import (
	"encoding/json"

	"github.com/prebid/prebid-server/openrtb_ext"
)

type appnexusImpExtAppnexus struct {
	PlacementID       int             `json:"placement_id,omitempty"`
	Keywords          string          `json:"keywords,omitempty"`
	TrafficSourceCode string          `json:"traffic_source_code,omitempty"`
	UsePmtRule        *bool           `json:"use_pmt_rule,omitempty"`
	PrivateSizes      json.RawMessage `json:"private_sizes,omitempty"`
}

type appnexusImpExt struct {
	Appnexus appnexusImpExtAppnexus `json:"appnexus"`
}

type appnexusBidExtVideo struct {
	Duration int `json:"duration"`
}

type appnexusBidExtCreative struct {
	Video appnexusBidExtVideo `json:"video"`
}

type appnexusBidExtAppnexus struct {
	BidType       int                    `json:"bid_ad_type"`
	BrandId       int                    `json:"brand_id"`
	BrandCategory int                    `json:"brand_category_id"`
	CreativeInfo  appnexusBidExtCreative `json:"creative_info"`
	DealPriority  int                    `json:"deal_priority"`
}

type appnexusBidExt struct {
	Appnexus appnexusBidExtAppnexus `json:"appnexus"`
}

type appnexusReqExtAppnexus struct {
	IncludeBrandCategory    *bool  `json:"include_brand_category,omitempty"`
	BrandCategoryUniqueness *bool  `json:"brand_category_uniqueness,omitempty"`
	IsAMP                   int    `json:"is_amp,omitempty"`
	HeaderBiddingSource     int    `json:"hb_source,omitempty"`
	AdPodId                 string `json:"adpod_id,omitempty"`
}

// Full request extension including appnexus extension object
type appnexusReqExt struct {
	openrtb_ext.ExtRequest
	Appnexus *appnexusReqExtAppnexus `json:"appnexus,omitempty"`
}
