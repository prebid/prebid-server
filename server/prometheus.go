package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/logger"
	metricsconfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusServer(cfg *config.Configuration, metrics *metricsconfig.DetailedMetricsEngine) *http.Server {
	proMetrics := metrics.PrometheusMetrics

	if proMetrics == nil {
		logger.Fatalf("Prometheus metrics configured, but a Prometheus metrics engine was not found. Cannot set up a Prometheus listener.")
	}

	// Gather from the PBS core registry and, when present, the dedicated registry hook modules use
	// for their own collectors so both are exported on /metrics.
	var gatherer prometheus.Gatherer = proMetrics.Gatherer
	if metrics.ModuleMetricsGatherer != nil {
		gatherer = prometheus.Gatherers{proMetrics.Gatherer, metrics.ModuleMetricsGatherer}
	}

	return &http.Server{
		Addr: cfg.Host + ":" + strconv.Itoa(cfg.Metrics.Prometheus.Port),
		Handler: promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
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
