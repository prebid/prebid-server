package openrtb_ext

// ExtImpVASTBidder defines the contract for bidrequest.imp[i].ext.vastbidder
type ExtImpVASTBidder struct {
	Tags    []*ExtImpVASTBidderTag `json:"tags,omitempty"`
	Parser  string                 `json:"parser,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
	Cookies map[string]string      `json:"cookies,omitempty"`
}

// ExtImpVASTBidderTag defines the contract for bidrequest.imp[i].ext.pubmatic.tags[i]
type ExtImpVASTBidderTag struct {
	TagID    string                 `json:"tagid"`
	URL      string                 `json:"url"`
	Duration int                    `json:"dur"`
	Price    float64                `json:"price"`
	Params   map[string]interface{} `json:"params,omitempty"`
}
