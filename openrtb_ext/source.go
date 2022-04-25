package openrtb_ext

// ExtSource defines the contract for bidrequest.source.ext
type ExtSource struct {
	SChain *ExtRequestPrebidSChainSChain `json:"schain"`
}
