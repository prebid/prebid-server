package openrtb_ext

import (
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

type ImpExtFreewheelSSP struct {
	ZoneId jsonutil.StringInt `json:"zoneId"`
}
