package openrtb_ext

import "github.com/prebid/prebid-server/v3/util/jsonutil"

type ImpExtNativo struct {
	PlacementID jsonutil.StringInt `json:"placementId"`
}
