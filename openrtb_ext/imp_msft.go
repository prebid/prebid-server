package openrtb_ext

import "github.com/prebid/openrtb/v20/adcom1"

// ImpExtMsft defines the contract for Microsoft bidder parameters.
type ImpExtMsft struct {
	PlacementID       int                   `json:"placement_id"`
	Member            int                   `json:"member"`
	InvCode           string                `json:"inv_code"`
	AllowSmallerSizes *bool                 `json:"allow_smaller_sizes"`
	UsePaymentRule    *bool                 `json:"use_pmt_rule"`
	Keywords          string                `json:"keywords"`
	TrafficSourceCode string                `json:"traffic_source_code"`
	PubClick          string                `json:"pubclick"`
	ExtInvCode        string                `json:"ext_inv_code"`
	ExtImpID          string                `json:"ext_imp_id"`
	BannerFrameworks  []adcom1.APIFramework `json:"banner_frameworks"`
}
