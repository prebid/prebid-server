package openrtb_ext

// ExtImpMarsmedia defines the contract for bidrequest.imp[i].ext.marsmedia
type ExtImpMarsmedia struct {
	Publisher	string	`json:"publisher"`
	ZoneId		string	`json:"ZoneId"`
	BidFloor	float64	`json:"bidfloor"`
}
