package opentelemetry

import (
	"context"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type (
	PbsMetrics struct {
		ConnectionsClosed   metric.Float64Counter `description:"Count of successful connections closed to Prebid Server" unit:"1"`
		ConnectionsError    metric.Float64Counter `description:"Count of errors for connection open and close attempts to Prebid Server labeled by type" unit:"1"`
		ConnectionsOpened   metric.Float64Counter `description:"Count of successful connections opened to Prebid Server" unit:"1"`
		TmaxTimeout         metric.Float64Counter `description:"Count of requests rejected due to Tmax timeout exceed" unit:"1"`
		CookieSyncRequests  metric.Float64Counter `description:"Count of cookie sync requests to Prebid Server" unit:"1"`
		SetuidRequests      metric.Float64Counter `description:"Count of set uid requests to Prebid Server" unit:"1"`
		ImpressionsRequests metric.Float64Counter `description:"Count of requested impressions to Prebid Server labeled by type" unit:"1"`

		PrebidcacheWriteTime metric.Float64Histogram `description:"Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts" unit:"s" buckets:"0.001,0.002,0.005,0.01,0.025,0.05,0.1,0.2,0.3,0.4,0.5,1"`

		Requests              metric.Float64Counter   `description:"Count of total requests to Prebid Server labeled by type and status" unit:"1"`
		DebugRequests         metric.Float64Counter   `description:"Count of total requests to Prebid Server that have debug enabled" unit:"1"`
		RequestTime           metric.Float64Histogram `description:"Seconds to resolve successful Prebid Server requests labeled by type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		RequestsWithoutCookie metric.Float64Counter   `description:"Count of total requests to Prebid Server without a cookie labeled by type" unit:"1"`

		StoredImpressionsCachePerformance metric.Float64Counter   `description:"Count of stored impression cache requests attempts by hits or miss" unit:"1"`
		StoredRequestCachePerformance     metric.Float64Counter   `description:"Count of stored request cache requests attempts by hits or miss" unit:"1"`
		AccountCachePerformance           metric.Float64Counter   `description:"Count of account cache lookups by hits or miss" unit:"1"`
		StoredAccountFetchTime            metric.Float64Histogram `description:"Seconds to fetch stored accounts labeled by fetch type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		StoredAccountErrors               metric.Float64Counter   `description:"Count of stored account errors by error type" unit:"1"`
		StoredAmpFetchTime                metric.Float64Histogram `description:"Seconds to fetch stored AMP requests labeled by fetch type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		StoredAmpErrors                   metric.Float64Counter   `description:"Count of stored AMP errors by error type" unit:"1"`
		StoredCategoryFetchTime           metric.Float64Histogram `description:"Seconds to fetch stored categories labeled by fetch type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		StoredCategoryErrors              metric.Float64Counter   `description:"Count of stored category errors by error type" unit:"1"`
		StoredRequestFetchTime            metric.Float64Histogram `description:"Seconds to fetch stored requests labeled by fetch type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		StoredRequestErrors               metric.Float64Counter   `description:"Count of stored request errors by error type" unit:"1"`
		StoredVideoFetchTime              metric.Float64Histogram `description:"Seconds to fetch stored video labeled by fetch type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		StoredVideoErrors                 metric.Float64Counter   `description:"Count of stored video errors by error type" unit:"1"`

		TimeoutNotification metric.Float64Counter   `description:"Count of timeout notifications triggered, and if they were successfully sent" unit:"1"`
		DnsLookupTime       metric.Float64Histogram `description:"Seconds to resolve DNS" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		TlsHandhakeTime     metric.Float64Histogram `description:"Seconds to perform TLS Handshake" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`

		PrivacyCCPA  metric.Float64Counter `description:"Count of total requests to Prebid Server where CCPA was provided by source and opt-out" unit:"1"`
		PrivacyCOPPA metric.Float64Counter `description:"Count of total requests to Prebid Server where the COPPA flag was set by source" unit:"1"`
		PrivacyTCF   metric.Float64Counter `description:"Count of TCF versions for requests where GDPR was enforced by source and version" unit:"1"`
		PrivacyLMT   metric.Float64Counter `description:"Count of total requests to Prebid Server where the LMT flag was set by source" unit:"1"`

		AdapterBuyeruidsScrubbed   metric.Float64Counter `description:"Count of total bidder requests with a scrubbed buyeruid due to a privacy policy" unit:"1"`
		AdapterGdprRequestsBlocked metric.Float64Counter `description:"Count of total bidder requests blocked due to unsatisfied GDPR purpose 2 legal basis" unit:"1"`

		StoredResponsesFetchTime metric.Float64Histogram `description:"Seconds to fetch stored responses labeled by fetch type" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		StoredResponsesErrors    metric.Float64Counter   `description:"Count of stored response errors by error type" unit:"1"`
		StoredResponses          metric.Float64Counter   `description:"Count of total requests to Prebid Server that have stored responses" unit:"1"`

		AdapterBids                         metric.Float64Counter   `description:"Count of bids labeled by adapter and markup delivery type (adm or nurl)" unit:"1"`
		AdapterErrors                       metric.Float64Counter   `description:"Count of errors labeled by adapter and error type" unit:"1"`
		AdapterPanics                       metric.Float64Counter   `description:"Count of panics labeled by adapter" unit:"1"`
		AdapterPrices                       metric.Float64Histogram `description:"Monetary value of the bids labeled by adapter" unit:"$" buckets:"250,500,750,1000,1500,2000,2500,3000,3500,4000"`
		AdapterRequests                     metric.Float64Counter   `description:"Count of requests labeled by adapter, if has a cookie, and if it resulted in bids" unit:"1"`
		AdapterConnectionCreated            metric.Float64Counter   `description:"Count that keeps track of new connections when contacting adapter bidder endpoints" unit:"1"`
		AdapterConnectionReused             metric.Float64Counter   `description:"Count that keeps track of reused connections when contacting adapter bidder endpoints" unit:"1"`
		AdapterConnectionWait               metric.Float64Histogram `description:"Seconds from when the connection was requested until it is either created or reused" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		AdapterResponseValidationSizeErr    metric.Float64Counter   `description:"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight" unit:"1"`
		AdapterResponseValidationSizeWarn   metric.Float64Counter   `description:"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight (warn)" unit:"1"`
		AdapterResponseValidationSecureErr  metric.Float64Counter   `description:"Count that tracks number of bids removed from bid response that had a invalid bidAdm" unit:"1"`
		AdapterResponseValidationSecureWarn metric.Float64Counter   `description:"Count that tracks number of bids removed from bid response that had a invalid bidAdm (warn)" unit:"1"`

		OverheadTime             metric.Float64Histogram `description:"Seconds to prepare adapter request or resolve adapter response" unit:"s" buckets:"0.05,0.06,0.07,0.08,0.09,0.1,0.2,0.3,0.4,0.5,0.6,0.7,0.8,0.9,1"`
		AdapterRequestTime       metric.Float64Histogram `description:"Seconds to resolve each successful request labeled by adapter" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		BidderServerResponseTime metric.Float64Histogram `description:"Duration needed to send HTTP request and receive response back from bidder server" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		SyncerRequests           metric.Float64Counter   `description:"Count of cookie sync requests where a syncer is a candidate to be synced labeled by syncer key and status" unit:"1"`
		SyncerSets               metric.Float64Counter   `description:"Count of setuid set requests for a syncer labeled by syncer key and status" unit:"1"`

		AccountRequests                     metric.Float64Counter `description:"Count of total requests to Prebid Server labeled by account" unit:"1"`
		AccountDebugRequests                metric.Int64Counter   `description:"Count of total requests to Prebid Server that have debug enabled labled by account" unit:"1"`
		AccountResponseValidationSizeErr    metric.Float64Counter `description:"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight labeled by account (enforce)" unit:"1"`
		AccountResponseValidationSizeWarn   metric.Float64Counter `description:"Count that tracks number of bids removed from bid response that had a creative size greater than maxWidth/maxHeight labeled by account (warn)" unit:"1"`
		AccountResponseValidationSecureErr  metric.Float64Counter `description:"Count that tracks number of bids removed from bid response that had a invalid bidAdm labeled by account (enforce)" unit:"1"`
		AccountResponseValidationSecureWarn metric.Float64Counter `description:"Count that tracks number of bids removed from bid response that had a invalid bidAdm labeled by account (warn)" unit:"1"`

		RequestsQueueTime      metric.Float64Histogram `description:"Seconds request was waiting in queue" unit:"s" buckets:"0,1,5,30,60,120,180,240,300"`
		AccountStoredResponses metric.Float64Counter   `description:"Count of total requests to Prebid Server that have stored responses labled by account" unit:"1"`
		AdsCertSignTime        metric.Float64Histogram `description:"Seconds to generate an AdsCert header" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
		AdsCertRequests        metric.Float64Counter   `description:"Count of AdsCert request, and if they were successfully sent" unit:"1"`

		// Module metrics - Note these are renamed so can use Int64Counter
		Module struct {
			Duration        metric.Float64Histogram `description:"Amount of seconds a module processed a hook labeled by stage name" unit:"s" buckets:"0.05,0.1,0.15,0.20,0.25,0.3,0.4,0.5,0.75,1"`
			Called          metric.Int64Counter     `description:"Count of module calls labeled by stage name" unit:"1"`
			Failed          metric.Int64Counter     `description:"Count of module fails labeled by stage name" unit:"1"`
			SuccessNoops    metric.Int64Counter     `description:"Count of module successful noops labeled by stage name" unit:"1"`
			SuccessUpdates  metric.Int64Counter     `description:"Count of module successful updates labeled by stage name" unit:"1"`
			SuccessRejects  metric.Int64Counter     `description:"Count of module successful rejects labeled by stage name" unit:"1"`
			ExecutionErrors metric.Int64Counter     `description:"Count of module execution errors labeled by stage name" unit:"1"`
			Timeouts        metric.Int64Counter     `description:"Count of module timeouts labeled by stage name" unit:"1"`
		}
	}
	PbsMetricsEngine struct {
		*PbsMetrics
		MetricsDisabled *config.DisabledMetrics
	}
)

const (
	markupDeliveryAdm  = "adm"
	markupDeliveryNurl = "nurl"
)

const (
	accountLabel         attribute.Key = "account"
	actionLabel          attribute.Key = "action"
	adapterErrorLabel    attribute.Key = "adapter_error"
	adapterLabel         attribute.Key = "adapter"
	bidTypeLabel         attribute.Key = "bid_type"
	cacheResultLabel     attribute.Key = "cache_result"
	connectionErrorLabel attribute.Key = "connection_error"
	cookieLabel          attribute.Key = "cookie"
	hasBidsLabel         attribute.Key = "has_bids"
	isAudioLabel         attribute.Key = "audio"
	isBannerLabel        attribute.Key = "banner"
	isNativeLabel        attribute.Key = "native"
	isVideoLabel         attribute.Key = "video"
	markupDeliveryLabel  attribute.Key = "delivery"
	optOutLabel          attribute.Key = "opt_out"
	overheadTypeLabel    attribute.Key = "overhead_type"
	privacyBlockedLabel  attribute.Key = "privacy_blocked"
	requestStatusLabel   attribute.Key = "request_status"
	requestTypeLabel     attribute.Key = "request_type"
	stageLabel           attribute.Key = "stage"
	statusLabel          attribute.Key = "status"
	successLabel         attribute.Key = "success"
	syncerLabel          attribute.Key = "syncer"
	versionLabel         attribute.Key = "version"

	storedDataFetchTypeLabel attribute.Key = "stored_data_fetch_type"
	storedDataErrorLabel     attribute.Key = "stored_data_error"

	requestSuccessLabel attribute.Key = "requestAcceptedLabel"
	requestRejectLabel  attribute.Key = "requestRejectedLabel"

	sourceLabel   attribute.Key = "source"
	sourceRequest attribute.Key = "request"

	moduleLabel attribute.Key = "module"
)

const (
	requestSuccessful = "ok"
	requestFailed     = "failed"
)

var (
	_ metrics.MetricsEngine = &PbsMetricsEngine{}

	Meter = otel.Meter("github.com/prebid/prebid-server/v3")
)

// NewEngine creates a new OpenTelemetry engine with the given prefix
func NewEngine(prefix string, disabledMetrics *config.DisabledMetrics) (*PbsMetricsEngine, error) {
	ret := &PbsMetricsEngine{
		PbsMetrics:      &PbsMetrics{},
		MetricsDisabled: disabledMetrics,
	}
	if err := InitMetrics(Meter, ret.PbsMetrics, prefix); err != nil {
		return nil, err
	}
	return ret, nil
}

func (o *PbsMetricsEngine) RecordConnectionAccept(success bool) {
	ctx := context.Background()
	if success {
		o.ConnectionsOpened.Add(ctx, 1)
	} else {
		o.ConnectionsError.Add(ctx, 1, metric.WithAttributes(connectionErrorLabel.String("accept")))
	}
}

func (o *PbsMetricsEngine) RecordTMaxTimeout() {
	ctx := context.Background()
	o.TmaxTimeout.Add(ctx, 1)
}

func (o *PbsMetricsEngine) RecordConnectionClose(success bool) {
	ctx := context.Background()
	if success {
		o.ConnectionsClosed.Add(ctx, 1)
	} else {
		o.ConnectionsError.Add(ctx, 1, metric.WithAttributes(connectionErrorLabel.String("close")))
	}
}

func (o *PbsMetricsEngine) RecordRequest(labels metrics.Labels) {
	ctx := context.Background()
	o.Requests.Add(ctx, 1, metric.WithAttributes(
		requestTypeLabel.String(string(labels.RType)),
		requestStatusLabel.String(string(labels.RequestStatus)),
	))

	if labels.CookieFlag == metrics.CookieFlagNo {
		o.RequestsWithoutCookie.Add(ctx, 1, metric.WithAttributes(
			requestTypeLabel.String(string(labels.RType)),
		))
	}

	if labels.PubID != metrics.PublisherUnknown {
		o.AccountRequests.Add(ctx, 1, metric.WithAttributes(
			accountLabel.String(labels.PubID),
		))
	}
}

func (o *PbsMetricsEngine) RecordImps(labels metrics.ImpLabels) {
	ctx := context.Background()
	o.ImpressionsRequests.Add(ctx, 1, metric.WithAttributes(
		isBannerLabel.Bool(labels.BannerImps),
		isVideoLabel.Bool(labels.VideoImps),
		isAudioLabel.Bool(labels.AudioImps),
		isNativeLabel.Bool(labels.NativeImps),
	))
}

func (o *PbsMetricsEngine) RecordRequestTime(labels metrics.Labels, length time.Duration) {
	ctx := context.Background()
	o.RequestTime.Record(ctx, length.Seconds(), metric.WithAttributes(
		requestTypeLabel.String(string(labels.RType)),
	))
}

func (o *PbsMetricsEngine) RecordOverheadTime(overHead metrics.OverheadType, length time.Duration) {
	ctx := context.Background()
	o.OverheadTime.Record(ctx, length.Seconds(), metric.WithAttributes(
		overheadTypeLabel.String(string(overHead)),
	))
}

func (o *PbsMetricsEngine) RecordAdapterRequest(labels metrics.AdapterLabels) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(labels.Adapter))
	o.AdapterRequests.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
		cookieLabel.String(string(labels.CookieFlag)),
		hasBidsLabel.Bool(labels.AdapterBids == metrics.AdapterBidPresent),
	))

	for adapterError := range labels.AdapterErrors {
		o.AdapterErrors.Add(ctx, 1, metric.WithAttributes(
			adapterLabel.String(lowerCasedAdapter),
			adapterErrorLabel.String(string(adapterError)),
		))
	}
}

func (o *PbsMetricsEngine) RecordAdapterConnections(adapterName openrtb_ext.BidderName, connWasReused bool, connWaitTime time.Duration) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(adapterName))
	if connWasReused {
		o.AdapterConnectionReused.Add(ctx, 1, metric.WithAttributes(
			adapterLabel.String(lowerCasedAdapter),
		))
	} else {
		o.AdapterConnectionCreated.Add(ctx, 1, metric.WithAttributes(
			adapterLabel.String(lowerCasedAdapter),
		))
	}

	o.AdapterConnectionWait.Record(ctx, connWaitTime.Seconds(), metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
	))
}

func (o *PbsMetricsEngine) RecordDNSTime(dnsLookupTime time.Duration) {
	ctx := context.Background()
	o.DnsLookupTime.Record(ctx, dnsLookupTime.Seconds())
}

func (o *PbsMetricsEngine) RecordTLSHandshakeTime(tlsHandshakeTime time.Duration) {
	ctx := context.Background()
	o.TlsHandhakeTime.Record(ctx, tlsHandshakeTime.Seconds())
}

func (o *PbsMetricsEngine) RecordBidderServerResponseTime(bidderServerResponseTime time.Duration) {
	ctx := context.Background()
	o.BidderServerResponseTime.Record(ctx, bidderServerResponseTime.Seconds())
}

func (o *PbsMetricsEngine) RecordAdapterPanic(labels metrics.AdapterLabels) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(labels.Adapter))
	o.AdapterPanics.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
	))
}

func (o *PbsMetricsEngine) RecordAdapterBidReceived(labels metrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	ctx := context.Background()
	markupDelivery := markupDeliveryNurl
	if hasAdm {
		markupDelivery = markupDeliveryAdm
	}
	lowerCasedAdapter := strings.ToLower(string(labels.Adapter))
	o.AdapterBids.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
		markupDeliveryLabel.String(markupDelivery),
	))
}

func (o *PbsMetricsEngine) RecordAdapterPrice(labels metrics.AdapterLabels, cpm float64) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(labels.Adapter))
	o.AdapterPrices.Record(ctx, cpm, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
	))
}

func (o *PbsMetricsEngine) RecordAdapterTime(labels metrics.AdapterLabels, length time.Duration) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(labels.Adapter))
	o.AdapterRequestTime.Record(ctx, length.Seconds(), metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
	))
}

func (o *PbsMetricsEngine) RecordCookieSync(status metrics.CookieSyncStatus) {
	ctx := context.Background()
	o.CookieSyncRequests.Add(ctx, 1, metric.WithAttributes(
		statusLabel.String(string(status)),
	))
}

func (o *PbsMetricsEngine) RecordSyncerRequest(key string, status metrics.SyncerCookieSyncStatus) {
	ctx := context.Background()
	o.SyncerRequests.Add(ctx, 1, metric.WithAttributes(
		syncerLabel.String(key),
		statusLabel.String(string(status)),
	))
}

func (o *PbsMetricsEngine) RecordSetUid(status metrics.SetUidStatus) {
	ctx := context.Background()
	o.SetuidRequests.Add(ctx, 1, metric.WithAttributes(
		statusLabel.String(string(status)),
	))
}

func (o *PbsMetricsEngine) RecordSyncerSet(key string, status metrics.SyncerSetUidStatus) {
	ctx := context.Background()
	o.SyncerSets.Add(ctx, 1, metric.WithAttributes(
		syncerLabel.String(key),
		statusLabel.String(string(status)),
	))
}

func (o *PbsMetricsEngine) RecordStoredReqCacheResult(cacheResult metrics.CacheResult, inc int) {
	ctx := context.Background()
	incVal := float64(inc)
	o.StoredRequestCachePerformance.Add(ctx, incVal, metric.WithAttributes(
		cacheResultLabel.String(string(cacheResult)),
	))
}

func (o *PbsMetricsEngine) RecordStoredImpCacheResult(cacheResult metrics.CacheResult, inc int) {
	ctx := context.Background()
	incVal := float64(inc)
	o.StoredImpressionsCachePerformance.Add(ctx, incVal, metric.WithAttributes(
		cacheResultLabel.String(string(cacheResult)),
	))
}

func (o *PbsMetricsEngine) RecordAccountCacheResult(cacheResult metrics.CacheResult, inc int) {
	ctx := context.Background()
	incVal := float64(inc)
	o.AccountCachePerformance.Add(ctx, incVal, metric.WithAttributes(
		cacheResultLabel.String(string(cacheResult)),
	))
}

func (o *PbsMetricsEngine) RecordStoredDataFetchTime(labels metrics.StoredDataLabels, length time.Duration) {
	ctx := context.Background()
	var histogramPtr *metric.Float64Histogram
	switch labels.DataType {
	case metrics.AccountDataType:
		histogramPtr = &o.StoredAccountFetchTime
	case metrics.AMPDataType:
		histogramPtr = &o.StoredAmpFetchTime
	case metrics.CategoryDataType:
		histogramPtr = &o.StoredCategoryFetchTime
	case metrics.RequestDataType:
		histogramPtr = &o.StoredRequestFetchTime
	case metrics.VideoDataType:
		histogramPtr = &o.StoredVideoFetchTime
	case metrics.ResponseDataType:
		histogramPtr = &o.StoredResponsesFetchTime
	default:
		glog.Error("unknown data type: %v", labels.DataType)
		return
	}
	// Record the chosen histogram
	(*histogramPtr).Record(ctx, length.Seconds(), metric.WithAttributes(
		storedDataFetchTypeLabel.String(string(labels.DataFetchType)),
	))
}

func (o *PbsMetricsEngine) RecordStoredDataError(labels metrics.StoredDataLabels) {
	ctx := context.Background()
	var counterPtr *metric.Float64Counter
	switch labels.DataType {
	case metrics.AccountDataType:
		counterPtr = &o.StoredAccountErrors
	case metrics.AMPDataType:
		counterPtr = &o.StoredAmpErrors
	case metrics.CategoryDataType:
		counterPtr = &o.StoredCategoryErrors
	case metrics.RequestDataType:
		counterPtr = &o.StoredRequestErrors
	case metrics.VideoDataType:
		counterPtr = &o.StoredVideoErrors
	case metrics.ResponseDataType:
		counterPtr = &o.StoredResponsesErrors
	default:
		glog.Error(ctx, "unknown data type: %v", labels.DataType)
		return
	}
	// Record the chosen histogram
	(*counterPtr).Add(ctx, 1, metric.WithAttributes(
		storedDataErrorLabel.String(string(labels.Error)),
	))
}

func (o *PbsMetricsEngine) RecordPrebidCacheRequestTime(success bool, length time.Duration) {
	ctx := context.Background()
	o.PrebidcacheWriteTime.Record(ctx, length.Seconds(), metric.WithAttributes(
		successLabel.Bool(success),
	))
}

func (o *PbsMetricsEngine) RecordRequestQueueTime(success bool, requestType metrics.RequestType, length time.Duration) {
	ctx := context.Background()
	successLabelFormatted := requestRejectLabel
	if success {
		successLabelFormatted = requestSuccessLabel
	}

	o.RequestsQueueTime.Record(ctx, length.Seconds(), metric.WithAttributes(
		requestTypeLabel.String(string(requestType)),
		requestStatusLabel.String(string(successLabelFormatted)),
	))
}

func (o *PbsMetricsEngine) RecordTimeoutNotice(success bool) {
	ctx := context.Background()
	successFormatted := requestFailed
	if success {
		successFormatted = requestSuccessful
	}
	o.TimeoutNotification.Add(ctx, 1, metric.WithAttributes(
		successLabel.String(successFormatted),
	))
}

func (o *PbsMetricsEngine) RecordRequestPrivacy(privacy metrics.PrivacyLabels) {
	ctx := context.Background()
	if privacy.CCPAProvided {
		o.PrivacyCCPA.Add(ctx, 1, metric.WithAttributes(
			sourceLabel.String(string(sourceRequest)),
			optOutLabel.Bool(privacy.CCPAEnforced),
		))
	}
	if privacy.COPPAEnforced {
		o.PrivacyCOPPA.Add(ctx, 1, metric.WithAttributes(
			sourceLabel.String(string(sourceRequest)),
		))
	}
	if privacy.GDPREnforced {
		o.PrivacyTCF.Add(ctx, 1, metric.WithAttributes(
			versionLabel.String(string(privacy.GDPRTCFVersion)),
			sourceLabel.String(string(sourceRequest)),
		))

	}
	if privacy.LMTEnforced {
		o.PrivacyLMT.Add(ctx, 1, metric.WithAttributes(
			sourceLabel.String(string(sourceRequest)),
		))
	}
}

func (o *PbsMetricsEngine) RecordAdapterBuyerUIDScrubbed(adapterName openrtb_ext.BidderName) {
	ctx := context.Background()
	o.AdapterBuyeruidsScrubbed.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(string(adapterName)),
	))
}

func (o *PbsMetricsEngine) RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName) {
	if o.MetricsDisabled.AdapterGDPRRequestBlocked {
		return
	}
	ctx := context.Background()
	o.AdapterGdprRequestsBlocked.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(string(adapterName)),
	))
}

func (o *PbsMetricsEngine) RecordDebugRequest(debugEnabled bool, pubId string) {
	ctx := context.Background()
	if debugEnabled {
		o.DebugRequests.Add(ctx, 1)
		if !o.MetricsDisabled.AccountDebug && pubId != metrics.PublisherUnknown {
			o.AccountDebugRequests.Add(ctx, 1, metric.WithAttributes(
				accountLabel.String(pubId),
			))
		}
	}
}

func (o *PbsMetricsEngine) RecordStoredResponse(pubId string) {
	ctx := context.Background()
	o.StoredResponses.Add(ctx, 1)
	if !o.MetricsDisabled.AccountStoredResponses && pubId != metrics.PublisherUnknown {
		o.AccountStoredResponses.Add(ctx, 1, metric.WithAttributes(
			accountLabel.String(pubId),
		))
	}
}

func (o *PbsMetricsEngine) RecordAdsCertReq(success bool) {
	ctx := context.Background()
	successFormatted := requestFailed
	if success {
		successFormatted = requestSuccessful
	}
	o.AdsCertRequests.Add(ctx, 1, metric.WithAttributes(
		successLabel.String(successFormatted),
	))
}

func (o *PbsMetricsEngine) RecordAdsCertSignTime(adsCertSignTime time.Duration) {
	ctx := context.Background()
	o.AdsCertSignTime.Record(ctx, adsCertSignTime.Seconds())
}

func (o *PbsMetricsEngine) RecordBidValidationCreativeSizeError(adapter openrtb_ext.BidderName, account string) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(adapter))
	o.AdapterResponseValidationSizeErr.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
		successLabel.String(string(successLabel)),
	))

	if !o.MetricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		o.AccountResponseValidationSizeErr.Add(ctx, 1, metric.WithAttributes(
			accountLabel.String(account),
			successLabel.String(string(successLabel)),
		))
	}
}

func (o *PbsMetricsEngine) RecordBidValidationCreativeSizeWarn(adapter openrtb_ext.BidderName, account string) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(adapter))
	o.AdapterResponseValidationSizeWarn.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
		successLabel.String(string(successLabel)),
	))

	if !o.MetricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		o.AccountResponseValidationSizeWarn.Add(ctx, 1, metric.WithAttributes(
			accountLabel.String(account),
			successLabel.String(string(successLabel)),
		))
	}
}

func (o *PbsMetricsEngine) RecordBidValidationSecureMarkupError(adapter openrtb_ext.BidderName, account string) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(adapter))
	o.AdapterResponseValidationSecureErr.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
		successLabel.String(string(successLabel)),
	))

	if !o.MetricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		o.AccountResponseValidationSecureErr.Add(ctx, 1, metric.WithAttributes(
			accountLabel.String(account),
			successLabel.String(string(successLabel)),
		))
	}
}

func (o *PbsMetricsEngine) RecordBidValidationSecureMarkupWarn(adapter openrtb_ext.BidderName, account string) {
	ctx := context.Background()
	lowerCasedAdapter := strings.ToLower(string(adapter))
	o.AdapterResponseValidationSecureWarn.Add(ctx, 1, metric.WithAttributes(
		adapterLabel.String(lowerCasedAdapter),
		successLabel.String(string(successLabel)),
	))

	if !o.MetricsDisabled.AccountAdapterDetails && account != metrics.PublisherUnknown {
		o.AccountResponseValidationSecureWarn.Add(ctx, 1, metric.WithAttributes(
			accountLabel.String(account),
			successLabel.String(string(successLabel)),
		))
	}
}

func (o *PbsMetricsEngine) RecordModuleCalled(labels metrics.ModuleLabels, duration time.Duration) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.Called.Add(ctx, 1, attributesOpt)
	o.Module.Duration.Record(ctx, duration.Seconds(), attributesOpt)
}

func (o *PbsMetricsEngine) RecordModuleFailed(labels metrics.ModuleLabels) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.Failed.Add(ctx, 1, attributesOpt)
}

func (o *PbsMetricsEngine) RecordModuleSuccessNooped(labels metrics.ModuleLabels) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.SuccessNoops.Add(ctx, 1, attributesOpt)
}

func (o *PbsMetricsEngine) RecordModuleSuccessUpdated(labels metrics.ModuleLabels) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.SuccessUpdates.Add(ctx, 1, attributesOpt)
}

func (o *PbsMetricsEngine) RecordModuleSuccessRejected(labels metrics.ModuleLabels) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.SuccessRejects.Add(ctx, 1, attributesOpt)
}

func (o *PbsMetricsEngine) RecordModuleExecutionError(labels metrics.ModuleLabels) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.ExecutionErrors.Add(ctx, 1, attributesOpt)
}

func (o *PbsMetricsEngine) RecordModuleTimeout(labels metrics.ModuleLabels) {
	ctx := context.Background()
	attributesOpt := metric.WithAttributes(
		stageLabel.String(labels.Stage),
		moduleLabel.String(labels.Module),
	)
	o.Module.Timeouts.Add(ctx, 1, attributesOpt)
}
