package openrtb_ext

// ExtImpAMX is the imp.ext format for the AMX bidder
type ExtImpAMX struct {
	TagID    string `json:"tagId,omitempty"`
	AdUnitID string `json:"adUnitId,omitempty"`
}
