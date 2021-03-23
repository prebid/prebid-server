package model

// SKANIDList format: https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/skadnetwork.md#iabtl-managed-skadnetwork-id-list
type SKANIDList struct {
	CompanyName    string           `json:"company_name"`
	CompanyAddress string           `json:"company_address"`
	CompanyDomain  string           `json:"company_domain"`
	SKAdNetworkIDs []SKAdNetworkIDs `json:"skadnetwork_ids"`
}

type SKAdNetworkIDs struct {
	ID            int    `json:"id"`
	EntityName    string `json:"entity_name"`
	EntityDomain  string `json:"entity_domain"`
	SKAdNetworkID string `json:"skadnetwork_id"`
	CreationDate  string `json:"creation_date"`
}
