package openrtb_ext

// ExtImpAdkernelAdn defines the contract for bidrequest.imp[i].ext.adkernelAdn
type ExtImpAdkernelAdn struct {
	PublisherID int    `json:"pubId"`
	Host        string `json:"host,omitempty"`
}
