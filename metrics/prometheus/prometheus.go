package prometheusmetrics

import (
	"strconv"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
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
	requestsQueueTimer           *prometheus.HistogramVec
	requestsWithoutCookie        *prometheus.CounterVec
	storedImpressionsCacheResult *prometheus.CounterVec
	storedRequestCacheResult     *prometheus.CounterVec
	accountCacheResult           *prometheus.CounterVec
	storedAccountFetchTimer      *prometheus.HistogramVec
	storedAccountErrors          *prometheus.CounterVec
	storedAMPFetchTimer          *prometheus.HistogramVec
	storedAMPErrors              *prometheus.CounterVec
	storedCategoryFetchTimer     *prometheus.HistogramVec
	storedCategoryErrors         *prometheus.CounterVec
	storedRequestFetchTimer      *prometheus.HistogramVec
	storedRequestErrors          *prometheus.CounterVec
	storedVideoFetchTimer        *prometheus.HistogramVec
	storedVideoErrors            *prometheus.CounterVec
	timeoutNotifications         *prometheus.CounterVec
	dnsLookupTimer               prometheus.Histogram
	tlsHandhakeTimer             prometheus.Histogram
	privacyCCPA                  *prometheus.CounterVec
	privacyCOPPA                 *prometheus.CounterVec
	privacyLMT                   *prometheus.CounterVec
	privacyTCF                   *prometheus.CounterVec

	// Adapter Metrics
	adapterBids                *prometheus.CounterVec
	adapterCookieSync          *prometheus.CounterVec
	adapterErrors              *prometheus.CounterVec
	adapterPanics              *prometheus.CounterVec
	adapterPrices              *prometheus.HistogramVec
	adapterRequests            *prometheus.CounterVec
	adapterRequestsTimer       *prometheus.HistogramVec
	adapterUserSync            *prometheus.CounterVec
	adapterReusedConnections   *prometheus.CounterVec
	adapterCreatedConnections  *prometheus.CounterVec
	adapterConnectionWaitTime  *prometheus.HistogramVec
	adapterGDPRBlockedRequests *prometheus.CounterVec

	// Account Metrics
	accountRequests *prometheus.CounterVec

	metricsDisabled config.DisabledMetrics
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
	optOutLabel          = "opt_out"
	privacyBlockedLabel  = "privacy_blocked"
	requestStatusLabel   = "request_status"
	requestTypeLabel     = "request_type"
	successLabel         = "success"
	versionLabel         = "version"
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

const (
	sourceLabel   = "source"
	sourceRequest = "request"
)

const (
	storedDataFetchTypeLabel = "stored_data_fetch_type"
	storedDataErrorLabel     = "stored_data_error"
)

// NewMetrics initializes a new Prometheus metrics instance with preloaded label values.
func NewMetrics(cfg config.PrometheusMetrics, disabledMetrics config.DisabledMetrics) *Metrics {
	standardTimeBuckets := []float64{0.05, 0.1, 0.15, 0.20, 0.25, 0.3, 0.4, 0.5, 0.75, 1}
	cacheWriteTimeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	priceBuckets := []float64{250, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	queuedRequestTimeBuckets := []float64{0, 1, 5, 30, 60, 120, 180, 240, 300}

	metrics := Metrics{}
	metrics.Registry = prometheus.NewRegistry()
	metrics.metricsDisabled = disabledMetrics

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

	metrics.prebidCacheWriteTimer = newHistogramVec(cfg, metrics.Registry,
		"prebidcache_write_time_seconds",
		"Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
		[]string{successLabel},
		cacheWriteTimeBuckets)

	metrics.requests = newCounter(cfg, metrics.Registry,
		"requests",
		"Count of total requests to Prebid Server labeled by type and status.",
		[]string{requestTypeLabel, requestStatusLabel})

	metrics.requestsTimer = newHistogramVec(cfg, metrics.Registry,
		"request_time_seconds",
		"Seconds to resolve successful Prebid Server requests labeled by type.",
		[]string{requestTypeLabel},
		standardTimeBuckets)

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

	metrics.accountCacheResult = newCounter(cfg, metrics.Registry,
		"account_cache_performance",
		"Count of account cache lookups by hits or miss.",
		[]string{cacheResultLabel})

	metrics.storedAccountFetchTimer = newHistogramVec(cfg, metrics.Registry,
		"stored_account_fetch_time_seconds",
		"Seconds to fetch stored accounts labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedAccountErrors = newCounter(cfg, metrics.Registry,
		"stored_account_errors",
		"Count of stored account errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedAMPFetchTimer = newHistogramVec(cfg, metrics.Registry,
		"stored_amp_fetch_time_seconds",
		"Seconds to fetch stored AMP requests labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedAMPErrors = newCounter(cfg, metrics.Registry,
		"stored_amp_errors",
		"Count of stored AMP errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedCategoryFetchTimer = newHistogramVec(cfg, metrics.Registry,
		"stored_category_fetch_time_seconds",
		"Seconds to fetch stored categories labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedCategoryErrors = newCounter(cfg, metrics.Registry,
		"stored_category_errors",
		"Count of stored category errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedRequestFetchTimer = newHistogramVec(cfg, metrics.Registry,
		"stored_request_fetch_time_seconds",
		"Seconds to fetch stored requests labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedRequestErrors = newCounter(cfg, metrics.Registry,
		"stored_request_errors",
		"Count of stored request errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedVideoFetchTimer = newHistogramVec(cfg, metrics.Registry,
		"stored_video_fetch_time_seconds",
		"Seconds to fetch stored video labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedVideoErrors = newCounter(cfg, metrics.Registry,
		"stored_video_errors",
		"Count of stored video errors by error type",
		[]string{storedDataErrorLabel})

	metrics.timeoutNotifications = newCounter(cfg, metrics.Registry,
		"timeout_notification",
		"Count of timeout notifications triggered, and if they were successfully sent.",
		[]string{successLabel})

	metrics.dnsLookupTimer = newHistogram(cfg, metrics.Registry,
		"dns_lookup_time",
		"Seconds to resolve DNS",
		standardTimeBuckets)

	metrics.tlsHandhakeTimer = newHistogram(cfg, metrics.Registry,
		"tls_handshake_time",
		"Seconds to perform TLS Handshake",
		standardTimeBuckets)

	metrics.privacyCCPA = newCounter(cfg, metrics.Registry,
		"privacy_ccpa",
		"Count of total requests to Prebid Server where CCPA was provided by source and opt-out .",
		[]string{sourceLabel, optOutLabel})

	metrics.privacyCOPPA = newCounter(cfg, metrics.Registry,
		"privacy_coppa",
		"Count of total requests to Prebid Server where the COPPA flag was set by source",
		[]string{sourceLabel})

	metrics.privacyTCF = newCounter(cfg, metrics.Registry,
		"privacy_tcf",
		"Count of TCF versions for requests where GDPR was enforced by source and version.",
		[]string{versionLabel, sourceLabel})

	metrics.privacyLMT = newCounter(cfg, metrics.Registry,
		"privacy_lmt",
		"Count of total requests to Prebid Server where the LMT flag was set by source",
		[]string{sourceLabel})

	if !metrics.metricsDisabled.AdapterGDPRRequestBlocked {
		metrics.adapterGDPRBlockedRequests = newCounter(cfg, metrics.Registry,
			"adapter_gdpr_requests_blocked",
			"Count of total bidder requests blocked due to unsatisfied GDPR purpose 2 legal basis",
			[]string{adapterLabel})
	}

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

	metrics.adapterPrices = newHistogramVec(cfg, metrics.Registry,
		"adapter_prices",
		"Monetary value of the bids labeled by adapter.",
		[]string{adapterLabel},
		priceBuckets)

	metrics.adapterRequests = newCounter(cfg, metrics.Registry,
		"adapter_requests",
		"Count of requests labeled by adapter, if has a cookie, and if it resulted in bids.",
		[]string{adapterLabel, cookieLabel, hasBidsLabel})

	if !metrics.metricsDisabled.AdapterConnectionMetrics {
		metrics.adapterCreatedConnections = newCounter(cfg, metrics.Registry,
			"adapter_connection_created",
			"Count that keeps track of new connections when contacting adapter bidder endpoints.",
			[]string{adapterLabel})

		metrics.adapterReusedConnections = newCounter(cfg, metrics.Registry,
			"adapter_connection_reused",
			"Count that keeps track of reused connections when contacting adapter bidder endpoints.",
			[]string{adapterLabel})

		metrics.adapterConnectionWaitTime = newHistogramVec(cfg, metrics.Registry,
			"adapter_connection_wait",
			"Seconds from when the connection was requested until it is either created or reused",
			[]string{adapterLabel},
			standardTimeBuckets)
	}

	metrics.adapterRequestsTimer = newHistogramVec(cfg, metrics.Registry,
		"adapter_request_time_seconds",
		"Seconds to resolve each successful request labeled by adapter.",
		[]string{adapterLabel},
		standardTimeBuckets)

	metrics.adapterUserSync = newCounter(cfg, metrics.Registry,
		"adapter_user_sync",
		"Count of user ID sync requests received labeled by adapter and action.",
		[]string{adapterLabel, actionLabel})

	metrics.accountRequests = newCounter(cfg, metrics.Registry,
		"account_requests",
		"Count of total requests to Prebid Server labeled by account.",
		[]string{accountLabel})

	metrics.requestsQueueTimer = newHistogramVec(cfg, metrics.Registry,
		"request_queue_time",
		"Seconds request was waiting in queue",
		[]string{requestTypeLabel, requestStatusLabel},
		queuedRequestTimeBuckets)

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

func newHistogramVec(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
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

func newHistogram(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, buckets []float64) prometheus.Histogram {
	opts := prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	histogram := prometheus.NewHistogram(opts)
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

func (m *Metrics) RecordRequest(labels metrics.Labels) {
	m.requests.With(prometheus.Labels{
		requestTypeLabel:   string(labels.RType),
		requestStatusLabel: string(labels.RequestStatus),
	}).Inc()

	if labels.CookieFlag == metrics.CookieFlagNo {
		m.requestsWithoutCookie.With(prometheus.Labels{
			requestTypeLabel: string(labels.RType),
		}).Inc()
	}

	if labels.PubID != metrics.PublisherUnknown {
		m.accountRequests.With(prometheus.Labels{
			accountLabel: labels.PubID,
		}).Inc()
	}
}

func (m *Metrics) RecordImps(labels metrics.ImpLabels) {
	m.impressions.With(prometheus.Labels{
		isBannerLabel: strconv.FormatBool(labels.BannerImps),
		isVideoLabel:  strconv.FormatBool(labels.VideoImps),
		isAudioLabel:  strconv.FormatBool(labels.AudioImps),
		isNativeLabel: strconv.FormatBool(labels.NativeImps),
	}).Inc()
}

func (m *Metrics) RecordLegacyImps(labels metrics.Labels, numImps int) {
	m.impressionsLegacy.Add(float64(numImps))
}

func (m *Metrics) RecordRequestTime(labels metrics.Labels, length time.Duration) {
	if labels.RequestStatus == metrics.RequestStatusOK {
		m.requestsTimer.With(prometheus.Labels{
			requestTypeLabel: string(labels.RType),
		}).Observe(length.Seconds())
	}
}

func (m *Metrics) RecordStoredDataFetchTime(labels metrics.StoredDataLabels, length time.Duration) {
	switch labels.DataType {
	case metrics.AccountDataType:
		m.storedAccountFetchTimer.With(prometheus.Labels{
			storedDataFetchTypeLabel: string(labels.DataFetchType),
		}).Observe(length.Seconds())
	case metrics.AMPDataType:
		m.storedAMPFetchTimer.With(prometheus.Labels{
			storedDataFetchTypeLabel: string(labels.DataFetchType),
		}).Observe(length.Seconds())
	case metrics.CategoryDataType:
		m.storedCategoryFetchTimer.With(prometheus.Labels{
			storedDataFetchTypeLabel: string(labels.DataFetchType),
		}).Observe(length.Seconds())
	case metrics.RequestDataType:
		m.storedRequestFetchTimer.With(prometheus.Labels{
			storedDataFetchTypeLabel: string(labels.DataFetchType),
		}).Observe(length.Seconds())
	case metrics.VideoDataType:
		m.storedVideoFetchTimer.With(prometheus.Labels{
			storedDataFetchTypeLabel: string(labels.DataFetchType),
		}).Observe(length.Seconds())
	}
}

func (m *Metrics) RecordStoredDataError(labels metrics.StoredDataLabels) {
	switch labels.DataType {
	case metrics.AccountDataType:
		m.storedAccountErrors.With(prometheus.Labels{
			storedDataErrorLabel: string(labels.Error),
		}).Inc()
	case metrics.AMPDataType:
		m.storedAMPErrors.With(prometheus.Labels{
			storedDataErrorLabel: string(labels.Error),
		}).Inc()
	case metrics.CategoryDataType:
		m.storedCategoryErrors.With(prometheus.Labels{
			storedDataErrorLabel: string(labels.Error),
		}).Inc()
	case metrics.RequestDataType:
		m.storedRequestErrors.With(prometheus.Labels{
			storedDataErrorLabel: string(labels.Error),
		}).Inc()
	case metrics.VideoDataType:
		m.storedVideoErrors.With(prometheus.Labels{
			storedDataErrorLabel: string(labels.Error),
		}).Inc()
	}
}

func (m *Metrics) RecordAdapterRequest(labels metrics.AdapterLabels) {
	m.adapterRequests.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
		cookieLabel:  string(labels.CookieFlag),
		hasBidsLabel: strconv.FormatBool(labels.AdapterBids == metrics.AdapterBidPresent),
	}).Inc()

	for err := range labels.AdapterErrors {
		m.adapterErrors.With(prometheus.Labels{
			adapterLabel:      string(labels.Adapter),
			adapterErrorLabel: string(err),
		}).Inc()
	}
}

// Keeps track of created and reused connections to adapter bidders and the time from the
// connection request, to the connection creation, or reuse from the pool across all engines
func (m *Metrics) RecordAdapterConnections(adapterName openrtb_ext.BidderName, connWasReused bool, connWaitTime time.Duration) {
	if m.metricsDisabled.AdapterConnectionMetrics {
		return
	}

	if connWasReused {
		m.adapterReusedConnections.With(prometheus.Labels{
			adapterLabel: string(adapterName),
		}).Inc()
	} else {
		m.adapterCreatedConnections.With(prometheus.Labels{
			adapterLabel: string(adapterName),
		}).Inc()
	}

	m.adapterConnectionWaitTime.With(prometheus.Labels{
		adapterLabel: string(adapterName),
	}).Observe(connWaitTime.Seconds())
}

func (m *Metrics) RecordDNSTime(dnsLookupTime time.Duration) {
	m.dnsLookupTimer.Observe(dnsLookupTime.Seconds())
}

func (m *Metrics) RecordTLSHandshakeTime(tlsHandshakeTime time.Duration) {
	m.tlsHandhakeTimer.Observe(tlsHandshakeTime.Seconds())
}

func (m *Metrics) RecordAdapterPanic(labels metrics.AdapterLabels) {
	m.adapterPanics.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
	}).Inc()
}

func (m *Metrics) RecordAdapterBidReceived(labels metrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	markupDelivery := markupDeliveryNurl
	if hasAdm {
		markupDelivery = markupDeliveryAdm
	}

	m.adapterBids.With(prometheus.Labels{
		adapterLabel:        string(labels.Adapter),
		markupDeliveryLabel: markupDelivery,
	}).Inc()
}

func (m *Metrics) RecordAdapterPrice(labels metrics.AdapterLabels, cpm float64) {
	m.adapterPrices.With(prometheus.Labels{
		adapterLabel: string(labels.Adapter),
	}).Observe(cpm)
}

func (m *Metrics) RecordAdapterTime(labels metrics.AdapterLabels, length time.Duration) {
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

func (m *Metrics) RecordUserIDSet(labels metrics.UserLabels) {
	adapter := string(labels.Bidder)
	if adapter != "" {
		m.adapterUserSync.With(prometheus.Labels{
			adapterLabel: adapter,
			actionLabel:  string(labels.Action),
		}).Inc()
	}
}

func (m *Metrics) RecordStoredReqCacheResult(cacheResult metrics.CacheResult, inc int) {
	m.storedRequestCacheResult.With(prometheus.Labels{
		cacheResultLabel: string(cacheResult),
	}).Add(float64(inc))
}

func (m *Metrics) RecordStoredImpCacheResult(cacheResult metrics.CacheResult, inc int) {
	m.storedImpressionsCacheResult.With(prometheus.Labels{
		cacheResultLabel: string(cacheResult),
	}).Add(float64(inc))
}

func (m *Metrics) RecordAccountCacheResult(cacheResult metrics.CacheResult, inc int) {
	m.accountCacheResult.With(prometheus.Labels{
		cacheResultLabel: string(cacheResult),
	}).Add(float64(inc))
}

func (m *Metrics) RecordPrebidCacheRequestTime(success bool, length time.Duration) {
	m.prebidCacheWriteTimer.With(prometheus.Labels{
		successLabel: strconv.FormatBool(success),
	}).Observe(length.Seconds())
}

func (m *Metrics) RecordRequestQueueTime(success bool, requestType metrics.RequestType, length time.Duration) {
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
		m.timeoutNotifications.With(prometheus.Labels{
			successLabel: requestSuccessful,
		}).Inc()
	} else {
		m.timeoutNotifications.With(prometheus.Labels{
			successLabel: requestFailed,
		}).Inc()
	}
}

func (m *Metrics) RecordRequestPrivacy(privacy metrics.PrivacyLabels) {
	if privacy.CCPAProvided {
		m.privacyCCPA.With(prometheus.Labels{
			sourceLabel: sourceRequest,
			optOutLabel: strconv.FormatBool(privacy.CCPAEnforced),
		}).Inc()
	}

	if privacy.COPPAEnforced {
		m.privacyCOPPA.With(prometheus.Labels{
			sourceLabel: sourceRequest,
		}).Inc()
	}

	if privacy.GDPREnforced {
		m.privacyTCF.With(prometheus.Labels{
			versionLabel: string(privacy.GDPRTCFVersion),
			sourceLabel:  sourceRequest,
		}).Inc()
	}

	if privacy.LMTEnforced {
		m.privacyLMT.With(prometheus.Labels{
			sourceLabel: sourceRequest,
		}).Inc()
	}
}

func (m *Metrics) RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName) {
	if m.metricsDisabled.AdapterGDPRRequestBlocked {
		return
	}

	m.adapterGDPRBlockedRequests.With(prometheus.Labels{
		adapterLabel: string(adapterName),
	}).Inc()
}
