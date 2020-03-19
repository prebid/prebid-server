package openrtb_ext

// ExtImpNanoInteractive defines the contract for bidrequest.imp[i].ext.nanointeractive
type ExtImpNanoInteractive struct {
	Pid      string   `json:"pid"`
	Nq       []string `json:"nq, omitempty"`
	Category string   `json:"category, omitempty"`
	SubId    string   `json:"subId, omitempty"`
	Ref      string   `json:"ref, omitempty"`
}
