package openrtb_ext

// ExtImpCriteo defines the contract for bidrequest.imp[i].ext.prebid.bidder.criteo
type ExtImpCriteo struct {
	ZoneID    int64  `json:"zoneId"`
	NetworkID int64  `json:"networkId"`
	UID       int64  `json:"uid"`
	PubID     string `json:"pubid"`
}
