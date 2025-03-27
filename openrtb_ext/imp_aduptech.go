package openrtb_ext

import "encoding/json"

type ExtImpAduptech struct {
	Publisher string          `json:"publisher"`
	Placement string          `json:"placement"`
	Query     string          `json:"query"`
	AdTest    bool            `json:"adtest"`
	Debug     bool            `json:"debug,omitempty"`
	Ext       json.RawMessage `json:"ext,omitempty"`
}
