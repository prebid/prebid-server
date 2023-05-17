package openrtb_ext

// ExtImpAdprime defines adprime specifiec param
type ExtImpAdprime struct {
	TagID     string   `json:"TagID"`
	Keywords  []string `json:"keywords"`
	Audiences []string `json:"audiences"`
}
