package openrtb_ext

// ExtImpNanoInteractive defines the contract for bidrequest.imp[i].ext.nanointeractive
type ExtImpNanoInteractive struct {
	Pid string `json:"pid"`
	Ref string `json:"ref, omitempty"`
}
