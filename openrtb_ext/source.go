package openrtb_ext

import "github.com/prebid/openrtb/v19/openrtb2"

// ExtSource defines the contract for bidrequest.source.ext
type ExtSource struct {
	SChain *openrtb2.SupplyChain `json:"schain"`
}
