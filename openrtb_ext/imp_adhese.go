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

type AdheseExt struct {
	Id                        string `json:"id"`
	OrderId                   string `json:"orderId"`
	Priority                  string `json:"priority"`
	OrderProperty             string `json:"orderProperty"`
	AdFormat                  string `json:"adFormat"`
	AdType                    string `json:"adType"`
	AdspaceId                 string `json:"adspaceId"`
	LibId                     string `json:"libId"`
	SlotID                    string `json:"slotID,omitempty"`
	SlotName                  string `json:"slotName,omitempty"`
	ImpressionCounter         string `json:"impressionCounter"`
	ViewableImpressionCounter string `json:"viewableImpressionCounter"`
	Tag                       string `json:"tag,omitempty"`
	Ext                       string `json:"ext"`
	CreativeName              string `json:"creativeName"`
	Tracker                   string `json:"tracker"`
}

type AdheseOrigin struct {
	Origin string `json:"origin"`
}

type AdheseBid struct {
	Origin                    string              `json:"origin"`
	OriginData                openrtb.BidResponse `json:"originData"`
	OriginInstance            string              `json:"originInstance,omitempty"`
	Body                      string              `json:"body,omitempty"`
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
