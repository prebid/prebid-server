package openrtb_ext

// ExtImpIx defines the contract for bidrequest.imp[i].ext.ix
type ExtImpIx struct {
	SiteId string `json:"siteId"`
	Size   []int  `json:"size"`
}
