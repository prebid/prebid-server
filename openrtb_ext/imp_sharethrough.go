package openrtb_ext

type ExtImpSharethrough struct {
	Pkey string   `json:"pkey"`
	BAdv []string `json:"badv,omitempty"`
	BCat []string `json:"bcat,omitempty"`
}
