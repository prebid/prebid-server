package newrelic

import (
	"context"
	"net/http"

	"github.com/golang/glog"

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
		ConfigTransactionTracerEnabled(false),
	)
}

// NoticeError ...
func NoticeError(ctx context.Context, err error) {
	// get newrelic transaction from context
	if ctx == nil {
		glog.Warningf("Context is nil, could not notice error: %v", err)
		return
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		glog.Warningf("Newrelic transaction is nil, could not notice error: %v", err)
		return
	}

	txn.NoticeError(err)
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
