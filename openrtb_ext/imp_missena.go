package openrtb_ext

type ExtImpMissena struct {
	APIKey    string         `json:"apiKey"`
	Formats   []string       `json:"formats"`
	Placement string         `json:"placement"`
	Sample    string         `json:"sample"`
	Settings  map[string]any `json:"settings,omitempty"`
}
