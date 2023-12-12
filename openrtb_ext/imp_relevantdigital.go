package openrtb_ext

type ExtRelevantDigital struct {
	AccountId   string `json:"accountId"`
	PlacementId string `json:"placementId"`
	Host        string `json:"pbsHost"`
	PbsBufferMs int    `json:"pbsBufferMs"`
}
