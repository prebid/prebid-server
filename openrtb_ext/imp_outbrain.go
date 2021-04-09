package openrtb_ext

// ExtImpOutbrain defines the contract for bidrequest.imp[i].ext.outbrain
type ExtImpOutbrain struct {
	Publisher ExtImpOutbrainPublisher `json:"publisher"`
	TagId     string                  `json:"tagid"`
	BCat      []string                `json:"bcat"`
	BAdv      []string                `json:"badv"`
}

type ExtImpOutbrainPublisher struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}
