package teads

import (
	"encoding/json"
	"text/template"
)

type adapter struct {
	endpointTemplate *template.Template
}

type defaultBidderImpExtension struct {
	Bidder bidder `json:"bidder"`
}

type bidder struct {
	PlacementId int `json:"placementId"`
}

type teadsImpExtension struct {
	KV teadsKV `json:"kv"`
}

type teadsKV struct {
	PlacementId int `json:"placementId"`
}

type teadsBidExt struct {
	Prebid teadsPrebidExt `json:"prebid"`
}

type teadsPrebidExt struct {
	Meta teadsPrebidMeta `json:"meta"`
}

type teadsPrebidMeta struct {
	RendererName    string          `json:"rendererName"`
	RendererVersion string          `json:"rendererVersion"`
	RendererData    json.RawMessage `json:"rendererData"`
}
