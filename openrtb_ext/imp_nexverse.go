package openrtb_ext

// ImpExtNexverse defines the contract for bidrequest.imp[i].ext.prebid.bidder.nexverse
type ImpExtNexverse struct {
	UID     string `json:"uid"`
	PubID   string `json:"pubId"`
	PubEpid string `json:"pubEpid"`
	IsDebug bool   `json:"isDebug,omitempty"`
}
