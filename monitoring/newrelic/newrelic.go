package newrelic

import (
	"net/http"

	"github.com/newrelic/go-agent/v3/integrations/nrlogrus"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/prebid/prebid-server/config"
	"github.com/sirupsen/logrus"
)

// Make ...
func Make(cfg config.NewRelic) (*newrelic.Application, error) {
	l, err := getLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	return newrelic.NewApplication(
		newrelic.ConfigAppName(cfg.AppName),
		newrelic.ConfigLicense(cfg.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		nrlogrus.ConfigLogger(l),
		ConfigIgnoreStatusCodes([]int{http.StatusUnprocessableEntity, http.StatusBadGateway}),
	)
}

func getLogger(logLevel string) (*logrus.Logger, error) {
	l := logrus.New()

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}

	l.SetLevel(level)

	return l, nil
}
