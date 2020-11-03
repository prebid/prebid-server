package prometheusmetrics

import (
	"strconv"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics defines the Prometheus metrics backing the MetricsEngine implementation.
type Metrics struct {
	Registry *prometheus.Registry

	// General Metrics
	connectionsClosed             prometheus.Counter
	connectionsError              *prometheus.CounterVec
	connectionsOpened             prometheus.Counter
	cookieSync                    prometheus.Counter
	impressions                   *prometheus.CounterVec
	impressionsLegacy             prometheus.Counter
	prebidCacheWriteTimer         *prometheus.HistogramVec
	requests                      *prometheus.CounterVec
	requestsTimer                 *prometheus.HistogramVec
	requestsQueueTimer            *prometheus.HistogramVec
	requestsWithoutCookie         *prometheus.CounterVec
	storedImpressionsCacheResult  *prometheus.CounterVec
	storedRequestCacheResult      *prometheus.CounterVec
	timeout_notifications         *prometheus.CounterVec
	requestsDuplicateBidIDCounter prometheus.Counter // total request having duplicate bid.id for given bidder

	// Adapter Metrics
	adapterBids                  *prometheus.CounterVec
	adapterCookieSync            *prometheus.CounterVec
	adapterErrors                *prometheus.CounterVec
	adapterPanics                *prometheus.CounterVec
	adapterPrices                *prometheus.HistogramVec
	adapterRequests              *prometheus.CounterVec
	adapterRequestsTimer         *prometheus.HistogramVec
	adapterUserSync              *prometheus.CounterVec
	adapterDuplicateBidIDCounter *prometheus.CounterVec

	// Account Metrics
	accountRequests *prometheus.CounterVec

	// Ad Pod Metrics

	// podImpGenTimer indicates time taken by impression generator
	// algorithm to generate impressions for given ad pod request
	podImpGenTimer *prometheus.HistogramVec

	// podImpGenTimer indicates time taken by combination generator
	// algorithm to generate combination based on bid response and ad pod request
	podCombGenTimer *prometheus.HistogramVec

	// podCompExclTimer indicates time taken by compititve exclusion
	// algorithm to generate final pod response based on bid response and ad pod request
	podCompExclTimer *prometheus.HistogramVec
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

const (
	requestSuccessLabel = "requestAcceptedLabel"
	requestRejectLabel  = "requestRejectedLabel"
)

const (
	requestSuccessful = "ok"
	requestFailed     = "failed"
)

// pod specific constants
const (
	podAlgorithm         = "algorithm"
	podNoOfImpressions   = "no_of_impressions"
	podTotalCombinations = "total_combinations"
	podNoOfResponseBids  = "no_of_response_bids"
)

// NewMetrics initializes a new Prometheus metrics instance with preloaded label values.
func NewMetrics(cfg config.PrometheusMetrics) *Metrics {
	requestTimeBuckets := []float64{0.05, 0.1, 0.15, 0.20, 0.25, 0.3, 0.4, 0.5, 0.75, 1}
	cacheWriteTimeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	priceBuckets := []float64{250, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	queuedRequestTimeBuckets := []float64{0, 1, 5, 30, 60, 120, 180, 240, 300}

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
		cacheWriteTimeBuckets)

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

	metrics.timeout_notifications = newCounter(cfg, metrics.Registry,
		"timeout_notification",
		"Count of timeout notifications triggered, and if they were successfully sent.",
		[]string{successLabel})

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

	metrics.requestsQueueTimer = newHistogram(cfg, metrics.Registry,
		"request_queue_time",
		"Seconds request was waiting in queue",
		[]string{requestTypeLabel, requestStatusLabel},
		queuedRequestTimeBuckets)

	metrics.adapterDuplicateBidIDCounter = newCounter(cfg, metrics.Registry,
		"duplicate_bid_ids",
		"Number of collisions observed for given adaptor",
		[]string{adapterLabel})

	metrics.requestsDuplicateBidIDCounter = newCounterWithoutLabels(cfg, metrics.Registry,
		"requests_having_duplicate_bid_ids",
		"Count of number of request where bid collision is detected.")

	// adpod specific metrics
	metrics.podImpGenTimer = newHistogram(cfg, metrics.Registry,
		"impr_gen",
		"Time taken by Ad Pod Impression Generator in seconds", []string{podAlgorithm, podNoOfImpressions},
		// 200 µS, 250 µS, 275 µS, 300 µS
		//[]float64{0.000200000, 0.000250000, 0.000275000, 0.000300000})
		// 100 µS, 200 µS, 300 µS, 400 µS, 500 µS,  600 µS,
		[]float64{0.000100000, 0.000200000, 0.000300000, 0.000400000, 0.000500000, 0.000600000})

	metrics.podCombGenTimer = newHistogram(cfg, metrics.Registry,
		"comb_gen",
		"Time taken by Ad Pod Combination Generator in seconds", []string{podAlgorithm, podTotalCombinations},
		// 200 µS, 250 µS, 275 µS, 300 µS
		//[]float64{0.000200000, 0.000250000, 0.000275000, 0.000300000})
		[]float64{0.000100000, 0.000200000, 0.000300000, 0.000400000, 0.000500000, 0.000600000})

	metrics.podCompExclTimer = newHistogram(cfg, metrics.Registry,
		"comp_excl",
		"Time taken by Ad Pod Compititve Exclusion in seconds", []string{podAlgorithm, podNoOfResponseBids},
		// 200 µS, 250 µS, 275 µS, 300 µS
		//[]float64{0.000200000, 0.000250000, 0.000275000, 0.000300000})
		[]float64{0.000100000, 0.000200000, 0.000300000, 0.000400000, 0.000500000, 0.000600000})

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

func (m *Metrics) RecordRequestQueueTime(success bool, requestType pbsmetrics.RequestType, length time.Duration) {
	successLabelFormatted := requestRejectLabel
	if success {
		successLabelFormatted = requestSuccessLabel
	}
	m.requestsQueueTimer.With(prometheus.Labels{
		requestTypeLabel:   string(requestType),
		requestStatusLabel: successLabelFormatted,
	}).Observe(length.Seconds())
}

func (m *Metrics) RecordTimeoutNotice(success bool) {
	if success {
		m.timeout_notifications.With(prometheus.Labels{
			successLabel: requestSuccessful,
		}).Inc()
	} else {
		m.timeout_notifications.With(prometheus.Labels{
			successLabel: requestFailed,
		}).Inc()
	}
}

// RecordAdapterDuplicateBidID captures the  bid.ID collisions when adaptor
// gives the bid response with multiple bids containing  same bid.ID
// ensure collisions value is greater than 1. This function will not give any error
// if collisions = 1 is passed
func (m *Metrics) RecordAdapterDuplicateBidID(adaptor string, collisions int) {
	m.adapterDuplicateBidIDCounter.With(prometheus.Labels{
		adapterLabel: adaptor,
	}).Add(float64(collisions))
}

// RecordRequestHavingDuplicateBidID keeps count of request when duplicate bid.id is
// detected in partner's response
func (m *Metrics) RecordRequestHavingDuplicateBidID() {
	m.requestsDuplicateBidIDCounter.Inc()
}

// pod specific metrics

// recordAlgoTime is common method which handles algorithm time performance
func recordAlgoTime(timer *prometheus.HistogramVec, labels pbsmetrics.PodLabels, elapsedTime time.Duration) {

	pmLabels := prometheus.Labels{
		podAlgorithm: labels.AlgorithmName,
	}

	if labels.NoOfImpressions != nil {
		pmLabels[podNoOfImpressions] = strconv.Itoa(*labels.NoOfImpressions)
	}
	if labels.NoOfCombinations != nil {
		pmLabels[podTotalCombinations] = strconv.Itoa(*labels.NoOfCombinations)
	}
	if labels.NoOfResponseBids != nil {
		pmLabels[podNoOfResponseBids] = strconv.Itoa(*labels.NoOfResponseBids)
	}

	timer.With(pmLabels).Observe(elapsedTime.Seconds())
}

// RecordPodImpGenTime records number of impressions generated and time taken
// by underneath algorithm to generate them
func (m *Metrics) RecordPodImpGenTime(labels pbsmetrics.PodLabels, start time.Time) {
	elapsedTime := time.Since(start)
	recordAlgoTime(m.podImpGenTimer, labels, elapsedTime)
}

// RecordPodCombGenTime records number of combinations generated and time taken
// by underneath algorithm to generate them
func (m *Metrics) RecordPodCombGenTime(labels pbsmetrics.PodLabels, elapsedTime time.Duration) {
	recordAlgoTime(m.podCombGenTimer, labels, elapsedTime)
}

// RecordPodCompititveExclusionTime records number of combinations comsumed for forming
// final ad pod response and time taken by underneath algorithm to generate them
func (m *Metrics) RecordPodCompititveExclusionTime(labels pbsmetrics.PodLabels, elapsedTime time.Duration) {
	recordAlgoTime(m.podCompExclTimer, labels, elapsedTime)
}
