package openrtb_ext

// ExtImpForeshop defines the contract for bidrequest.imp[i].foreshop
type ExtImpForeshop struct {
	SourceID    int    `json:"sourceId"`
	PlacementID int    `json:"placementId,omitempty"`
	Host        string `json:"host"`
}
