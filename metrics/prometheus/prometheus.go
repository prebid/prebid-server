package prometheusmetrics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prometheus/client_golang/prometheus"
	promCollector "github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics defines the Prometheus metrics backing the MetricsEngine implementation.
type Metrics struct {
	Registerer prometheus.Registerer
	Gatherer   *prometheus.Registry

	// General Metrics
	tmaxTimeout                  prometheus.Counter
	connectionsClosed            prometheus.Counter
	connectionsError             *prometheus.CounterVec
	connectionsOpened            prometheus.Counter
	cookieSync                   *prometheus.CounterVec
	setUid                       *prometheus.CounterVec
	impressions                  *prometheus.CounterVec
	prebidCacheWriteTimer        *prometheus.HistogramVec
	requests                     *prometheus.CounterVec
	debugRequests                prometheus.Counter
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
	storedResponses              prometheus.Counter
	storedResponsesFetchTimer    *prometheus.HistogramVec
	storedResponsesErrors        *prometheus.CounterVec
	adsCertRequests              *prometheus.CounterVec
	adsCertSignTimer             prometheus.Histogram
	bidderServerResponseTimer    prometheus.Histogram

	// Adapter Metrics
	adapterBids                           *prometheus.CounterVec
	adapterErrors                         *prometheus.CounterVec
	adapterPanics                         *prometheus.CounterVec
	adapterPrices                         *prometheus.HistogramVec
	adapterRequests                       *prometheus.CounterVec
	overheadTimer                         *prometheus.HistogramVec
	adapterRequestsTimer                  *prometheus.HistogramVec
	adapterReusedConnections              *prometheus.CounterVec
	adapterCreatedConnections             *prometheus.CounterVec
	adapterConnectionWaitTime             *prometheus.HistogramVec
	adapterScrubbedBuyerUIDs              *prometheus.CounterVec
	adapterGDPRBlockedRequests            *prometheus.CounterVec
	adapterBidResponseValidationSizeError *prometheus.CounterVec
	adapterBidResponseValidationSizeWarn  *prometheus.CounterVec
	adapterBidResponseSecureMarkupError   *prometheus.CounterVec
	adapterBidResponseSecureMarkupWarn    *prometheus.CounterVec

	// Syncer Metrics
	syncerRequests *prometheus.CounterVec
	syncerSets     *prometheus.CounterVec

	// Account Metrics
	accountRequests                       *prometheus.CounterVec
	accountDebugRequests                  *prometheus.CounterVec
	accountStoredResponses                *prometheus.CounterVec
	accountBidResponseValidationSizeError *prometheus.CounterVec
	accountBidResponseValidationSizeWarn  *prometheus.CounterVec
	accountBidResponseSecureMarkupError   *prometheus.CounterVec
	accountBidResponseSecureMarkupWarn    *prometheus.CounterVec

	// Module Metrics as a map where the key is the module name
	moduleDuration        map[string]*prometheus.HistogramVec
	moduleCalls           map[string]*prometheus.CounterVec
	moduleFailures        map[string]*prometheus.CounterVec
	moduleSuccessNoops    map[string]*prometheus.CounterVec
	moduleSuccessUpdates  map[string]*prometheus.CounterVec
	moduleSuccessRejects  map[string]*prometheus.CounterVec
	moduleExecutionErrors map[string]*prometheus.CounterVec
	moduleTimeouts        map[string]*prometheus.CounterVec

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
	overheadTypeLabel    = "overhead_type"
	privacyBlockedLabel  = "privacy_blocked"
	requestStatusLabel   = "request_status"
	requestTypeLabel     = "request_type"
	stageLabel           = "stage"
	statusLabel          = "status"
	successLabel         = "success"
	syncerLabel          = "syncer"
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
func NewMetrics(cfg config.PrometheusMetrics, disabledMetrics config.DisabledMetrics, syncerKeys []string, moduleStageNames map[string][]string) *Metrics {
	standardTimeBuckets := []float64{0.05, 0.1, 0.15, 0.20, 0.25, 0.3, 0.4, 0.5, 0.75, 1}
	cacheWriteTimeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	priceBuckets := []float64{250, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	queuedRequestTimeBuckets := []float64{0, 1, 5, 30, 60, 120, 180, 240, 300}
	overheadTimeBuckets := []float64{0.05, 0.06, 0.07, 0.08, 0.09, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}

	metrics := Metrics{}
	reg := prometheus.NewRegistry()
	metrics.metricsDisabled = disabledMetrics

	metrics.connectionsClosed = newCounterWithoutLabels(cfg, reg,
		"connections_closed",
		"Count of successful connections closed to Prebid Server.")

	metrics.connectionsError = newCounter(cfg, reg,
		"connections_error",
		"Count of errors for connection open and close attempts to Prebid Server labeled by type.",
		[]string{connectionErrorLabel})

	metrics.connectionsOpened = newCounterWithoutLabels(cfg, reg,
		"connections_opened",
		"Count of successful connections opened to Prebid Server.")

	metrics.tmaxTimeout = newCounterWithoutLabels(cfg, reg,
		"tmax_timeout",
		"Count of requests rejected due to Tmax timeout exceed.")

	metrics.cookieSync = newCounter(cfg, reg,
		"cookie_sync_requests",
		"Count of cookie sync requests to Prebid Server.",
		[]string{statusLabel})

	metrics.setUid = newCounter(cfg, reg,
		"setuid_requests",
		"Count of set uid requests to Prebid Server.",
		[]string{statusLabel})

	metrics.impressions = newCounter(cfg, reg,
		"impressions_requests",
		"Count of requested impressions to Prebid Server labeled by type.",
		[]string{isBannerLabel, isVideoLabel, isAudioLabel, isNativeLabel})

	metrics.prebidCacheWriteTimer = newHistogramVec(cfg, reg,
		"prebidcache_write_time_seconds",
		"Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
		[]string{successLabel},
		cacheWriteTimeBuckets)

	metrics.requests = newCounter(cfg, reg,
		"requests",
		"Count of total requests to Prebid Server labeled by type and status.",
		[]string{requestTypeLabel, requestStatusLabel})

	metrics.debugRequests = newCounterWithoutLabels(cfg, reg,
		"debug_requests",
		"Count of total requests to Prebid Server that have debug enabled")

	metrics.requestsTimer = newHistogramVec(cfg, reg,
		"request_time_seconds",
		"Seconds to resolve successful Prebid Server requests labeled by type.",
		[]string{requestTypeLabel},
		standardTimeBuckets)

	metrics.requestsWithoutCookie = newCounter(cfg, reg,
		"requests_without_cookie",
		"Count of total requests to Prebid Server without a cookie labeled by type.",
		[]string{requestTypeLabel})

	metrics.storedImpressionsCacheResult = newCounter(cfg, reg,
		"stored_impressions_cache_performance",
		"Count of stored impression cache requests attempts by hits or miss.",
		[]string{cacheResultLabel})

	metrics.storedRequestCacheResult = newCounter(cfg, reg,
		"stored_request_cache_performance",
		"Count of stored request cache requests attempts by hits or miss.",
		[]string{cacheResultLabel})

	metrics.accountCacheResult = newCounter(cfg, reg,
		"account_cache_performance",
		"Count of account cache lookups by hits or miss.",
		[]string{cacheResultLabel})

	metrics.storedAccountFetchTimer = newHistogramVec(cfg, reg,
		"stored_account_fetch_time_seconds",
		"Seconds to fetch stored accounts labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedAccountErrors = newCounter(cfg, reg,
		"stored_account_errors",
		"Count of stored account errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedAMPFetchTimer = newHistogramVec(cfg, reg,
		"stored_amp_fetch_time_seconds",
		"Seconds to fetch stored AMP requests labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedAMPErrors = newCounter(cfg, reg,
		"stored_amp_errors",
		"Count of stored AMP errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedCategoryFetchTimer = newHistogramVec(cfg, reg,
		"stored_category_fetch_time_seconds",
		"Seconds to fetch stored categories labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedCategoryErrors = newCounter(cfg, reg,
		"stored_category_errors",
		"Count of stored category errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedRequestFetchTimer = newHistogramVec(cfg, reg,
		"stored_request_fetch_time_seconds",
		"Seconds to fetch stored requests labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedRequestErrors = newCounter(cfg, reg,
		"stored_request_errors",
		"Count of stored request errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedVideoFetchTimer = newHistogramVec(cfg, reg,
		"stored_video_fetch_time_seconds",
		"Seconds to fetch stored video labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedVideoErrors = newCounter(cfg, reg,
		"stored_video_errors",
		"Count of stored video errors by error type",
		[]string{storedDataErrorLabel})

	metrics.timeoutNotifications = newCounter(cfg, reg,
		"timeout_notification",
		"Count of timeout notifications triggered, and if they were successfully sent.",
		[]string{successLabel})

	metrics.dnsLookupTimer = newHistogram(cfg, reg,
		"dns_lookup_time",
		"Seconds to resolve DNS",
		standardTimeBuckets)

	metrics.tlsHandhakeTimer = newHistogram(cfg, reg,
		"tls_handshake_time",
		"Seconds to perform TLS Handshake",
		standardTimeBuckets)

	metrics.privacyCCPA = newCounter(cfg, reg,
		"privacy_ccpa",
		"Count of total requests to Prebid Server where CCPA was provided by source and opt-out .",
		[]string{sourceLabel, optOutLabel})

	metrics.privacyCOPPA = newCounter(cfg, reg,
		"privacy_coppa",
		"Count of total requests to Prebid Server where the COPPA flag was set by source",
		[]string{sourceLabel})

	metrics.privacyTCF = newCounter(cfg, reg,
		"privacy_tcf",
		"Count of TCF versions for requests where GDPR was enforced by source and version.",
		[]string{versionLabel, sourceLabel})

	metrics.privacyLMT = newCounter(cfg, reg,
		"privacy_lmt",
		"Count of total requests to Prebid Server where the LMT flag was set by source",
		[]string{sourceLabel})

	if !metrics.metricsDisabled.AdapterBuyerUIDScrubbed {
		metrics.adapterScrubbedBuyerUIDs = newCounter(cfg, reg,
			"adapter_buyeruids_scrubbed",
			"Count of total bidder requests with a scrubbed buyeruid due to a privacy policy",
			[]string{adapterLabel})
	}
	if !metrics.metricsDisabled.AdapterGDPRRequestBlocked {
		metrics.adapterGDPRBlockedRequests = newCounter(cfg, reg,
			"adapter_gdpr_requests_blocked",
			"Count of total bidder requests blocked due to unsatisfied GDPR purpose 2 legal basis",
			[]string{adapterLabel})
	}

	metrics.storedResponsesFetchTimer = newHistogramVec(cfg, reg,
		"stored_response_fetch_time_seconds",
		"Seconds to fetch stored responses labeled by fetch type",
		[]string{storedDataFetchTypeLabel},
		standardTimeBuckets)

	metrics.storedResponsesErrors = newCounter(cfg, reg,
		"stored_response_errors",
		"Count of stored video errors by error type",
		[]string{storedDataErrorLabel})

	metrics.storedResponses = newCounterWithoutLabels(cfg, reg,
		"stored_responses",
		"Count of total requests to Prebid Server that have stored responses")

	metrics.adapterBids = newCounter(cfg, reg,
		"adapter_bids",
		"Count of bids labeled by adapter and markup delivery type (adm or nurl).",
		[]string{adapterLabel, markupDeliveryLabel})

	metrics.adapterErrors = newCounter(cfg, reg,
		"adapter_errors",
		"Count of errors labeled by adapter and error type.",
		[]string{adapterLabel, adapterErrorLabel})

	metrics.adapterPanics = newCounter(cfg, reg,
		"adapter_panics",
		"Count of panics labeled by adapter.",
		[]string{adapterLabel})

	metrics.adapterPrices = newHistogramVec(cfg, reg,
		"adapter_prices",
		"Monetary value of the bids labeled by adapter.",
		[]string{adapterLabel},
		priceBuckets)

	metrics.adapterRequests = newCounter(cfg, reg,
		"adapter_requests",
		"Count of requests labeled by adapter, if has a cookie, and if it resulted in bids.",
		[]string{adapterLabel, cookieLabel, hasBidsLabel})

	if !metrics.metricsDisabled.AdapterConnectionMetrics {
		metrics.adapterCreatedConnections = newCounter(cfg, reg,
			"adapter_connection_created",
			"Count that keeps track of new connections when contacting adapter bidder endpoints.",
			[]string{adapterLabel})

		metrics.adapterReusedConnections = newCounter(cfg, reg,
			"adapter_connection_reused",
			"Count that keeps track of reused connections when contacting adapter bidder endpoints.",
			[]string{adapterLabel})

		metrics.adapterConnectionWaitTime = newHistogramVec(cfg, reg,
			"adapter_connection_wait",
			"Seconds from when the connection was requested until it is either created or reused",
			[]string{adapterLabel},
			standardTimeBuckets)
	}

	metrics.adapterBidResponseValidationSizeError = newCounter(cfg, reg,
		"adapter_response_validation_size_err",
		"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight",
		[]string{adapterLabel, successLabel})

	metrics.adapterBidResponseValidationSizeWarn = newCounter(cfg, reg,
		"adapter_response_validation_size_warn",
		"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight (warn)",
		[]string{adapterLabel, successLabel})

	metrics.adapterBidResponseSecureMarkupError = newCounter(cfg, reg,
		"adapter_response_validation_secure_err",
		"Count that tracks number of bids removed from bid response that had a invalid bidAdm",
		[]string{adapterLabel, successLabel})

	metrics.adapterBidResponseSecureMarkupWarn = newCounter(cfg, reg,
		"adapter_response_validation_secure_warn",
		"Count that tracks number of bids removed from bid response that had a invalid bidAdm (warn)",
		[]string{adapterLabel, successLabel})

	metrics.overheadTimer = newHistogramVec(cfg, reg,
		"overhead_time_seconds",
		"Seconds to prepare adapter request or resolve adapter response",
		[]string{overheadTypeLabel},
		overheadTimeBuckets)

	metrics.adapterRequestsTimer = newHistogramVec(cfg, reg,
		"adapter_request_time_seconds",
		"Seconds to resolve each successful request labeled by adapter.",
		[]string{adapterLabel},
		standardTimeBuckets)

	metrics.bidderServerResponseTimer = newHistogram(cfg, reg,
		"bidder_server_response_time_seconds",
		"Duration needed to send HTTP request and receive response back from bidder server.",
		standardTimeBuckets)

	metrics.syncerRequests = newCounter(cfg, reg,
		"syncer_requests",
		"Count of cookie sync requests where a syncer is a candidate to be synced labeled by syncer key and status.",
		[]string{syncerLabel, statusLabel})

	metrics.syncerSets = newCounter(cfg, reg,
		"syncer_sets",
		"Count of setuid set requests for a syncer labeled by syncer key and status.",
		[]string{syncerLabel, statusLabel})

	metrics.accountRequests = newCounter(cfg, reg,
		"account_requests",
		"Count of total requests to Prebid Server labeled by account.",
		[]string{accountLabel})

	metrics.accountDebugRequests = newCounter(cfg, reg,
		"account_debug_requests",
		"Count of total requests to Prebid Server that have debug enabled labled by account",
		[]string{accountLabel})

	metrics.accountBidResponseValidationSizeError = newCounter(cfg, reg,
		"account_response_validation_size_err",
		"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight labeled by account (enforce) ",
		[]string{accountLabel, successLabel})

	metrics.accountBidResponseValidationSizeWarn = newCounter(cfg, reg,
		"account_response_validation_size_warn",
		"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight labeled by account (warn)",
		[]string{accountLabel, successLabel})

	metrics.accountBidResponseSecureMarkupError = newCounter(cfg, reg,
		"account_response_validation_secure_err",
		"Count that tracks number of bids removed from bid response that had a invalid bidAdm labeled by account (enforce) ",
		[]string{accountLabel, successLabel})

	metrics.accountBidResponseSecureMarkupWarn = newCounter(cfg, reg,
		"account_response_validation_secure_warn",
		"Count that tracks number of bids removed from bid response that had a invalid bidAdm labeled by account (warn)",
		[]string{accountLabel, successLabel})

	metrics.requestsQueueTimer = newHistogramVec(cfg, reg,
		"request_queue_time",
		"Seconds request was waiting in queue",
		[]string{requestTypeLabel, requestStatusLabel},
		queuedRequestTimeBuckets)

	metrics.accountStoredResponses = newCounter(cfg, reg,
		"account_stored_responses",
		"Count of total requests to Prebid Server that have stored responses labled by account",
		[]string{accountLabel})

	metrics.adsCertSignTimer = newHistogram(cfg, reg,
		"ads_cert_sign_time",
		"Seconds to generate an AdsCert header",
		standardTimeBuckets)

	metrics.adsCertRequests = newCounter(cfg, reg,
		"ads_cert_requests",
		"Count of AdsCert request, and if they were successfully sent.",
		[]string{successLabel})

	createModulesMetrics(cfg, reg, &metrics, moduleStageNames, standardTimeBuckets)

	metrics.Gatherer = reg

	metricsPrefix := ""
	if len(cfg.Namespace) > 0 {
		metricsPrefix += fmt.Sprintf("%s_", cfg.Namespace)
	}
	if len(cfg.Subsystem) > 0 {
		metricsPrefix += fmt.Sprintf("%s_", cfg.Subsystem)
	}

	metrics.Registerer = prometheus.WrapRegistererWithPrefix(metricsPrefix, reg)
	metrics.Registerer.MustRegister(promCollector.NewGoCollector())

	preloadLabelValues(&metrics, syncerKeys, moduleStageNames)

	return &metrics
}

func createModulesMetrics(cfg config.PrometheusMetrics, registry *prometheus.Registry, m *Metrics, moduleStageNames map[string][]string, standardTimeBuckets []float64) {
	l := len(moduleStageNames)
	m.moduleDuration = make(map[string]*prometheus.HistogramVec, l)
	m.moduleCalls = make(map[string]*prometheus.CounterVec, l)
	m.moduleFailures = make(map[string]*prometheus.CounterVec, l)
	m.moduleSuccessNoops = make(map[string]*prometheus.CounterVec, l)
	m.moduleSuccessUpdates = make(map[string]*prometheus.CounterVec, l)
	m.moduleSuccessRejects = make(map[string]*prometheus.CounterVec, l)
	m.moduleExecutionErrors = make(map[string]*prometheus.CounterVec, l)
	m.moduleTimeouts = make(map[string]*prometheus.CounterVec, l)

	// create for each registered module its own metric
	for module := range moduleStageNames {
		m.moduleDuration[module] = newHistogramVec(cfg, registry,
			fmt.Sprintf("modules_%s_duration", module),
			"Amount of seconds a module processed a hook labeled by stage name.",
			[]string{stageLabel},
			standardTimeBuckets)

		m.moduleCalls[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_called", module),
			"Count of module calls labeled by stage name.",
			[]string{stageLabel})

		m.moduleFailures[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_failed", module),
			"Count of module fails labeled by stage name.",
			[]string{stageLabel})

		m.moduleSuccessNoops[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_success_noops", module),
			"Count of module successful noops labeled by stage name.",
			[]string{stageLabel})

		m.moduleSuccessUpdates[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_success_updates", module),
			"Count of module successful updates labeled by stage name.",
			[]string{stageLabel})

		m.moduleSuccessRejects[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_success_rejects", module),
			"Count of module successful rejects labeled by stage name.",
			[]string{stageLabel})

		m.moduleExecutionErrors[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_execution_errors", module),
			"Count of module execution errors labeled by stage name.",
			[]string{stageLabel})

		m.moduleTimeouts[module] = newCounter(cfg, registry,
			fmt.Sprintf("modules_%s_timeouts", module),
			"Count of module timeouts labeled by stage name.",
			[]string{stageLabel})
	}
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

func (m *Metrics) RecordTMaxTimeout() {
	m.tmaxTimeout.Inc()
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

func (m *Metrics) RecordDebugRequest(debugEnabled bool, pubID string) {
	if debugEnabled {
		m.debugRequests.Inc()
		if !m.metricsDisabled.AccountDebug && pubID != metrics.PublisherUnknown {
			m.accountDebugRequests.With(prometheus.Labels{
				accountLabel: pubID,
			}).Inc()
		}
	}
}

func (m *Metrics) RecordStoredResponse(pubId string) {
	m.storedResponses.Inc()
	if !m.metricsDisabled.AccountStoredResponses && pubId != metrics.PublisherUnknown {
		m.accountStoredResponses.With(prometheus.Labels{
			accountLabel: pubId,
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
	case metrics.ResponseDataType:
		m.storedResponsesFetchTimer.With(prometheus.Labels{
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
	case metrics.ResponseDataType:
		m.storedResponsesErrors.With(prometheus.Labels{
			storedDataErrorLabel: string(labels.Error),
		}).Inc()
	}
}

func (m *Metrics) RecordAdapterRequest(labels metrics.AdapterLabels) {
	lowerCasedAdapter := strings.ToLower(string(labels.Adapter))
	m.adapterRequests.With(prometheus.Labels{
		adapterLabel: lowerCasedAdapter,
		cookieLabel:  string(labels.CookieFlag),
		hasBidsLabel: strconv.FormatBool(labels.AdapterBids == metrics.AdapterBidPresent),
	}).Inc()

	for err := range labels.AdapterErrors {
		m.adapterErrors.With(prometheus.Labels{
			adapterLabel:      lowerCasedAdapter,
			adapterErrorLabel: string(err),
		}).Inc()
	}
}

// Keeps track of created and reused connections to adapter bidders and the time from the
// connection request, to the connection creation, or reuse from the pool across all engines
func (m *Metrics) RecordAdapterConnections(adapterName openrtb_ext.BidderName, connWasReused bool, connWaitTime time.Duration) {
	lowerCasedAdapterName := strings.ToLower(string(adapterName))
	if m.metricsDisabled.AdapterConnectionMetrics {
		return
	}

	if connWasReused {
		m.adapterReusedConnections.With(prometheus.Labels{
			adapterLabel: lowerCasedAdapterName,
		}).Inc()
	} else {
		m.adapterCreatedConnections.With(prometheus.Labels{
			adapterLabel: lowerCasedAdapterName,
		}).Inc()
	}

	m.adapterConnectionWaitTime.With(prometheus.Labels{
		adapterLabel: lowerCasedAdapterName,
	}).Observe(connWaitTime.Seconds())
}

func (m *Metrics) RecordDNSTime(dnsLookupTime time.Duration) {
	m.dnsLookupTimer.Observe(dnsLookupTime.Seconds())
}

func (m *Metrics) RecordTLSHandshakeTime(tlsHandshakeTime time.Duration) {
	m.tlsHandhakeTimer.Observe(tlsHandshakeTime.Seconds())
}

func (m *Metrics) RecordBidderServerResponseTime(bidderServerResponseTime time.Duration) {
	m.bidderServerResponseTimer.Observe(bidderServerResponseTime.Seconds())
}

func (m *Metrics) RecordAdapterPanic(labels metrics.AdapterLabels) {
	m.adapterPanics.With(prometheus.Labels{
		adapterLabel: strings.ToLower(string(labels.Adapter)),
	}).Inc()
}

func (m *Metrics) RecordAdapterBidReceived(labels metrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	markupDelivery := markupDeliveryNurl
	if hasAdm {
		markupDelivery = markupDeliveryAdm
	}

	m.adapterBids.With(prometheus.Labels{
		adapterLabel:        strings.ToLower(string(labels.Adapter)),
		markupDeliveryLabel: markupDelivery,
	}).Inc()
}

func (m *Metrics) RecordAdapterPrice(labels metrics.AdapterLabels, cpm float64) {
	m.adapterPrices.With(prometheus.Labels{
		adapterLabel: strings.ToLower(string(labels.Adapter)),
	}).Observe(cpm)
}

func (m *Metrics) RecordOverheadTime(overhead metrics.OverheadType, duration time.Duration) {
	m.overheadTimer.With(prometheus.Labels{
		overheadTypeLabel: overhead.String(),
	}).Observe(duration.Seconds())
}

func (m *Metrics) RecordAdapterTime(labels metrics.AdapterLabels, length time.Duration) {
	if len(labels.AdapterErrors) == 0 {
		m.adapterRequestsTimer.With(prometheus.Labels{
			adapterLabel: strings.ToLower(string(labels.Adapter)),
		}).Observe(length.Seconds())
	}
}

func (m *Metrics) RecordCookieSync(status metrics.CookieSyncStatus) {
	m.cookieSync.With(prometheus.Labels{
		statusLabel: string(status),
	}).Inc()
}

func (m *Metrics) RecordSyncerRequest(key string, status metrics.SyncerCookieSyncStatus) {
	m.syncerRequests.With(prometheus.Labels{
		syncerLabel: key,
		statusLabel: string(status),
	}).Inc()
}

func (m *Metrics) RecordSetUid(status metrics.SetUidStatus) {
	m.setUid.With(prometheus.Labels{
		statusLabel: string(status),
	}).Inc()
}

func (m *Metrics) RecordSyncerSet(key string, status metrics.SyncerSetUidStatus) {
	m.syncerSets.With(prometheus.Labels{
		syncerLabel: key,
		statusLabel: string(status),
	}).Inc()
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

func (m *Metrics) RecordAdapterBuyerUIDScrubbed(adapterName openrtb_ext.BidderName) {
	if m.metricsDisabled.AdapterBuyerUIDScrubbed {
		return
	}

	m.adapterScrubbedBuyerUIDs.With(prometheus.Labels{
		adapterLabel: strings.ToLower(string(adapterName)),
	}).Inc()
}

func (m *Metrics) RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName) {
	if m.metricsDisabled.AdapterGDPRRequestBlocked {
		return
	}

	m.adapterGDPRBlockedRequests.With(prometheus.Labels{
		adapterLabel: strings.ToLower(string(adapterName)),
	}).Inc()
}

func (m *Metrics) RecordAdsCertReq(success bool) {
	if success {
		m.adsCertRequests.With(prometheus.Labels{
			successLabel: requestSuccessful,
		}).Inc()
	} else {
		m.adsCertRequests.With(prometheus.Labels{
			successLabel: requestFailed,
		}).Inc()
	}
}
func (m *Metrics) RecordAdsCertSignTime(adsCertSignTime time.Duration) {
	m.adsCertSignTimer.Observe(adsCertSignTime.Seconds())
}

func (m *Metrics) RecordBidValidationCreativeSizeError(adapter openrtb_ext.BidderName, account string) {
	lowerCasedAdapter := strings.ToLower(string(adapter))
	m.adapterBidResponseValidationSizeError.With(prometheus.Labels{
		adapterLabel: lowerCasedAdapter, successLabel: successLabel,
	}).Inc()

	if !m.metricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		m.accountBidResponseValidationSizeError.With(prometheus.Labels{
			accountLabel: account, successLabel: successLabel,
		}).Inc()
	}
}

func (m *Metrics) RecordBidValidationCreativeSizeWarn(adapter openrtb_ext.BidderName, account string) {
	lowerCasedAdapter := strings.ToLower(string(adapter))
	m.adapterBidResponseValidationSizeWarn.With(prometheus.Labels{
		adapterLabel: lowerCasedAdapter, successLabel: successLabel,
	}).Inc()

	if !m.metricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		m.accountBidResponseValidationSizeWarn.With(prometheus.Labels{
			accountLabel: account, successLabel: successLabel,
		}).Inc()
	}
}

func (m *Metrics) RecordBidValidationSecureMarkupError(adapter openrtb_ext.BidderName, account string) {
	m.adapterBidResponseSecureMarkupError.With(prometheus.Labels{
		adapterLabel: strings.ToLower(string(adapter)), successLabel: successLabel,
	}).Inc()

	if !m.metricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		m.accountBidResponseSecureMarkupError.With(prometheus.Labels{
			accountLabel: account, successLabel: successLabel,
		}).Inc()
	}
}

func (m *Metrics) RecordBidValidationSecureMarkupWarn(adapter openrtb_ext.BidderName, account string) {
	m.adapterBidResponseSecureMarkupWarn.With(prometheus.Labels{
		adapterLabel: strings.ToLower(string(adapter)), successLabel: successLabel,
	}).Inc()

	if !m.metricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		m.accountBidResponseSecureMarkupWarn.With(prometheus.Labels{
			accountLabel: account, successLabel: successLabel,
		}).Inc()
	}
}

func (m *Metrics) RecordModuleCalled(labels metrics.ModuleLabels, duration time.Duration) {
	m.moduleCalls[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()

	m.moduleDuration[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Observe(duration.Seconds())
}

func (m *Metrics) RecordModuleFailed(labels metrics.ModuleLabels) {
	m.moduleFailures[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()
}

func (m *Metrics) RecordModuleSuccessNooped(labels metrics.ModuleLabels) {
	m.moduleSuccessNoops[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()
}

func (m *Metrics) RecordModuleSuccessUpdated(labels metrics.ModuleLabels) {
	m.moduleSuccessUpdates[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()
}

func (m *Metrics) RecordModuleSuccessRejected(labels metrics.ModuleLabels) {
	m.moduleSuccessRejects[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()
}

func (m *Metrics) RecordModuleExecutionError(labels metrics.ModuleLabels) {
	m.moduleExecutionErrors[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()
}

func (m *Metrics) RecordModuleTimeout(labels metrics.ModuleLabels) {
	m.moduleTimeouts[labels.Module].With(prometheus.Labels{
		stageLabel: labels.Stage,
	}).Inc()
}
