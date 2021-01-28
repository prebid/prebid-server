package openrtb_ext

// ExtImpUnicorn defines the contract for bidrequest.imp[i].ext.unicorn
type ExtImpUnicorn struct {
	PlacementId  string                 `json:"placementId"`
	PublisherId  int                    `json:"publisherId"`
	MediaId      string                 `json:"mediaId"`
	AccountId    int                    `json:"accountId"`
}
