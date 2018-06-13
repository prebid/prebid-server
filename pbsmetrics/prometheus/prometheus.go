package prometheusmetrics

import (
	"time"

	_ "github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define the options for prometheus metrics vectors we need to support prometheus.
type opts struct {
	connCounter   prometheus.GaugeOpts
	connError     prometheus.CounterOpts
	imps          prometheus.CounterOpts
	requests      prometheus.HistogramOpts
	reqTimer      prometheus.HistogramOpts
	adaptRequests prometheus.HistogramOpts
	adaptTimer    prometheus.HistogramOpts
	cookieSync    prometheus.HistogramOpts
	userID        prometheus.HistogramOpts
}

// Defines the actual Prometheus metrics we will be using. Satisfies interface MetricsEngine
type Metrics struct {
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

	standardLabelNames := []string{"source", "type", "pubid", "browser", "cookie", "status"}

	adapterLabelNames := []string{"source", "type", "pubid", "browser", "cookie", "status", "adapter"}
	bidLabelNames := append([]string{"bidtype", "hasadm"}, adapterLabelNames...)

	metrics := Metrics{}
	metrics.connCounter = newConnCounter(cfg)
	prometheus.MustRegister(metrics.connCounter)
	metrics.connError = newCounter(cfg, "active_connections_total",
		"Errors reported on the connections coming in.",
		[]string{"ErrorType"},
	)
	prometheus.MustRegister(metrics.connError)
	metrics.imps = newCounter(cfg, "imps_requested_total",
		"Total number of impressions requested through PBS.",
		standardLabelNames,
	)
	prometheus.MustRegister(metrics.imps)
	metrics.requests = newCounter(cfg, "requests_total",
		"Total number of requests made to PBS.",
		standardLabelNames,
	)
	prometheus.MustRegister(metrics.requests)
	metrics.reqTimer = newHistogram(cfg, "request_time_seconds",
		"Seconds to resolve each PBS request.",
		standardLabelNames, timerBuckets,
	)
	prometheus.MustRegister(metrics.reqTimer)
	metrics.adaptRequests = newCounter(cfg, "adapter_requests_total",
		"Number of requests sent out to each bidder.",
		adapterLabelNames,
	)
	prometheus.MustRegister(metrics.adaptRequests)
	metrics.adaptTimer = newHistogram(cfg, "adapter_time_seconds",
		"Seconds to resolve each request to a bidder.",
		adapterLabelNames, timerBuckets,
	)
	prometheus.MustRegister(metrics.adaptTimer)
	metrics.adaptBids = newCounter(cfg, "adapter_bids_recieved_total",
		"Number of bids recieved from each bidder.",
		adapterLabelNames,
	)
	prometheus.MustRegister(metrics.adaptBids)
	metrics.adaptPrices = newHistogram(cfg, "adapter_prices",
		"Value of the highest bids from each bidder.",
		bidLabelNames, prometheus.LinearBuckets(0.1, 0.1, 200),
	)
	prometheus.MustRegister(metrics.adaptPrices)
	metrics.cookieSync = newCookieSync(cfg)
	prometheus.MustRegister(metrics.cookieSync)
	metrics.userID = newCounter(cfg, "usersync_total",
		"Number of user ID syncs performed",
		[]string{"action", "bidder"},
	)
	prometheus.MustRegister(metrics.userID)

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

func (me *Metrics) RecordAdapterBidsReceived(labels pbsmetrics.AdapterLabels, bids int64) {
	me.adaptBids.With(resolveAdapterLabels(labels)).Add(float64(bids))
}

func (me *Metrics) RecordAdapterBidAdm(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
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
		"source":  string(labels.Source),
		"type":    string(labels.RType),
		"pubid":   labels.PubID,
		"browser": string(labels.Browser),
		"cookie":  string(labels.CookieFlag),
		"status":  string(labels.RequestStatus),
	}
}

func resolveAdapterLabels(labels pbsmetrics.AdapterLabels) prometheus.Labels {
	return prometheus.Labels{
		"source":  string(labels.Source),
		"type":    string(labels.RType),
		"pubid":   labels.PubID,
		"browser": string(labels.Browser),
		"cookie":  string(labels.CookieFlag),
		"status":  string(labels.AdapterStatus),
		"adapter": string(labels.Adapter),
	}
}

func resolveBidLabels(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) prometheus.Labels {
	bidLabels := prometheus.Labels{
		"source":  string(labels.Source),
		"type":    string(labels.RType),
		"pubid":   labels.PubID,
		"browser": string(labels.Browser),
		"cookie":  string(labels.CookieFlag),
		"status":  string(labels.AdapterStatus),
		"adapter": string(labels.Adapter),
		"bidtype": string(bidType),
	}
	if hasAdm {
		bidLabels["hasadm"] = "true"
	}
	return bidLabels
}

func resolveUserSyncLabels(userLabels pbsmetrics.UserLabels) prometheus.Labels {
	return prometheus.Labels{
		"action": string(userLabels.Action),
		"bidder": string(userLabels.Bidder),
	}
}
