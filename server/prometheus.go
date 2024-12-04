package server

import (
	"net/http"
	"strconv"

	"github.com/prebid/prebid-server/v3/di"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prebid/prebid-server/v3/config"
	metricsconfig "github.com/prebid/prebid-server/v3/metrics/config"
)

func newPrometheusServer(cfg *config.Configuration, metrics *metricsconfig.DetailedMetricsEngine) *http.Server {
	proMetrics := metrics.PrometheusMetrics

	if proMetrics == nil {
		di.Log.Fatal("Prometheus metrics configured, but a Prometheus metrics engine was not found. Cannot set up a Prometheus listener.")
	}
	return &http.Server{
		Addr: cfg.Host + ":" + strconv.Itoa(cfg.Metrics.Prometheus.Port),
		Handler: promhttp.HandlerFor(proMetrics.Gatherer, promhttp.HandlerOpts{
			ErrorLog:            loggerForPrometheus{},
			MaxRequestsInFlight: 5,
			Timeout:             cfg.Metrics.Prometheus.Timeout(),
		}),
	}
}

type loggerForPrometheus struct{}

func (loggerForPrometheus) Println(v ...interface{}) {
	di.Log.Warningln(v...)
}
