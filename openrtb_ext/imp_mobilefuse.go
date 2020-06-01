package openrtb_ext

// ExtImpMobileFuse defines the contract for bidrequest.imp[i].ext.mobilefuse
type ExtImpMobileFuse struct {
	PlacementId int    `json:"placement_id"`
	PublisherId int    `json:"pub_id"`
	TagidSrc    string `json:"tagid_src"`
}
