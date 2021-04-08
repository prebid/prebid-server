package openrtb_ext

// ExtImpZemanta defines the contract for bidrequest.imp[i].ext.zemanta
type ExtImpZemanta struct {
	Publisher ExtImpZemantaPublisher `json:"publisher"`
	TagId     string                 `json:"tagid"`
	BCat      []string               `json:"bcat"`
	BAdv      []string               `json:"badv"`
}

type ExtImpZemantaPublisher struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}
