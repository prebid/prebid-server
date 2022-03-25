package openrtb_ext

// ExtImpAdyoulike defines the contract for bidrequest.imp[i].ext.adyoulike
type ExtImpAdyoulike struct {
	// placementId, only mandatory field
	PlacementId string `json:"placement"`

	// Id of the forced campaign
	Campaign string `json:"campaign"`
	// Id of the forced track
	Track string `json:"track"`
	// Id of the forced creative
	Creative string `json:"creative"`
	// Context of the campaign values [SSP|AdServer]
	Source string `json:"source"`
	// Abitrary Id used for debug purpose
	Debug string `json:"debug"`
}
