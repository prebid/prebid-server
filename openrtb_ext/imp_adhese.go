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
	Dm                        string                     `json:"dm"`
	AdType                    string                     `json:"adType"`
	AdFormat                  string                     `json:"adFormat"`
	TimeStamp                 string                     `json:"timeStamp"`
	Share                     string                     `json:"share"`
	Priority                  string                     `json:"priority"`
	OrderId                   string                     `json:"orderId"`
	AdspaceId                 string                     `json:"adspaceId"`
	AdspaceKey                string                     `json:"adspaceKey"`
	Body                      string                     `json:"body,omitempty"`
	TrackingUrl               string                     `json:"trackingUrl"`
	Tracker                   string                     `json:"tracker"`
	ExtraField1               string                     `json:"extraField1"`
	ExtraField2               string                     `json:"extraField2"`
	AltText                   string                     `json:"altText"`
	Height                    string                     `json:"height"`
	Width                     string                     `json:"width"`
	Tag                       string                     `json:"tag,omitempty"`
	TagUrl                    string                     `json:"tagUrl"`
	HeightLarge               string                     `json:"heightLarge"`
	WidthLarge                string                     `json:"widthLarge"`
	LibId                     string                     `json:"libId"`
	Id                        string                     `json:"id"`
	AdvertiserId              string                     `json:"advertiserId"`
	OrderProperty             string                     `json:"orderProperty"`
	Ext                       string                     `json:"ext"`
	SwfSrc                    string                     `json:"swfSrc"`
	Url                       string                     `json:"url"`
	ClickTag                  string                     `json:"clickTag"`
	SwfSrc2nd                 string                     `json:"swfSrc2nd"`
	SwfSrc3rd                 string                     `json:"swfSrc3rd"`
	SwfSrc4th                 string                     `json:"swfSrc4th"`
	PoolPath                  string                     `json:"poolPath"`
	Comment                   string                     `json:"comment"`
	AdDuration                string                     `json:"adDuration"`
	AdDuration2nd             string                     `json:"adDuration2nd"`
	AdDuration3rd             string                     `json:"adDuration3rd"`
	AdDuration4th             string                     `json:"adDuration4th"`
	OrderName                 string                     `json:"orderName"`
	CreativeName              string                     `json:"creativeName"`
	DeliveryMultiples         string                     `json:"deliveryMultiples"`
	DeliveryGroupId           string                     `json:"deliveryGroupId"`
	AdspaceStart              string                     `json:"adspaceStart"`
	AdspaceEnd                string                     `json:"adspaceEnd"`
	SwfSrc5th                 string                     `json:"swfSrc5th"`
	SwfSrc6th                 string                     `json:"swfSrc6th"`
	AdDuration5th             string                     `json:"adDuration5th"`
	AdDuration6th             string                     `json:"adDuration6th"`
	Width3rd                  string                     `json:"width3rd"`
	Width4th                  string                     `json:"width4th"`
	Width5th                  string                     `json:"width5th"`
	Width6th                  string                     `json:"width6th"`
	Height3rd                 string                     `json:"height3rd"`
	Height4th                 string                     `json:"height4th"`
	Height5th                 string                     `json:"height5th"`
	Height6th                 string                     `json:"height6th"`
	SlotName                  string                     `json:"slotName"`
	SlotID                    string                     `json:"slotID"`
	ImpressionCounter         string                     `json:"impressionCounter"`
	TrackedImpressionCounter  string                     `json:"trackedImpressionCounter"`
	ViewableImpressionCounter string                     `json:"viewableImpressionCounter"`
	AdditionalCreatives       []AdheseAdditionalCreative `json:"additionalCreatives"`
	Origin                    string                     `json:"origin"`
	OriginData                json.RawMessage            `json:"originData"`
	Auctionable               string                     `json:"auctionable"`
	AdditionalViewableTracker string                     `json:"additionalViewableTracker"`
	Extension                 Prebid                     `json:"extension"`
}

type AdheseAdditionalCreative struct {
	Dm                        string          `json:"dm"`
	AdType                    string          `json:"adType"`
	AdFormat                  string          `json:"adFormat"`
	TimeStamp                 string          `json:"timeStamp"`
	Share                     string          `json:"share"`
	Priority                  string          `json:"priority"`
	OrderId                   string          `json:"orderId"`
	AdspaceId                 string          `json:"adspaceId"`
	AdspaceKey                string          `json:"adspaceKey"`
	Body                      string          `json:"body,omitempty"`
	TrackingUrl               string          `json:"trackingUrl"`
	Tracker                   string          `json:"tracker"`
	ExtraField1               string          `json:"extraField1"`
	ExtraField2               string          `json:"extraField2"`
	AltText                   string          `json:"altText"`
	Height                    string          `json:"height"`
	Width                     string          `json:"width"`
	Tag                       string          `json:"tag,omitempty"`
	TagUrl                    string          `json:"tagUrl"`
	HeightLarge               string          `json:"heightLarge"`
	WidthLarge                string          `json:"widthLarge"`
	LibId                     string          `json:"libId"`
	Id                        string          `json:"id"`
	AdvertiserId              string          `json:"advertiserId"`
	OrderProperty             string          `json:"orderProperty"`
	Ext                       string          `json:"ext"`
	SwfSrc                    string          `json:"swfSrc"`
	Url                       string          `json:"url"`
	ClickTag                  string          `json:"clickTag"`
	SwfSrc2nd                 string          `json:"swfSrc2nd"`
	SwfSrc3rd                 string          `json:"swfSrc3rd"`
	SwfSrc4th                 string          `json:"swfSrc4th"`
	PoolPath                  string          `json:"poolPath"`
	Comment                   string          `json:"comment"`
	AdDuration                string          `json:"adDuration"`
	AdDuration2nd             string          `json:"adDuration2nd"`
	AdDuration3rd             string          `json:"adDuration3rd"`
	AdDuration4th             string          `json:"adDuration4th"`
	OrderName                 string          `json:"orderName"`
	CreativeName              string          `json:"creativeName"`
	DeliveryMultiples         string          `json:"deliveryMultiples"`
	DeliveryGroupId           string          `json:"deliveryGroupId"`
	AdspaceStart              string          `json:"adspaceStart"`
	AdspaceEnd                string          `json:"adspaceEnd"`
	SwfSrc5th                 string          `json:"swfSrc5th"`
	SwfSrc6th                 string          `json:"swfSrc6th"`
	AdDuration5th             string          `json:"adDuration5th"`
	AdDuration6th             string          `json:"adDuration6th"`
	Width3rd                  string          `json:"width3rd"`
	Width4th                  string          `json:"width4th"`
	Width5th                  string          `json:"width5th"`
	Width6th                  string          `json:"width6th"`
	Height3rd                 string          `json:"height3rd"`
	Height4th                 string          `json:"height4th"`
	Height5th                 string          `json:"height5th"`
	Height6th                 string          `json:"height6th"`
	SlotName                  string          `json:"slotName"`
	SlotID                    string          `json:"slotID"`
	ImpressionCounter         string          `json:"impressionCounter"`
	TrackedImpressionCounter  string          `json:"trackedImpressionCounter"`
	ViewableImpressionCounter string          `json:"viewableImpressionCounter"`
	AdditionalCreatives       string          `json:"additionalCreatives"`
	Origin                    string          `json:"origin"`
	OriginData                json.RawMessage `json:"originData"`
	Auctionable               string          `json:"auctionable"`
	AdditionalViewableTracker string          `json:"additionalViewableTracker"`
	Extension                 Prebid          `json:"extension"`
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
