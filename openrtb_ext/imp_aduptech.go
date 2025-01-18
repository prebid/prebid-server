package openrtb_ext

import "encoding/json"

type ExtImpAduptech struct {
	Publisher string          `json:"publisher"`
	Placement string          `json:"placement"`
	Query     string          `json:"query"`
	AdTest    string          `json:"adtest"`
	Ext       json.RawMessage `json:"ext,omitempty"`
}
