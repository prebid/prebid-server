package models

import "encoding/json"

const (
	ExtEDSKey         = "eds"
	PubmaticBidderKey = "pubmatic"
	EDSDeviceKey      = "device"
	EDSAppKey         = "app"

	// EDSBlockedCountriesAll disables EDS params for all countries when used as feature value.
	EDSBlockedCountriesAll = "*"
)

// DeviceEDSTier1BlockedParams are stripped from bidderparams.pubmatic.eds.device for EDS blocked countries.
var DeviceEDSTier1BlockedParams = []string{
	"boottime",
	"diskspace",
	"totaldisk",
	"inputlanguage",
	"totalmem",
}

// AppEDSTier1BlockedParams are stripped from bidderparams.pubmatic.eds.app for EDS blocked countries.
var AppEDSTier1BlockedParams = []string{
	"install_time",
	"first_launch_time",
}

// ResolvedEds holds flattened PubMatic-only enrichment parameters carried in
// ext.prebid.bidderparams.{pubmatic}.eds until the PubMatic adapter merges them.
type ResolvedEds struct {
	Device json.RawMessage `json:"device,omitempty"`
	App    json.RawMessage `json:"app,omitempty"`
}

func (r ResolvedEds) IsEmpty() bool {
	return len(r.Device) == 0 && len(r.App) == 0
}
