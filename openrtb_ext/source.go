package openrtb_ext

import "github.com/mxmCherry/openrtb/v16/openrtb2"

// ExtSource defines the contract for bidrequest.source.ext
type ExtSource struct {
	SChain *openrtb2.SupplyChain `json:"schain"`
}
