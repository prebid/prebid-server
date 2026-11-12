package doohcreativeapproval

type approvalStatus string

const (
	approvalStatusApproved approvalStatus = "approved"
	approvalStatusRejected approvalStatus = "rejected"
	approvalStatusPending  approvalStatus = "pending"
)

type approvalRequest struct {
	AccountID string             `json:"account_id"`
	Creatives []creativeApproval `json:"creatives"`
}

type approvalResponse struct {
	Creatives []creativeApprovalResult `json:"creatives"`
}

type creativeApproval struct {
	CreativeApprovalID string   `json:"creative_approval_id"`
	Bidder             string   `json:"bidder"`
	CreativeID         string   `json:"creative_id"`
	AdID               string   `json:"ad_id,omitempty"`
	CampaignID         string   `json:"campaign_id,omitempty"`
	AdvertiserDomains  []string `json:"advertiser_domains,omitempty"`
	Categories         []string `json:"categories,omitempty"`
	CategoryTaxonomy   int      `json:"cat_tax,omitempty"`
	MediaType          string   `json:"media_type,omitempty"`
	Width              int64    `json:"width,omitempty"`
	Height             int64    `json:"height,omitempty"`
	Duration           int64    `json:"duration,omitempty"`
	DealID             string   `json:"deal_id,omitempty"`
	IURL               string   `json:"iurl,omitempty"`
}

type creativeApprovalResult struct {
	CreativeApprovalID string         `json:"creative_approval_id"`
	Status             approvalStatus `json:"status"`
}

func isValidApprovalStatus(status approvalStatus) bool {
	switch status {
	case approvalStatusApproved, approvalStatusRejected, approvalStatusPending:
		return true
	default:
		return false
	}
}
