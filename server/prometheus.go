package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/logger"
	metricsconfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusServer(cfg *config.Configuration, metrics *metricsconfig.DetailedMetricsEngine) *http.Server {
	proMetrics := metrics.PrometheusMetrics

	if proMetrics == nil {
		logger.Fatalf("Prometheus metrics configured, but a Prometheus metrics engine was not found. Cannot set up a Prometheus listener.")
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
	logger.Warnf(fmt.Sprintln(v...))
}
