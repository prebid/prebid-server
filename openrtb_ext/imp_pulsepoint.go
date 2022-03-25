package openrtb_ext

// ExtImpPulsePoint defines the json spec for bidrequest.imp[i].ext.pulsepoint
// PubId/TagId are mandatory params

type ExtImpPulsePoint struct {
	PubID int `json:"cp"`
	TagID int `json:"ct"`
}
