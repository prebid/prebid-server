package teads

import (
	"encoding/json"
	"text/template"
)

type adapter struct {
	endpointTemplate *template.Template
}

type DefaultBidderImpExtension struct {
	Bidder Bidder `json:"bidder"`
}

type Bidder struct {
	PlacementId int `json:"placementId"`
}

type TeadsImpExtension struct {
	KV TeadsKV `json:"kv"`
}

type TeadsKV struct {
	PlacementId int `json:"placementId"`
}

type TeadsBidExt struct {
	Prebid TeadsPrebidExt `json:"prebid"`
}

type TeadsPrebidExt struct {
	Meta TeadsPrebidMeta `json:"meta"`
}

type TeadsPrebidMeta struct {
	RendererName    string          `json:"rendererName"`
	RendererVersion string          `json:"rendererVersion"`
	RendererData    json.RawMessage `json:"rendererData"`
}
