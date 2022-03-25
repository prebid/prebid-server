package adhese

import "github.com/mxmCherry/openrtb/v15/openrtb2"

type AdheseOriginData struct {
	Priority                  string `json:"priority"`
	OrderProperty             string `json:"orderProperty"`
	AdFormat                  string `json:"adFormat"`
	AdType                    string `json:"adType"`
	AdspaceId                 string `json:"adspaceId"`
	LibId                     string `json:"libId"`
	SlotID                    string `json:"slotID,omitempty"`
	ViewableImpressionCounter string `json:"viewableImpressionCounter"`
}

type AdheseExt struct {
	Id                string `json:"id"`
	OrderId           string `json:"orderId"`
	ImpressionCounter string `json:"impressionCounter"`
	Tag               string `json:"tag,omitempty"`
	Ext               string `json:"ext"`
}

type AdheseBid struct {
	Origin         string               `json:"origin"`
	OriginData     openrtb2.BidResponse `json:"originData"`
	OriginInstance string               `json:"originInstance,omitempty"`
	Body           string               `json:"body,omitempty"`
	Height         string               `json:"height"`
	Width          string               `json:"width"`
	Extension      Prebid               `json:"extension"`
}

type Prebid struct {
	Prebid CPM `json:"prebid"`
}

type CPM struct {
	Cpm CPMValues `json:"cpm"`
}

type CPMValues struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
