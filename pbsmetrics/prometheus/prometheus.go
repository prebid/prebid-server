package prometheusmetrics

import (
	"strconv"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics defines the Prometheus metrics backing the MetricsEngine implementation.
type Metrics struct {
	Registry *prometheus.Registry

	// General Metrics
	connectionsClosed            prometheus.Counter
	connectionsError             *prometheus.CounterVec
	connectionsOpened            prometheus.Counter
	cookieSync                   prometheus.Counter
	impressions                  *prometheus.CounterVec
	impressionsLegacy            prometheus.Counter
	prebidCacheWriteTimer        *prometheus.HistogramVec
	requests                     *prometheus.CounterVec
	requestsTimer                *prometheus.HistogramVec
	requestsWithoutCookie        *prometheus.CounterVec
	storedImpressionsCacheResult *prometheus.CounterVec
	storedRequestCacheResult     *prometheus.CounterVec

	// Adapter Metrics
	adapterBids          *prometheus.CounterVec
	adapterCookieSync    *prometheus.CounterVec
	adapterErrors        *prometheus.CounterVec
	adapterPanics        *prometheus.CounterVec
	adapterPrices        *prometheus.HistogramVec
	adapterRequests      *prometheus.CounterVec
	adapterRequestsTimer *prometheus.HistogramVec
	adapterUserSync      *prometheus.CounterVec

	// Account Metrics
	accountRequests *prometheus.CounterVec
}

const (
	accountLabel         = "account"
	actionLabel          = "action"
	adapterErrorLabel    = "adapter_error"
	adapterLabel         = "adapter"
	bidTypeLabel         = "bid_type"
	cacheResultLabel     = "cache_result"
	connectionErrorLabel = "connection_error"
	cookieLabel          = "cookie"
	hasBidsLabel         = "has_bids"
	isAudioLabel         = "audio"
	isBannerLabel        = "banner"
	isNativeLabel        = "native"
	isVideoLabel         = "video"
	markupDeliveryLabel  = "delivery"
	privacyBlockedLabel  = "privacy_blocked"
	requestStatusLabel   = "request_status"
	requestTypeLabel     = "request_type"
	successLabel         = "success"
)

const (
	connectionAcceptError = "accept"
	connectionCloseError  = "close"
)

const (
	markupDeliveryAdm  = "adm"
	markupDeliveryNurl = "nurl"
)

// NewMetrics initializes a new Prometheus metrics instance with preloaded label values.
func NewMetrics(cfg config.PrometheusMetrics) *Metrics {
	requestTimeBuckets := []float64{0.05, 0.1, 0.15, 0.20, 0.25, 0.3, 0.4, 0.5, 0.75, 1}
	cacheWriteTimeBuckts := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	priceBuckets := []float64{250, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}

	metrics := Metrics{}
	metrics.Registry = prometheus.NewRegistry()

	metrics.connectionsClosed = newCounterWithoutLabels(cfg, metrics.Registry,
		"connections_closed",
		"Count of successful connections closed to Prebid Server.")

	metrics.connectionsError = newCounter(cfg, metrics.Registry,
		"connections_error",
		"Count of errors for connection open and close attempts to Prebid Server labeled by type.",
		[]string{connectionErrorLabel})

	metrics.connectionsOpened = newCounterWithoutLabels(cfg, metrics.Registry,
		"connections_opened",
		"Count of successful connections opened to Prebid Server.")

	metrics.cookieSync = newCounterWithoutLabels(cfg, metrics.Registry,
		"cookie_sync_requests",
		"Count of cookie sync requests to Prebid Server.")

	metrics.impressions = newCounter(cfg, metrics.Registry,
		"impressions_requests",
		"Count of requested impressions to Prebid Server labeled by type.",
		[]string{isBannerLabel, isVideoLabel, isAudioLabel, isNativeLabel})

	metrics.impressionsLegacy = newCounterWithoutLabels(cfg, metrics.Registry,
		"impressions_requests_legacy",
		"Count of requested impressions to Prebid Server using the legacy endpoint.")

	metrics.prebidCacheWriteTimer = newHistogram(cfg, metrics.Registry,
		"prebidcache_write_time_seconds",
		"Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
		[]string{successLabel},
		cacheWriteTimeBuckts)

	metrics.requests = newCounter(cfg, metrics.Registry,
		"requests",
		"Count of total requests to Prebid Server labeled by type and status.",
		[]string{requestTypeLabel, requestStatusLabel})

	metrics.requestsTimer = newHistogram(cfg, metrics.Registry,
		"request_time_seconds",
		"Seconds to resolve successful Prebid Server requests labeled by type.",
		[]string{requestTypeLabel},
		requestTimeBuckets)

	metrics.requestsWithoutCookie = newCounter(cfg, metrics.Registry,
		"requests_without_cookie",
		"Count of total requests to Prebid Server without a cookie labeled by type.",
		[]string{requestTypeLabel})

	metrics.storedImpressionsCacheResult = newCounter(cfg, metrics.Registry,
		"stored_impressions_cache_performance",
		"Count of stored impression cache requests attempts by hits or miss.",
		[]string{cacheResultLabel})

	metrics.storedRequestCacheResult = newCounter(cfg, metrics.Registry,
		"stored_request_cache_performance",
		"Count of stored request cache requests attempts by hits or miss.",
		[]string{cacheResultLabel})

	metrics.adapterBids = newCounter(cfg, metrics.Registry,
		"adapter_bids",
		"Count of bids labeled by adapter and markup delivery type (adm or nurl).",
		[]string{adapterLabel, markupDeliveryLabel})

	metrics.adapterCookieSync = newCounter(cfg, metrics.Registry,
		"adapter_cookie_sync",
		"Count of cookie sync requests received labeled by adapter and if the sync was blocked due to privacy regulation (GDPR, CCPA, etc...).",
		[]string{adapterLabel, privacyBlockedLabel})

	metrics.adapterErrors = newCounter(cfg, metrics.Registry,
		"adapter_errors",
		"Count of errors labeled by adapter and error type.",
		[]string{adapterLabel, adapterErrorLabel})

	metrics.adapterPanics = newCounter(cfg, metrics.Registry,
		"adapter_panics",
		"Count of panics labeled by adapter.",
		[]string{adapterLabel})

	metrics.adapterPrices = newHistogram(cfg, metrics.Registry,
		"adapter_prices",
		"Monetary value of the bids labeled by adapter.",
		[]string{adapterLabel},
		priceBuckets)

	metrics.adapterRequests = newCounter(cfg, metrics.Registry,
		"adapter_requests",
		"Count of requests labeled by adapter, if has a cookie, and if it resulted in bids.",
		[]string{adapterLabel, cookieLabel, hasBidsLabel})

	metrics.adapterRequestsTimer = newHistogram(cfg, metrics.Registry,
		"adapter_request_time_seconds",
		"Seconds to resolve each successful request labeled by adapter.",
		[]string{adapterLabel},
		requestTimeBuckets)

	metrics.adapterUserSync = newCounter(cfg, metrics.Registry,
		"adapter_user_sync",
		"Count of user ID sync requests received labeled by adapter and action.",
		[]string{adapterLabel, actionLabel})

	metrics.accountRequests = newCounter(cfg, metrics.Registry,
		"account_requests",
		"Count of total requests to Prebid Server labeled by account.",
		[]string{accountLabel})

	preloadLabelValues(&metrics)

	return &metrics
}

func newCounter(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, labels []string) *prometheus.CounterVec {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	counter := prometheus.NewCounterVec(opts, labels)
	registry.MustRegister(counter)
	return counter
}

func newCounterWithoutLabels(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string) prometheus.Counter {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	counter := prometheus.NewCounter(opts)
	registry.MustRegister(counter)
	return counter
}

func newHistogram(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	histogram := prometheus.NewHistogramVec(opts, labels)
	registry.MustRegister(histogram)
	return histogram
}

func (m *Metrics) RecordConnectionAccept(success bool) {
	if success {
		m.connectionsOpened.Inc()
	} else {
		m.connectionsError.With(prometheus.Labels{
			connectionErrorLabel: connectionAcceptError,
		}).Inc()
	}
}

func (m *Metrics) RecordConnectionClose(success bool) {
	if success {
		m.connectionsClosed.Inc()
	} else {
		m.connectionsError.With(prometheus.Labels{
			connectionErrorLabel: connectionCloseError,
		}).Inc()
	}
}

func (m *Metrics) RecordRequest(labels pbsmetrics.Labels) {
	m.requests.With(prometheus.Labels{
		requestTypeLabel:   string(labels.RType),
		requestStatusLabel: string(labels.RequestStatus),
	}).Inc()

	if labels.CookieFlag == pbsmetrics.CookieFlagNo {
		m.requestsWithoutCookie.With(prometheus.Labels{
			requestTypeLabel: string(labels.RType),
		}).Inc()
	}

	if labels.PubID != pbsmetrics.PublisherUnknown {
		m.accountRequests.With(prometheus.Labels{
			accountLabel: labels.PubID,
		}).Inc()
	}
}

func (m *Metrics) RecordImps(labels pbsmetrics.ImpLabels) {
	m.impressions.With(prometheus.Labels{
		isBannerLabel: strconv.FormatBool(labels.BannerImps),
		isVideoLabel:  strconv.FormatBool(labels.VideoImps),
		isAudioLabel:  strconv.FormatBool(labels.AudioImps),
		isNativeLabel: strconv.FormatBool(labels.NativeImps),
	}).Inc()
}

func (m *Metrics) RecordLegacyImps(labels pbsmetrics.Labels, numImps int) {
	m.impressionsLegacy.Add(float64(numImps))
}

func (m *Metrics) RecordRequestTime(labels pbsmetrics.Labels, length time.Duration) {
	if labels.RequestStatus == pbsmetrics.RequestStatusOK {
		m.requestsTimer.With(prometheus.Labels{
			requestTypeLabel: string(labels.RType),
		}).Observe(length.Seconds())
	}
}

func (m *Metrics) RecordAdapterRequest(labels pbsmetrics.AdapterLabels) {
	m.adapterRequests.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
		cookieLabel:  string(labels.CookieFlag),
		hasBidsLabel: strconv.FormatBool(labels.AdapterBids == pbsmetrics.AdapterBidPresent),
	}).Inc()

	for err := range labels.AdapterErrors {
		m.adapterErrors.With(prometheus.Labels{
			adapterLabel:      string(labels.Adapter),
			adapterErrorLabel: string(err),
		}).Inc()
	}
}

func (m *Metrics) RecordAdapterPanic(labels pbsmetrics.AdapterLabels) {
	m.adapterPanics.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
	}).Inc()
}

func (m *Metrics) RecordAdapterBidReceived(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	markupDelivery := markupDeliveryNurl
	if hasAdm {
		markupDelivery = markupDeliveryAdm
	}

	m.adapterBids.With(prometheus.Labels{
		adapterLabel:        string(labels.Adapter),
		markupDeliveryLabel: markupDelivery,
	}).Inc()
}

func (m *Metrics) RecordAdapterPrice(labels pbsmetrics.AdapterLabels, cpm float64) {
	m.adapterPrices.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
	}).Observe(cpm)
}

func (m *Metrics) RecordAdapterTime(labels pbsmetrics.AdapterLabels, length time.Duration) {
	if len(labels.AdapterErrors) == 0 {
		m.adapterRequestsTimer.With(prometheus.Labels{
			adapterLabel: string(labels.Adapter),
		}).Observe(length.Seconds())
	}
}

func (m *Metrics) RecordCookieSync() {
	m.cookieSync.Inc()
}

func (m *Metrics) RecordAdapterCookieSync(adapter openrtb_ext.BidderName, privacyBlocked bool) {
	m.adapterCookieSync.With(prometheus.Labels{
		adapterLabel:        string(adapter),
		privacyBlockedLabel: strconv.FormatBool(privacyBlocked),
	}).Inc()
}

func (m *Metrics) RecordUserIDSet(labels pbsmetrics.UserLabels) {
	adapter := string(labels.Bidder)
	if adapter != "" {
		m.adapterUserSync.With(prometheus.Labels{
			adapterLabel: adapter,
			actionLabel:  string(labels.Action),
		}).Inc()
	}
}

func (m *Metrics) RecordStoredReqCacheResult(cacheResult pbsmetrics.CacheResult, inc int) {
	m.storedRequestCacheResult.With(prometheus.Labels{
		cacheResultLabel: string(cacheResult),
	}).Add(float64(inc))
}

func (m *Metrics) RecordStoredImpCacheResult(cacheResult pbsmetrics.CacheResult, inc int) {
	m.storedImpressionsCacheResult.With(prometheus.Labels{
		cacheResultLabel: string(cacheResult),
	}).Add(float64(inc))
}

func (m *Metrics) RecordPrebidCacheRequestTime(success bool, length time.Duration) {
	m.prebidCacheWriteTimer.With(prometheus.Labels{
		successLabel: strconv.FormatBool(success),
	}).Observe(length.Seconds())
}
