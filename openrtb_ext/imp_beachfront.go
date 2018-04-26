package openrtb_ext

type ExtImpBeachfront struct {
	CrID     string  `json:"crid"`
	AppId    string  `json:"appId"`
	BidFloor float64 `json:"bidfloor"`
}

/*
type ExtImpBeachfront struct {
	Domain         string        `json:"domain,omitempty"`
	BidFloor       string        `json:"bidfloor,omitempty"`
	Page           string        `json:"page,omitempty"`
	Referrer       string        `json:"referrer,omitempty"`
	Search         string        `json:"search,omitempty"`
	DeviceOs       string        `json:"deviceOs,omitempty"`
	DeviceModel    string        `json:"deviceModel,omitempty"`
	IsMobile       int           `json:"isMobile,omitempty"`
	Ua             string        `json:"ua,omitempty"`
	Dnt            int           `json:"dnt,omitempty"`
	AdapterName    string        `json:"adapterName,omitempty"`
	AdapterVersion string        `json:"adapterVersion,omitempty"`
	Ip             string        `json:"ip,omitempty"`
}
*/
