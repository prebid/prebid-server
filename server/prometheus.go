package server

import (
	"net/http"
	"reflect"
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbsmetrics"
	prometheusMetrics "github.com/prebid/prebid-server/pbsmetrics/prometheus"
)

func newPrometheusServer(cfg *config.Configuration, metrics pbsmetrics.MetricsEngine) *http.Server {
	vMetrics := reflect.ValueOf(metrics)

	var proMetrics *prometheusMetrics.Metrics
	if vMetrics.Kind() == reflect.Slice {
		// We need to iterate through the metrics engines available to find the Prometheus engine.
		for i := 0; i < vMetrics.Len(); i++ {
			proMetrics = resolvePrometheusMetrics(vMetrics.Index(i))
			if proMetrics != nil {
				break
			}
		}
	} else {
		proMetrics = resolvePrometheusMetrics(vMetrics)
	}

	if proMetrics == nil {
		glog.Fatal("Prometheus metrics configured, but a Prometheus metrics engine was not found. Cannot set up a Prometheus listener.")
	}
	return &http.Server{
		Addr:    cfg.Host + ":" + strconv.Itoa(cfg.Metrics.Prometheus.Port),
		Handler: promhttp.HandlerFor(proMetrics.Registry, promhttp.HandlerOpts{}),
	}
}

func resolvePrometheusMetrics(v reflect.Value) *prometheusMetrics.Metrics {
	var proMetrics *prometheusMetrics.Metrics
	if v.Type() == reflect.TypeOf(proMetrics) {
		return v.Interface().(*prometheusMetrics.Metrics)
	}
	return nil
}
