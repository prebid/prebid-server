package adnuntius

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/util/timeutil"
)

type QueryString map[string]string
type adapter struct {
	time      timeutil.Time
	endpoint  string
	extraInfo string
}

type NativeRequest struct {
	Ortb json.RawMessage `json:"ortb,omitempty"`
}

type adnRequestAdunit struct {
	AuId          string        `json:"auId"`
	TargetId      string        `json:"targetId"`
	AdType        string        `json:"adType,omitempty"`
	NativeRequest NativeRequest `json:"nativeRequest,omitempty"`
	Dimensions    [][]int64     `json:"dimensions,omitempty"`
	MaxDeals      int           `json:"maxDeals,omitempty"`
}

type extDeviceAdnuntius struct {
	NoCookies bool `json:"noCookies,omitempty"`
}
type siteExt struct {
	Data interface{} `json:"data"`
}

type adnAdvertiser struct {
	LegalName string `json:"legalName,omitempty"`
	Name      string `json:"name,omitempty"`
}

type Ad struct {
	Bid struct {
		Amount   float64
		Currency string
	}
	NetBid struct {
		Amount float64
	}
	GrossBid struct {
		Amount float64
	}
	DealID            string `json:"dealId,omitempty"`
	AdId              string
	CreativeWidth     string
	CreativeHeight    string
	CreativeId        string
	LineItemId        string
	Html              string
	DestinationUrls   map[string]string
	AdvertiserDomains []string
	Advertiser        adnAdvertiser `json:"advertiser,omitempty"`
}

type AdUnit struct {
	AuId           string
	TargetId       string
	Html           string
	MatchedAdCount int
	ResponseId     string
	NativeJson     json.RawMessage `json:"nativeJson,omitempty"`
	Ads            []Ad
	Deals          []Ad `json:"deals,omitempty"`
}

type AdnResponse struct {
	AdUnits []AdUnit
}
type adnMetaData struct {
	Usi string `json:"usi,omitempty"`
}
type adnRequest struct {
	AdUnits   []adnRequestAdunit `json:"adUnits"`
	MetaData  adnMetaData        `json:"metaData,omitempty"`
	Context   string             `json:"context,omitempty"`
	KeyValues interface{}        `json:"kv,omitempty"`
}
