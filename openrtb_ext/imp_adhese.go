package openrtb_ext

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb"
)

type ExtImpAdhese struct {
	Account  string                  `json:"account"`
	Location string                  `json:"location"`
	Format   string                  `json:"format"`
	Keywords []*AdheseKeywordsParams `json:"targets,omitempty"`
}

type ExtAdhese struct {
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

type AdheseKeywordsParams struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

type AdheseOrigin struct {
	Origin string `json:"origin"`
}

type AdheseOpenRtbBid struct {
	Origin            string              `json:"origin"`
	OriginInstance    string              `json:"originInstance"`
	OriginData        openrtb.BidResponse `json:"originData"`
	Ext               string              `json:"ext"`
	AdType            string              `json:"adType"`
	SlotName          string              `json:"slotName"`
	SlotID            string              `json:"slotID,omitempty"`
	Height            string              `json:"height"`
	Width             string              `json:"width"`
	Body              string              `json:"body"`
	ImpressionCounter string              `json:"impressionCounter"`
	Extension         Prebid              `json:"extension"`
}

type AdheseBid struct {
	Origin     string          `json:"origin"`
	OriginData json.RawMessage `json:"originData"`

	OrderProperty             string `json:"orderProperty"`
	Ext                       string `json:"ext"`
	Body                      string `json:"body,omitempty"`
	Tag                       string `json:"tag,omitempty"`
	AdspaceId                 string `json:"adspaceId"`
	Priority                  string `json:"priority"`
	LibId                     string `json:"libId"`
	OrderId                   string `json:"orderId"`
	AdType                    string `json:"adType"`
	ImpressionCounter         string `json:"impressionCounter"`
	ViewableImpressionCounter string `json:"viewableImpressionCounter"`
	Tracker                   string `json:"tracker"`
	Id                        string `json:"id"`
	CreativeName              string `json:"creativeName"`
	AdFormat                  string `json:"adFormat"`
	Height                    string `json:"height"`
	Width                     string `json:"width"`
	Extension                 Prebid `json:"extension"`
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
