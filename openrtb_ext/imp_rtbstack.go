package openrtb_ext

type ExtImpRTBStack struct {
	Endpoint     string                 `json:"endpoint,omitempty"`
	TagId        string                 `json:"tagid,omitempty"`
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}
