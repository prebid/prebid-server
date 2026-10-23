package openrtb_ext

// ExtImpChartboost defines the contract for bidrequest.imp[i].ext.prebid.bidder.chartboost
//
// These are supplied by the stored imp and forwarded untouched; they are read on the
// Chartboost side.
type ExtImpChartboost struct {
	AppID        string `json:"app_id"`
	LocationName string `json:"location_name"`
}
