// Package hookanalytics provides basic primitives for use by the hook modules.
//
// Structures of the package allow modules to provide information
// about what activity has been performed against the hook payload.
package hookanalytics

type Analytics struct {
	Activities []Activity `json:"activities,omitempty"`
}

type Activity struct {
	Name    string         `json:"name"`
	Status  ActivityStatus `json:"status"`
	Results []Result       `json:"results,omitempty"`
}

type ActivityStatus string

const (
	ActivityStatusSuccess ActivityStatus = "success"
	ActivityStatusError   ActivityStatus = "error"
)

type Result struct {
	Status    ResultStatus           `json:"status,omitempty"`
	Values    map[string]interface{} `json:"values,omitempty"`
	AppliedTo AppliedTo              `json:"appliedto,omitempty"`
}

type AppliedTo struct {
	Bidder   string   `json:"bidder,omitempty"`
	BidIds   []string `json:"bidids,omitempty"`
	ImpIds   []string `json:"impids,omitempty"`
	Request  bool     `json:"request,omitempty"`
	Response bool     `json:"response,omitempty"`
}

type ResultStatus string

const (
	ResultStatusAllow  ResultStatus = "success-allow"
	ResultStatusBlock  ResultStatus = "success-block"
	ResultStatusModify ResultStatus = "success-modify"
	ResultStatusError  ResultStatus = "error"
)
