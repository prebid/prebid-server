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
	cookieSync                   prometheus.Counter
	impressions                  *prometheus.CounterVec
	impressionsLegacy            prometheus.Counter
	prebidCacheWriteTimer        *prometheus.HistogramVec
	requests                     *prometheus.CounterVec
	requestsTimer                *prometheus.HistogramVec
	requestsWithoutCookie        *prometheus.CounterVec
	storedRequestCacheResult     *prometheus.CounterVec
	storedImpressionsCacheResult *prometheus.CounterVec

	// Adapter Metrics
	adapterBids              *prometheus.CounterVec
	adapterConnectionsOpened *prometheus.CounterVec
	adapterConnectionsClosed *prometheus.CounterVec
	adapterConnectionsError  *prometheus.CounterVec
	adapterCookieSync        *prometheus.CounterVec
	adapterErrors            *prometheus.CounterVec
	adapterPanics            *prometheus.CounterVec
	adapterPrices            *prometheus.HistogramVec
	adapterRequests          *prometheus.CounterVec
	adapterRequestsTimer     *prometheus.HistogramVec
	adapterUserSync          *prometheus.CounterVec

	// Account Metrics
	accountRequests *prometheus.CounterVec
}

const (
	requestTypeLabel     = "request_type"
	requestStatusLabel   = "request_status"
	isBannerLabel        = "is_banner"
	isVideoLabel         = "is_video"
	isAudioLabel         = "is_audio"
	isNativeLabel        = "is_native"
	successLabel         = "success"
	cacheResultLabel     = "cache_result"
	adapterLabel         = "adapter"
	connectionErrorLabel = "connection_error"
	bidTypeLabel         = "bid_type"
	hasCookieLabel       = "has_cookie"
	hasBidsLabel         = "has_bids"
	accountLabel         = "account"
	adapterErrorLabel    = "adapter_error"
	privacyBlockedLabel  = "privacy_blocked"
	actionLabel          = "action"
)

// NewMetrics constructs the appropriate options for the Prometheus metrics. Needs to be fed the promethus config
// Its own function to keep the metric creation function cleaner.
func NewMetrics(cfg config.PrometheusMetrics) *Metrics {
	timerBuckets := prometheus.LinearBuckets(0.05, 0.05, 20)
	timerBuckets = append(timerBuckets, []float64{1.5, 2.0, 3.0, 5.0, 10.0, 50.0}...)

	timerBucketsQuickTasks := prometheus.LinearBuckets(0.005, 0.005, 20)
	timerBucketsQuickTasks = append([]float64{0.001, 0.0015, 0.003}, timerBucketsQuickTasks...)

	priceBuckets := []float64{0, 250, 500, 1000, 1500, 2000, 2500, 3000, 3500, 4000}

	metrics := Metrics{}
	metrics.Registry = prometheus.NewRegistry()

	metrics.cookieSync = newCounterWithoutLabels(cfg, metrics.Registry,
		"cookie_sync_requests",
		"Count of cookie sync requests to Prebid Server.")

	metrics.impressions = newCounter(cfg, metrics.Registry,
		"impressions_requested",
		"Count of requested impressions to Prebid Server labeled by type.",
		[]string{isBannerLabel, isVideoLabel, isAudioLabel, isNativeLabel})

	metrics.impressionsLegacy = newCounterWithoutLabels(cfg, metrics.Registry,
		"impressions_requested_legacy",
		"Count of requested impressions to Prebid Server using the legacy endpoint.")

	metrics.prebidCacheWriteTimer = newHistogram(cfg, metrics.Registry,
		"prebidcache_write_time_seconds",
		"Seconds to complete a write to Prebid Cache labeled by success or failure.",
		[]string{successLabel},
		timerBucketsQuickTasks)

	metrics.requests = newCounter(cfg, metrics.Registry,
		"requests",
		"Count of total requests to Prebid Server labeled by type and status.",
		[]string{requestTypeLabel, requestStatusLabel})

	metrics.requestsTimer = newHistogram(cfg, metrics.Registry,
		"request_time_seconds",
		"Seconds to resolve successful Prebid Server requests labeled by type.",
		[]string{requestTypeLabel},
		timerBuckets)

	metrics.requestsWithoutCookie = newCounter(cfg, metrics.Registry,
		"requests_without_cookie",
		"Count of total requests to Prebid Server without a cookie labeled by type.",
		[]string{requestTypeLabel})

	metrics.storedRequestCacheResult = newCounter(cfg, metrics.Registry,
		"stored_request_cache_performance",
		"Count of stored request cache requests attempts by hits or miss.",
		[]string{cacheResultLabel})

	metrics.storedImpressionsCacheResult = newCounter(cfg, metrics.Registry,
		"stored_impressions_cache_performance",
		"Count of stored impression cache requests attempts by hits or miss.",
		[]string{cacheResultLabel})

	metrics.adapterBids = newCounter(cfg, metrics.Registry,
		"adapter_bids",
		"Count of bids labeled by adapter, if a cookie is present, and if bids are present.",
		[]string{adapterLabel, hasCookieLabel, hasBidsLabel})

	metrics.adapterConnectionsOpened = newCounter(cfg, metrics.Registry,
		"adapter_connections_opened",
		"Count of connections successfully opened labeled by adapter.",
		[]string{adapterLabel})

	metrics.adapterConnectionsClosed = newCounter(cfg, metrics.Registry,
		"adapter_connections_closed",
		"Count of connections successfully closed labeled by adapter.",
		[]string{adapterLabel})

	metrics.adapterConnectionsError = newCounter(cfg, metrics.Registry,
		"adapter_connections_error",
		"Count of errors with connections labeled by adapter and error type.",
		[]string{adapterLabel, connectionErrorLabel})

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
		[]string{adapterLabel, adapterErrorLabel})

	metrics.adapterPrices = newHistogram(cfg, metrics.Registry,
		"adapter_prices",
		"Monetary value of the bids labeled by adapter.",
		[]string{adapterLabel},
		priceBuckets)

	metrics.adapterRequests = newCounter(cfg, metrics.Registry,
		"adapter_requests",
		"Count of requests labeled by adapter, type, and status.",
		[]string{adapterLabel, requestTypeLabel, requestStatusLabel})

	metrics.adapterRequestsTimer = newHistogram(cfg, metrics.Registry,
		"adapter_request_time_seconds",
		"Seconds to resolve each successful request labeled by adapter.",
		[]string{adapterLabel, requestTypeLabel},
		timerBuckets)

	metrics.adapterUserSync = newCounter(cfg, metrics.Registry,
		"adapter_uset_sync",
		"Count of user ID sync requests received labeled by adapter and action.",
		[]string{adapterLabel, actionLabel})

	metrics.accountRequests = newCounter(cfg, metrics.Registry,
		"account_requests",
		"Count of total requests to Prebid Server labeled by account.",
		[]string{accountLabel})

	initializeTimeSeries(&metrics)

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

// init labels

func (m *Metrics) RecordConnectionAccept(adapter openrtb_ext.BidderName, success bool) {
	if success {
		m.adapterConnectionsOpened.With(prometheus.Labels{
			adapterLabel: string(adapter),
		}).Inc()
	} else {
		m.adapterConnectionsError.With(prometheus.Labels{
			adapterLabel:         string(adapter),
			connectionErrorLabel: "accept_error",
		}).Inc()
	}
}

func (m *Metrics) RecordConnectionClose(adapter openrtb_ext.BidderName, success bool) {
	if success {
		m.adapterConnectionsClosed.With(prometheus.Labels{
			adapterLabel: string(adapter),
		}).Inc()
	} else {
		m.adapterConnectionsError.With(prometheus.Labels{
			adapterLabel:         string(adapter),
			connectionErrorLabel: "close_error",
		}).Inc()
	}
}

func (m *Metrics) RecordRequest(labels pbsmetrics.Labels) {
	m.requests.With(prometheus.Labels{
		requestTypeLabel:   string(labels.RType),
		requestStatusLabel: string(labels.RequestStatus),
	}).Inc()

	if labels.PubID != pbsmetrics.PublisherUnknown {
		m.accountRequests.With(prometheus.Labels{
			accountLabel: labels.PubID,
		}).Inc()
	}

	if labels.CookieFlag == pbsmetrics.CookieFlagNo {
		m.requestsWithoutCookie.With(prometheus.Labels{
			requestTypeLabel: string(labels.RType),
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
	timeInSeconds := float64(length) / float64(time.Second)
	m.requestsTimer.With(prometheus.Labels{
		requestTypeLabel: string(labels.RType),
	}).Observe(timeInSeconds)
}

// todo: can have multiple errors per request. this is not represented correctly.
func (me *Metrics) RecordAdapterRequest(labels pbsmetrics.AdapterLabels) {
	me.adaptRequests.With(resolveAdapterLabels(labels)).Inc()
	for k := range labels.AdapterErrors {
		me.adaptErrors.With(resolveAdapterErrorLabels(labels, string(k))).Inc()
	}
}

func (m *Metrics) RecordAdapterPanic(labels pbsmetrics.AdapterLabels) {
	m.adapterPanics.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
	}).Inc()
}

// todo: also an issue. does cookie or bid received make sense here?
func (me *Metrics) RecordAdapterBidReceived(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	m.adapterBids.With(prometheus.Labels{
		adapterLabel:   string(labels.Adapter),
		hasCookieLabel: strconv.FormatBool(labels.CookieFlag != pbsmetrics.CookieFlagNo),
		hasBidsLabel:   strconv.FormatBool(labels.AdapterBids == pbsmetrics.AdapterBidPresent),
	}).Inc()
}

func (m *Metrics) RecordAdapterPrice(labels pbsmetrics.AdapterLabels, cpm float64) {
	m.adapterPrices.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
	}).Observe(cpm)
}

func (m *Metrics) RecordAdapterTime(labels pbsmetrics.AdapterLabels, length time.Duration) {
	if len(labels.AdapterErrors) == 0 {
		timeInSeconds := float64(length) / float64(time.Second)
		m.adapterRequestsTimer.With(prometheus.Labels{
			adapterLabel:     string(labels.Adapter),
			requestTypeLabel: string(labels.RType),
		}).Observe(timeInSeconds)
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
	m.adapterUserSync.With(prometheus.Labels{
		adapterLabel: string(labels.Bidder),
		actionLabel:  string(labels.Action),
	}).Inc()
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
	timeInSeconds := float64(length) / float64(time.Second)
	m.prebidCacheWriteTimer.With(prometheus.Labels{
		successLabel: strconv.FormatBool(success),
	}).Observe(timeInSeconds)
}

// initializeTimeSeries precreates all possible metric label values, so there is no locking needed at run time creating new instances
func initializeTimeSeries(m *Metrics) {
	// Connection errors
	labels := addDimension([]prometheus.Labels{}, "ErrorType", []string{"accept_error", "close_error"})
	for _, l := range labels {
		_ = m.connError.With(l)
	}

	// Standard labels
	labels = addDimension([]prometheus.Labels{}, demandSourceLabel, demandTypesAsString())
	labels = addDimension(labels, requestTypeLabel, requestTypesAsString())
	labels = addDimension(labels, browserLabel, browserTypesAsString())
	labels = addDimension(labels, cookieLabel, cookieTypesAsString())
	adapterLabels := labels // save regenerating these dimensions for adapter status
	labels = addDimension(labels, responseStatusLabel, requestStatusesAsString())
	// If we implement an account whitelist, we can seed the metrics with that list to redusce latency associated with registering new lable values on the fly.
	labels = addDimension(labels, accountLabel, []string{pbsmetrics.PublisherUnknown})
	for _, l := range labels {
		_ = m.requests.With(l)
		_ = m.reqTimer.With(l)
	}

	// Adapter labels
	labels = addDimension(adapterLabels, adapterLabel, adaptersAsString())
	errorLabels := labels // save regenerating these dimensions for adapter errors
	labels = addDimension(labels, adapterBidLabel, adapterBidsAsString())
	for _, l := range labels {
		_ = m.adaptRequests.With(l)
		_ = m.adaptTimer.With(l)
		_ = m.adaptPrices.With(l)
		_ = m.adaptPanics.With(l)
	}

	// AdapterBid labels
	labels = addDimension(labels, bidTypeLabel, bidTypesAsString())
	labels = addDimension(labels, markupTypeLabel, []string{"unknown", "adm"})
	for _, l := range labels {
		_ = m.adaptBids.With(l)
	}
	labels = addDimension(errorLabels, adapterErrLabel, adapterErrorsAsString())
	for _, l := range labels {
		_ = m.adaptErrors.With(l)
	}
	cookieLabels := addDimension([]prometheus.Labels{}, adapterLabel, adaptersAsString())
	cookieLabels = addDimension(cookieLabels, gdprBlockedLabel, []string{"true", "false"})
	for _, l := range cookieLabels {
		_ = m.adaptCookieSync.With(l)
	}
	cacheLabels := addDimension([]prometheus.Labels{}, "cache_result", cacheResultAsString())
	for _, l := range cacheLabels {
		_ = m.storedImpCacheResult.With(l)
		_ = m.storedReqCacheResult.With(l)
	}

	// ImpType labels
	impTypeLabels := addDimension([]prometheus.Labels{}, bannerLabel, []string{"yes", "no"})
	impTypeLabels = addDimension(impTypeLabels, videoLabel, []string{"yes", "no"})
	impTypeLabels = addDimension(impTypeLabels, audioLabel, []string{"yes", "no"})
	impTypeLabels = addDimension(impTypeLabels, nativeLabel, []string{"yes", "no"})
	for _, l := range impTypeLabels {
		_ = m.imps.With(l)
	}
}

// addDimesion will expand a slice of labels to add the dimension of a new set of values for a new label name
func addDimension(labels []prometheus.Labels, field string, values []string) []prometheus.Labels {
	if len(labels) == 0 {
		// We are starting a new slice of labels, so we can't loop.
		return addToLabel(make(prometheus.Labels), field, values)
	}
	newLabels := make([]prometheus.Labels, 0, len(labels)*len(values))
	for _, l := range labels {
		newLabels = append(newLabels, addToLabel(l, field, values)...)
	}
	return newLabels
}

// addToLabel will create a slice of labels adding a set of values tied to a label name.
func addToLabel(label prometheus.Labels, field string, values []string) []prometheus.Labels {
	newLabels := make([]prometheus.Labels, len(values))
	for i, v := range values {
		l := copyLabel(label)
		l[field] = v
		newLabels[i] = l
	}
	return newLabels
}

// Need to be able to deep copy prometheus labels.
func copyLabel(label prometheus.Labels) prometheus.Labels {
	newLabel := make(prometheus.Labels)
	for k, v := range label {
		newLabel[k] = v
	}
	return newLabel
}

func demandTypesAsString() []string {
	list := pbsmetrics.DemandTypes()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func requestTypesAsString() []string {
	list := pbsmetrics.RequestTypes()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func browserTypesAsString() []string {
	list := pbsmetrics.BrowserTypes()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func cookieTypesAsString() []string {
	list := pbsmetrics.CookieTypes()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func requestStatusesAsString() []string {
	list := pbsmetrics.RequestStatuses()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func adapterBidsAsString() []string {
	list := pbsmetrics.AdapterBids()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func adapterErrorsAsString() []string {
	list := pbsmetrics.AdapterErrors()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func cacheResultAsString() []string {
	list := pbsmetrics.CacheResults()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output
}

func adaptersAsString() []string {
	list := openrtb_ext.BidderList()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output

}

func bidTypesAsString() []string {
	list := openrtb_ext.BidTypes()
	output := make([]string, len(list))
	for i, s := range list {
		output[i] = string(s)
	}
	return output

}
