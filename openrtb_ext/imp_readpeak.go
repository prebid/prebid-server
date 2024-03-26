package openrtb_ext

type ImpExtReadpeak struct {
	PublisherId	string `json:"publisherId"`
	SiteId 		string `json:"siteId"`
	Bidfloor    int	   `json:"bidfloor"`
	TagId		int    `json:"tagId"`
}
