package openrtb_ext

// ExtImpSmaato defines the contract for bidrequest.imp[i].ext.smaato
// PublisherId is mandatory parameters, others are optional parameters
// TagId is identifier for specific ad placement or ad tag

type ExtImpSmaato struct {
	Id     string `json:"id"`
	TagId  string `json:"tagid"`
	Instl  int8   `json:"instl"`
	Secure *int8  `json:"secure"`
}
