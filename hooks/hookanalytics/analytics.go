package hookanalytics

type Analytics struct {
	Activities []Activity `json:"activities"`
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
	Bidders  []string `json:"bidders,omitempty"`
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
