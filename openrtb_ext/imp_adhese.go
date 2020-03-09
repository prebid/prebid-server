package openrtb_ext

type ExtImpAdhese struct {
	Account  string                  `json:"account"`
	Location string                  `json:"location"`
	Format   string                  `json:"format"`
	Keywords []*AdheseKeywordsParams `json:"targets,omitempty"`
}

type AdheseKeywordsParams struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
