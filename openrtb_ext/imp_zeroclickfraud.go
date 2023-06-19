package openrtb_ext

// ExtImpZeroClickFraud defines the contract for bidrequest.imp[i].ext.prebid.bidder.datablocks
type ExtImpZeroClickFraud struct {
	SourceId int    `json:"sourceId"`
	Host     string `json:"host"`
}
