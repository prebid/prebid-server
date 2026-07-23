package openrtb_ext

import "encoding/json"

type ExtImpAkcelo struct {
	AdUnitID json.Number `json:"adUnitId,omitempty"`
	SiteID   json.Number `json:"siteId,omitempty"`
	Test     json.Number `json:"test,omitempty"`
}
