package prometheusmetrics

import (
	"strconv"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Defines the actual Prometheus metrics we will be using. Satisfies interface MetricsEngine
type Metrics struct {
	Registry      *prometheus.Registry
	connCounter   prometheus.Gauge
	connError     *prometheus.CounterVec
	imps          *prometheus.CounterVec
	requests      *prometheus.CounterVec
	reqTimer      *prometheus.HistogramVec
	adaptRequests *prometheus.CounterVec
	adaptTimer    *prometheus.HistogramVec
	adaptBids     *prometheus.CounterVec
	adaptPrices   *prometheus.HistogramVec
	cookieSync    prometheus.Counter
	userID        *prometheus.CounterVec
}

// NewMetrics constructs the appropriate options for the Prometheus metrics. Needs to be fed the promethus config
// Its own function to keep the metric creation function cleaner.
func NewMetrics(cfg config.PrometheusMetrics) *Metrics {
	// define the buckets for timers
	timerBuckets := prometheus.LinearBuckets(0.05, 0.05, 20)
	timerBuckets = append(timerBuckets, []float64{1.5, 2.0, 3.0, 5.0, 10.0, 50.0}...)

	standardLabelNames := []string{"demand_source", "request_type", "browser", "cookie", "response_status"}

	adapterLabelNames := []string{"demand_source", "request_type", "browser", "cookie", "response_status", "adapter"}
	bidLabelNames := []string{"demand_source", "request_type", "browser", "cookie", "response_status", "adapter", "bidtype", "hasadm"}

	metrics := Metrics{}
	metrics.Registry = prometheus.NewRegistry()
	metrics.connCounter = newConnCounter(cfg)
	metrics.Registry.MustRegister(metrics.connCounter)
	metrics.connError = newCounter(cfg, "active_connections_total",
		"Errors reported on the connections coming in.",
		[]string{"ErrorType"},
	)
	metrics.Registry.MustRegister(metrics.connError)
	metrics.imps = newCounter(cfg, "imps_requested_total",
		"Total number of impressions requested through PBS.",
		standardLabelNames,
	)
	metrics.Registry.MustRegister(metrics.imps)
	metrics.requests = newCounter(cfg, "requests_total",
		"Total number of requests made to PBS.",
		standardLabelNames,
	)
	metrics.Registry.MustRegister(metrics.requests)
	metrics.reqTimer = newHistogram(cfg, "request_time_seconds",
		"Seconds to resolve each PBS request.",
		standardLabelNames, timerBuckets,
	)
	metrics.Registry.MustRegister(metrics.reqTimer)
	metrics.adaptRequests = newCounter(cfg, "adapter_requests_total",
		"Number of requests sent out to each bidder.",
		adapterLabelNames,
	)
	metrics.Registry.MustRegister(metrics.adaptRequests)
	metrics.adaptTimer = newHistogram(cfg, "adapter_time_seconds",
		"Seconds to resolve each request to a bidder.",
		adapterLabelNames, timerBuckets,
	)
	metrics.Registry.MustRegister(metrics.adaptTimer)
	metrics.adaptBids = newCounter(cfg, "adapter_bids_recieved_total",
		"Number of bids recieved from each bidder.",
		bidLabelNames,
	)
	metrics.Registry.MustRegister(metrics.adaptBids)
	metrics.adaptPrices = newHistogram(cfg, "adapter_prices",
		"Value of the highest bids from each bidder.",
		adapterLabelNames, prometheus.LinearBuckets(0.1, 0.1, 200),
	)
	metrics.Registry.MustRegister(metrics.adaptPrices)
	metrics.cookieSync = newCookieSync(cfg)
	metrics.Registry.MustRegister(metrics.cookieSync)
	metrics.userID = newCounter(cfg, "usersync_total",
		"Number of user ID syncs performed",
		[]string{"action", "bidder"},
	)
	metrics.Registry.MustRegister(metrics.userID)

	return &metrics
}

func newConnCounter(cfg config.PrometheusMetrics) prometheus.Gauge {
	opts := prometheus.GaugeOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "active_connections",
		Help:      "Current number of active (open) connections.",
	}
	return prometheus.NewGauge(opts)
}

func newCookieSync(cfg config.PrometheusMetrics) prometheus.Counter {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      "cookie_sync_requests_total",
		Help:      "Number of cookie sync requests recieved.",
	}
	return prometheus.NewCounter(opts)
}

func newCounter(cfg config.PrometheusMetrics, name string, help string, labels []string) *prometheus.CounterVec {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	return prometheus.NewCounterVec(opts, labels)
}

func newHistogram(cfg config.PrometheusMetrics, name string, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	return prometheus.NewHistogramVec(opts, labels)
}

func (me *Metrics) RecordConnectionAccept(success bool) {
	if success {
		me.connCounter.Inc()
	} else {
		me.connError.WithLabelValues("accept_error").Inc()
	}

}

func (me *Metrics) RecordConnectionClose(success bool) {
	if success {
		me.connCounter.Dec()
	} else {
		me.connError.WithLabelValues("close_error").Inc()
	}
}

func (me *Metrics) RecordRequest(labels pbsmetrics.Labels) {
	me.requests.With(resolveLabels(labels)).Inc()
}

func (me *Metrics) RecordImps(labels pbsmetrics.Labels, numImps int) {
	me.imps.With(resolveLabels(labels)).Add(float64(numImps))
}

func (me *Metrics) RecordRequestTime(labels pbsmetrics.Labels, length time.Duration) {
	time := float64(length) / float64(time.Second)
	me.reqTimer.With(resolveLabels(labels)).Observe(time)
}

func (me *Metrics) RecordAdapterRequest(labels pbsmetrics.AdapterLabels) {
	me.adaptRequests.With(resolveAdapterLabels(labels)).Inc()
}

func (me *Metrics) RecordAdapterBidReceived(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	me.adaptBids.With(resolveBidLabels(labels, bidType, hasAdm)).Inc()
}

func (me *Metrics) RecordAdapterPrice(labels pbsmetrics.AdapterLabels, cpm float64) {
	me.adaptPrices.With(resolveAdapterLabels(labels)).Observe(cpm)
}

func (me *Metrics) RecordAdapterTime(labels pbsmetrics.AdapterLabels, length time.Duration) {
	time := float64(length) / float64(time.Second)
	me.adaptTimer.With(resolveAdapterLabels(labels)).Observe(time)
}

func (me *Metrics) RecordCookieSync(labels pbsmetrics.Labels) {
	me.cookieSync.Inc()
}

func (me *Metrics) RecordUserIDSet(userLabels pbsmetrics.UserLabels) {
	me.userID.With(resolveUserSyncLabels(userLabels)).Inc()
}

func resolveLabels(labels pbsmetrics.Labels) prometheus.Labels {
	return prometheus.Labels{
		"demand_source": string(labels.Source),
		"request_type":  string(labels.RType),
		// "pubid":   labels.PubID,
		"browser":         string(labels.Browser),
		"cookie":          string(labels.CookieFlag),
		"response_status": string(labels.RequestStatus),
	}
}

func resolveAdapterLabels(labels pbsmetrics.AdapterLabels) prometheus.Labels {
	return prometheus.Labels{
		"demand_source": string(labels.Source),
		"request_type":  string(labels.RType),
		// "pubid":   labels.PubID,
		"browser":         string(labels.Browser),
		"cookie":          string(labels.CookieFlag),
		"response_status": string(labels.AdapterStatus),
		"adapter":         string(labels.Adapter),
	}
}

func resolveBidLabels(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) prometheus.Labels {
	bidLabels := prometheus.Labels{
		"demand_source": string(labels.Source),
		"request_type":  string(labels.RType),
		// "pubid":   labels.PubID,
		"browser":         string(labels.Browser),
		"cookie":          string(labels.CookieFlag),
		"response_status": string(labels.AdapterStatus),
		"adapter":         string(labels.Adapter),
		"bidtype":         string(bidType),
		"hasadm":          strconv.FormatBool(hasAdm),
	}
	return bidLabels
}

func resolveUserSyncLabels(userLabels pbsmetrics.UserLabels) prometheus.Labels {
	return prometheus.Labels{
		"action": string(userLabels.Action),
		"bidder": string(userLabels.Bidder),
	}
}
