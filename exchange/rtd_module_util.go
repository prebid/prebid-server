package exchange

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/rtdmodule"
	"net/http"
)

func BuildRtdModules(client *http.Client, cfg *config.Configuration) rtdmodule.RtdProcessor {
	return buildRtdModules(client, cfg)
}

func buildRtdModules(client *http.Client, cfg *config.Configuration) *rtdmodule.RtdModules {
	// build modules here
	return &rtdmodule.RtdModules{}
}
