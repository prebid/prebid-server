package openrtb_ext

type ExtImpRTBStack struct {
	Route        string                 `json:"route"`
	TagId        string                 `json:"tagId"`
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}
