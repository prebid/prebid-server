package openrtb_ext

// ExtImpCriteo defines the contract for bidrequest.imp[i].ext.criteo
type ExtImpCriteo struct {
	ZoneID    int64 `json:"zoneId"`
	NetworkID int64 `json:"networkId"`
}
