package openrtb_ext

// ExtImpAdkernel defines the contract for bidrequest.imp[i].ext.adkernel
type ExtImpAdYouLike struct {
	// placementId, only mandatory field
	PlacementId string `json:"placementId"`
	
	// Id of the forced campaign	
	Campaign string    `json:"Campaign"`
	// Id of the forced track
	Track string       `json:"Track"`
	// Id of the forced creative
	Creative string    `json:"Creative"`
	// Context of the campaign values [SSP|AdServer]
	Source string	  `json:"Source"`
	// Abitrary Id used for debug purpose
	Debug string	  `json:"Debug"`

}
