package openrtb_ext

type ExtImpAdmixer struct {
	ZoneId         string                 `json:"zone"`
	CustomBidFloor float64                `json:"customFloor"`
	CustomParams   map[string]interface{} `json:"customParams"`
}
