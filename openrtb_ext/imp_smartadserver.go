package openrtb_ext

// ExtImpSmartadserver defines the contract for bidrequest.imp[i].ext.smartadserver
type ExtImpSmartadserver struct {
	Domain   string `json:"domain"`
	SiteID   int    `json:"siteId"`
	PageID   int    `json:"pageId"`
	FormatID int    `json:"formatId"`
}
