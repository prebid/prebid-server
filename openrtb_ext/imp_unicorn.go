package openrtb_ext

// ExtImpUnicorn defines the contract for bidrequest.imp[i].ext.unicorn
type ExtImpUnicorn struct {
	PlacementID string `json:"placementId,omitempty"`
	PublisherID int    `json:"publisherId,omitempty"`
	MediaID     string `json:"mediaId"`
	AccountID   int    `json:"accountId"`

	Reward         int    `json:"reward,omitempty"`
	Region         string `json:"region,omitempty"`
	SKADNSupported bool   `json:"skadn_supported,omitempty"`
	MRAIDSupported bool   `json:"mraid_supported,omitempty"`
}
