package openrtb_ext

import "github.com/prebid/prebid-server/v3/util/jsonutil"

type ExtImpConnectAd struct {
	NetworkID jsonutil.StringInt `json:"networkId"`
	SiteID    jsonutil.StringInt `json:"siteId"`
	Bidfloor  float64            `json:"bidfloor,omitempty"`
}
