package openrtb_ext

// ExtImpMobilefuse defines the contract for bidrequest.imp[i].ext.mobilefuse
type ExtImpMobilefuse struct {
	PlacementId int    `json:"placement_id"`
	PublisherId int    `json:"pub_id"`
	TagidSrc    string `json:"tagid_src"`
}
