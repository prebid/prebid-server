package openrtb_ext

type ExtImpRTBStack struct {
	Host         string                 `json:"host,omitempty"`
	Query        string                 `json:"query,omitempty"`
	TagId        string                 `json:"tagid,omitempty"`
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}
