package openrtb_ext

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb"
)

type ExtImpAdhese struct {
	Account  string          `json:"account"`
	Location string          `json:"location"`
	Format   string          `json:"format"`
	Keywords json.RawMessage `json:"targets,omitempty"`
}

type Ext struct {
	CreativeId                string `json:"creativeId"`
	DealId                    string `json:"dealId"`
	Priority                  string `json:"priority"`
	OrderProperty             string `json:"orderProperty"`
	AdFormat                  string `json:"adFormat"`
	AdType                    string `json:"adType"`
	AdspaceId                 string `json:"adspaceId"`
	LibId                     string `json:"libId"`
	ViewableImpressionCounter string `json:"viewableImpressionCounter"`
}

type AdheseOrigin struct {
	Origin string `json:"origin"`
}

type AdheseBid struct {
	Origin                    string              `json:"origin"`
	OriginData                openrtb.BidResponse `json:"originData"`
	OriginInstance            string              `json:"originInstance,omitempty"`
	OrderProperty             string              `json:"orderProperty"`
	Ext                       string              `json:"ext"`
	Body                      string              `json:"body,omitempty"`
	Tag                       string              `json:"tag,omitempty"`
	SlotName                  string              `json:"slotName,omitempty"`
	SlotID                    string              `json:"slotID,omitempty"`
	AdspaceId                 string              `json:"adspaceId"`
	Priority                  string              `json:"priority"`
	LibId                     string              `json:"libId"`
	OrderId                   string              `json:"orderId"`
	AdType                    string              `json:"adType"`
	ImpressionCounter         string              `json:"impressionCounter"`
	ViewableImpressionCounter string              `json:"viewableImpressionCounter"`
	Tracker                   string              `json:"tracker"`
	Id                        string              `json:"id"`
	CreativeName              string              `json:"creativeName"`
	AdFormat                  string              `json:"adFormat"`
	Height                    string              `json:"height"`
	Width                     string              `json:"width"`
	Extension                 Prebid              `json:"extension"`
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
