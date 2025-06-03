package openrtb_ext

type ExtImpMissena struct {
	APIKey    string         `json:"apiKey"`
	Formats   []string       `json:"formats"`
	Placement string         `json:"placement"`
	TestMode  string         `json:"test"`
	Settings  map[string]any `json:"settings,omitempty"`
}
