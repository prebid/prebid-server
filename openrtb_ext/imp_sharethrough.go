package openrtb_ext

type ExtImpSharethrough struct {
	Pkey       string  `json:"pkey"`
	Iframe     bool    `json:"iframe"`
	IframeSize []int   `json:"iframeSize"`
	BidFloor   float64 `json:"bidfloor"`
}

// ExtImpSharethrough defines the contract for bidrequest.imp[i].ext.sharethrough
type ExtImpSharethroughResponse struct {
	AdServerRequestID string                       `json:"adserverRequestId"`
	BidID             string                       `json:"bidId"`
	Creatives         []ExtImpSharethroughCreative `json:"creatives"`
}
type ExtImpSharethroughCreative struct {
	AuctionWinID string                             `json:"auctionWinId"`
	CPM          float64                            `json:"cpm"`
	Metadata     ExtImpSharethroughCreativeMetadata `json:"creative"`
}

type ExtImpSharethroughCreativeMetadata struct {
	CampaignKey string `json:"campaign_key"`
	CreativeKey string `json:"creative_key"`
	DealID      string `json:"deal_id"`
}
