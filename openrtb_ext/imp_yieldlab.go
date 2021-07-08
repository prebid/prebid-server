package openrtb_ext

// ExtImpYieldlab defines the contract for bidrequest.imp[i].ext.yieldlab
type ExtImpYieldlab struct {
	AdslotID  string            `json:"adslotId"`
	SupplyID  string            `json:"supplyId"`
	AdSize    string            `json:"adSize"`
	Targeting map[string]string `json:"targeting"`
	ExtId     string            `json:"extId"`
}
