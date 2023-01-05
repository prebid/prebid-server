package openrtb_ext

// ExtImpYieldlab defines the contract for bidrequest.imp[i].ext.prebid.bidder.yieldlab
type ExtImpYieldlab struct {
	AdslotID  string            `json:"adslotId"`
	SupplyID  string            `json:"supplyId"`
	Targeting map[string]string `json:"targeting"`
	ExtId     string            `json:"extId"`
}
