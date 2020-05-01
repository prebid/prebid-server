package newrelic

import (
	"net/http"

	nr "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/prebid/prebid-server/config"
)

// Make ...
func Make(cfg config.NewRelic) (*nr.Application, error) {

	return nr.NewApplication(
		nr.ConfigAppName(cfg.AppName),
		nr.ConfigLicense(cfg.LicenseKey),
		nr.ConfigDistributedTracerEnabled(true),
		ConfigIgnoreStatusCodes([]int{http.StatusUnprocessableEntity, http.StatusBadGateway}),
	)
}
