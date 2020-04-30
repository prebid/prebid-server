package openrtb_ext

// ExtImpAdkernel defines the contract for bidrequest.imp[i].ext.adkernel
type ExtImpAdkernel struct {
	ZoneId int    `json:"zoneId"`
	Host   string `json:"host"`
}
