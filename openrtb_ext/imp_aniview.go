package openrtb_ext

// ImpExtAniview defines the contract for bidrequest.imp[i].ext.prebid.bidder.aniview
type ImpExtAniview struct {
	PublisherId string `json:"AV_PUBLISHERID"`
	ChannelId   string `json:"AV_CHANNELID"`
}
