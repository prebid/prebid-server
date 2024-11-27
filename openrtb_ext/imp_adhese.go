package openrtb_ext

type ExtImpAdhese struct {
	Account  string              `json:"account"`
	Location string              `json:"location"`
	Format   string              `json:"format"`
	Targets  map[string][]string `json:"targets,omitempty"`
}
