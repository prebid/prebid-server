package openrtb_ext

import (
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type ImpExtGoldbach struct {
	PublisherID     string                                      `json:"publisherId"`
	SlotID          string                                      `json:"slotId"`
	CustomTargeting map[string]jsonutil.ItemOrItemArray[string] `json:"customTargeting,omitempty"`
}
