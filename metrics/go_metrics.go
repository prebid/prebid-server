package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	metrics "github.com/rcrowley/go-metrics"
)

// Metrics is the legacy Metrics object (go-metrics) expanded to also satisfy the MetricsEngine interface
type Metrics struct {
	MetricsRegistry                metrics.Registry
	ConnectionCounter              metrics.Counter
	ConnectionAcceptErrorMeter     metrics.Meter
	ConnectionCloseErrorMeter      metrics.Meter
	ImpMeter                       metrics.Meter
	LegacyImpMeter                 metrics.Meter
	AppRequestMeter                metrics.Meter
	NoCookieMeter                  metrics.Meter
	RequestTimer                   metrics.Timer
	RequestsQueueTimer             map[RequestType]map[bool]metrics.Timer
	PrebidCacheRequestTimerSuccess metrics.Timer
	PrebidCacheRequestTimerError   metrics.Timer
	StoredDataFetchTimer           map[StoredDataType]map[StoredDataFetchType]metrics.Timer
	StoredDataErrorMeter           map[StoredDataType]map[StoredDataError]metrics.Meter
	StoredReqCacheMeter            map[CacheResult]metrics.Meter
	StoredImpCacheMeter            map[CacheResult]metrics.Meter
	AccountCacheMeter              map[CacheResult]metrics.Meter
	DNSLookupTimer                 metrics.Timer
	TLSHandshakeTimer              metrics.Timer

	// Metrics for OpenRTB requests specifically. So we can track what % of RequestsMeter are OpenRTB
	// and know when legacy requests have been abandoned.
	RequestStatuses       map[RequestType]map[RequestStatus]metrics.Meter
	AmpNoCookieMeter      metrics.Meter
	CookieSyncMeter       metrics.Meter
	CookieSyncGen         map[openrtb_ext.BidderName]metrics.Meter
	CookieSyncGDPRPrevent map[openrtb_ext.BidderName]metrics.Meter
	userSyncOptout        metrics.Meter
	userSyncBadRequest    metrics.Meter
	userSyncSet           map[openrtb_ext.BidderName]metrics.Meter
	userSyncGDPRPrevent   map[openrtb_ext.BidderName]metrics.Meter

	// Media types found in the "imp" JSON object
	ImpsTypeBanner metrics.Meter
	ImpsTypeVideo  metrics.Meter
	ImpsTypeAudio  metrics.Meter
	ImpsTypeNative metrics.Meter

	// Notification timeout metrics
	TimeoutNotificationSuccess metrics.Meter
	TimeoutNotificationFailure metrics.Meter

	// TCF adaption metrics
	PrivacyCCPARequest       metrics.Meter
	PrivacyCCPARequestOptOut metrics.Meter
	PrivacyCOPPARequest      metrics.Meter
	PrivacyLMTRequest        metrics.Meter
	PrivacyTCFRequestVersion map[TCFVersionValue]metrics.Meter

	AdapterMetrics map[openrtb_ext.BidderName]*AdapterMetrics
	// Don't export accountMetrics because we need helper functions here to insure its properly populated dynamically
	accountMetrics        map[string]*accountMetrics
	accountMetricsRWMutex sync.RWMutex
	userSyncRwMutex       sync.RWMutex

	exchanges []openrtb_ext.BidderName
	// Will hold boolean values to help us disable metric collection if needed
	MetricsDisabled config.DisabledMetrics
}

// AdapterMetrics houses the metrics for a particular adapter
type AdapterMetrics struct {
	NoCookieMeter      metrics.Meter
	ErrorMeters        map[AdapterError]metrics.Meter
	NoBidMeter         metrics.Meter
	GotBidsMeter       metrics.Meter
	RequestTimer       metrics.Timer
	PriceHistogram     metrics.Histogram
	BidsReceivedMeter  metrics.Meter
	PanicMeter         metrics.Meter
	MarkupMetrics      map[openrtb_ext.BidType]*MarkupDeliveryMetrics
	ConnCreated        metrics.Counter
	ConnReused         metrics.Counter
	ConnWaitTime       metrics.Timer
	GDPRRequestBlocked metrics.Meter
}

type MarkupDeliveryMetrics struct {
	AdmMeter  metrics.Meter
	NurlMeter metrics.Meter
}

type accountMetrics struct {
	requestMeter      metrics.Meter
	bidsReceivedMeter metrics.Meter
	priceHistogram    metrics.Histogram
	// store account by adapter metrics. Type is map[PBSBidder.BidderCode]
	adapterMetrics map[openrtb_ext.BidderName]*AdapterMetrics
}

// Defining an "unknown" bidder
const unknownBidder openrtb_ext.BidderName = "unknown"

// NewBlankMetrics creates a new Metrics object with all blank metrics object. This may also be useful for
// testing routines to ensure that no metrics are written anywhere.
//
// This will be useful when removing endpoints, we can just run will the blank metrics function
// rather than loading legacy metrics that never get filled.
// This will also eventually let us configure metrics, such as setting a limited set of metrics
// for a production instance, and then expanding again when we need more debugging.
func NewBlankMetrics(registry metrics.Registry, exchanges []openrtb_ext.BidderName, disabledMetrics config.DisabledMetrics) *Metrics {
	blankMeter := &metrics.NilMeter{}
	blankTimer := &metrics.NilTimer{}

	newMetrics := &Metrics{
		MetricsRegistry:                registry,
		RequestStatuses:                make(map[RequestType]map[RequestStatus]metrics.Meter),
		ConnectionCounter:              metrics.NilCounter{},
		ConnectionAcceptErrorMeter:     blankMeter,
		ConnectionCloseErrorMeter:      blankMeter,
		ImpMeter:                       blankMeter,
		LegacyImpMeter:                 blankMeter,
		AppRequestMeter:                blankMeter,
		NoCookieMeter:                  blankMeter,
		RequestTimer:                   blankTimer,
		DNSLookupTimer:                 blankTimer,
		TLSHandshakeTimer:              blankTimer,
		RequestsQueueTimer:             make(map[RequestType]map[bool]metrics.Timer),
		PrebidCacheRequestTimerSuccess: blankTimer,
		PrebidCacheRequestTimerError:   blankTimer,
		StoredDataFetchTimer:           make(map[StoredDataType]map[StoredDataFetchType]metrics.Timer),
		StoredDataErrorMeter:           make(map[StoredDataType]map[StoredDataError]metrics.Meter),
		StoredReqCacheMeter:            make(map[CacheResult]metrics.Meter),
		StoredImpCacheMeter:            make(map[CacheResult]metrics.Meter),
		AccountCacheMeter:              make(map[CacheResult]metrics.Meter),
		AmpNoCookieMeter:               blankMeter,
		CookieSyncMeter:                blankMeter,
		CookieSyncGen:                  make(map[openrtb_ext.BidderName]metrics.Meter),
		CookieSyncGDPRPrevent:          make(map[openrtb_ext.BidderName]metrics.Meter),
		userSyncOptout:                 blankMeter,
		userSyncBadRequest:             blankMeter,
		userSyncSet:                    make(map[openrtb_ext.BidderName]metrics.Meter),
		userSyncGDPRPrevent:            make(map[openrtb_ext.BidderName]metrics.Meter),

		ImpsTypeBanner: blankMeter,
		ImpsTypeVideo:  blankMeter,
		ImpsTypeAudio:  blankMeter,
		ImpsTypeNative: blankMeter,

		TimeoutNotificationSuccess: blankMeter,
		TimeoutNotificationFailure: blankMeter,

		PrivacyCCPARequest:       blankMeter,
		PrivacyCCPARequestOptOut: blankMeter,
		PrivacyCOPPARequest:      blankMeter,
		PrivacyLMTRequest:        blankMeter,
		PrivacyTCFRequestVersion: make(map[TCFVersionValue]metrics.Meter, len(TCFVersions())),

		AdapterMetrics:  make(map[openrtb_ext.BidderName]*AdapterMetrics, len(exchanges)),
		accountMetrics:  make(map[string]*accountMetrics),
		MetricsDisabled: disabledMetrics,

		exchanges: exchanges,
	}

	for _, a := range exchanges {
		newMetrics.AdapterMetrics[a] = makeBlankAdapterMetrics(newMetrics.MetricsDisabled)
	}

	for _, t := range RequestTypes() {
		newMetrics.RequestStatuses[t] = make(map[RequestStatus]metrics.Meter)
		for _, s := range RequestStatuses() {
			newMetrics.RequestStatuses[t][s] = blankMeter
		}
	}

	for _, c := range CacheResults() {
		newMetrics.StoredReqCacheMeter[c] = blankMeter
		newMetrics.StoredImpCacheMeter[c] = blankMeter
		newMetrics.AccountCacheMeter[c] = blankMeter
	}

	for _, v := range TCFVersions() {
		newMetrics.PrivacyTCFRequestVersion[v] = blankMeter
	}

	for _, dt := range StoredDataTypes() {
		newMetrics.StoredDataFetchTimer[dt] = make(map[StoredDataFetchType]metrics.Timer)
		newMetrics.StoredDataErrorMeter[dt] = make(map[StoredDataError]metrics.Meter)
		for _, ft := range StoredDataFetchTypes() {
			newMetrics.StoredDataFetchTimer[dt][ft] = blankTimer
		}
		for _, e := range StoredDataErrors() {
			newMetrics.StoredDataErrorMeter[dt][e] = blankMeter
		}
	}

	//to minimize memory usage, queuedTimeout metric is now supported for video endpoint only
	//boolean value represents 2 general request statuses: accepted and rejected
	newMetrics.RequestsQueueTimer["video"] = make(map[bool]metrics.Timer)
	newMetrics.RequestsQueueTimer["video"][true] = blankTimer
	newMetrics.RequestsQueueTimer["video"][false] = blankTimer
	return newMetrics
}

// NewMetrics creates a new Metrics object with needed metrics defined. In time we may develop to the point
// where Metrics contains all the metrics we might want to record, and then we build the actual
// metrics object to contain only the metrics we are interested in. This would allow for debug
// mode metrics. The code would allways try to record the metrics, but effectively noop if we are
// using a blank meter/timer.
func NewMetrics(registry metrics.Registry, exchanges []openrtb_ext.BidderName, disableAccountMetrics config.DisabledMetrics) *Metrics {
	newMetrics := NewBlankMetrics(registry, exchanges, disableAccountMetrics)
	newMetrics.ConnectionCounter = metrics.GetOrRegisterCounter("active_connections", registry)
	newMetrics.ConnectionAcceptErrorMeter = metrics.GetOrRegisterMeter("connection_accept_errors", registry)
	newMetrics.ConnectionCloseErrorMeter = metrics.GetOrRegisterMeter("connection_close_errors", registry)
	newMetrics.ImpMeter = metrics.GetOrRegisterMeter("imps_requested", registry)
	newMetrics.LegacyImpMeter = metrics.GetOrRegisterMeter("legacy_imps_requested", registry)

	newMetrics.ImpsTypeBanner = metrics.GetOrRegisterMeter("imp_banner", registry)
	newMetrics.ImpsTypeVideo = metrics.GetOrRegisterMeter("imp_video", registry)
	newMetrics.ImpsTypeAudio = metrics.GetOrRegisterMeter("imp_audio", registry)
	newMetrics.ImpsTypeNative = metrics.GetOrRegisterMeter("imp_native", registry)

	newMetrics.NoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", registry)
	newMetrics.AppRequestMeter = metrics.GetOrRegisterMeter("app_requests", registry)
	newMetrics.RequestTimer = metrics.GetOrRegisterTimer("request_time", registry)
	newMetrics.DNSLookupTimer = metrics.GetOrRegisterTimer("dns_lookup_time", registry)
	newMetrics.TLSHandshakeTimer = metrics.GetOrRegisterTimer("tls_handshake_time", registry)
	newMetrics.PrebidCacheRequestTimerSuccess = metrics.GetOrRegisterTimer("prebid_cache_request_time.ok", registry)
	newMetrics.PrebidCacheRequestTimerError = metrics.GetOrRegisterTimer("prebid_cache_request_time.err", registry)

	for _, dt := range StoredDataTypes() {
		for _, ft := range StoredDataFetchTypes() {
			timerName := fmt.Sprintf("stored_%s_fetch_time.%s", string(dt), string(ft))
			newMetrics.StoredDataFetchTimer[dt][ft] = metrics.GetOrRegisterTimer(timerName, registry)
		}
		for _, e := range StoredDataErrors() {
			meterName := fmt.Sprintf("stored_%s_error.%s", string(dt), string(e))
			newMetrics.StoredDataErrorMeter[dt][e] = metrics.GetOrRegisterMeter(meterName, registry)
		}
	}

	newMetrics.AmpNoCookieMeter = metrics.GetOrRegisterMeter("amp_no_cookie_requests", registry)
	newMetrics.CookieSyncMeter = metrics.GetOrRegisterMeter("cookie_sync_requests", registry)
	newMetrics.userSyncBadRequest = metrics.GetOrRegisterMeter("usersync.bad_requests", registry)
	newMetrics.userSyncOptout = metrics.GetOrRegisterMeter("usersync.opt_outs", registry)
	for _, a := range exchanges {
		newMetrics.CookieSyncGen[a] = metrics.GetOrRegisterMeter(fmt.Sprintf("cookie_sync.%s.gen", string(a)), registry)
		newMetrics.CookieSyncGDPRPrevent[a] = metrics.GetOrRegisterMeter(fmt.Sprintf("cookie_sync.%s.gdpr_prevent", string(a)), registry)
		newMetrics.userSyncSet[a] = metrics.GetOrRegisterMeter(fmt.Sprintf("usersync.%s.sets", string(a)), registry)
		newMetrics.userSyncGDPRPrevent[a] = metrics.GetOrRegisterMeter(fmt.Sprintf("usersync.%s.gdpr_prevent", string(a)), registry)
		registerAdapterMetrics(registry, "adapter", string(a), newMetrics.AdapterMetrics[a])
	}
	for typ, statusMap := range newMetrics.RequestStatuses {
		for stat := range statusMap {
			statusMap[stat] = metrics.GetOrRegisterMeter("requests."+string(stat)+"."+string(typ), registry)
		}
	}

	for _, cacheRes := range CacheResults() {
		newMetrics.StoredReqCacheMeter[cacheRes] = metrics.GetOrRegisterMeter(fmt.Sprintf("stored_request_cache_%s", string(cacheRes)), registry)
		newMetrics.StoredImpCacheMeter[cacheRes] = metrics.GetOrRegisterMeter(fmt.Sprintf("stored_imp_cache_%s", string(cacheRes)), registry)
		newMetrics.AccountCacheMeter[cacheRes] = metrics.GetOrRegisterMeter(fmt.Sprintf("account_cache_%s", string(cacheRes)), registry)
	}

	newMetrics.RequestsQueueTimer["video"][true] = metrics.GetOrRegisterTimer("queued_requests.video.accepted", registry)
	newMetrics.RequestsQueueTimer["video"][false] = metrics.GetOrRegisterTimer("queued_requests.video.rejected", registry)

	newMetrics.userSyncSet[unknownBidder] = metrics.GetOrRegisterMeter("usersync.unknown.sets", registry)
	newMetrics.userSyncGDPRPrevent[unknownBidder] = metrics.GetOrRegisterMeter("usersync.unknown.gdpr_prevent", registry)

	newMetrics.TimeoutNotificationSuccess = metrics.GetOrRegisterMeter("timeout_notification.ok", registry)
	newMetrics.TimeoutNotificationFailure = metrics.GetOrRegisterMeter("timeout_notification.failed", registry)

	newMetrics.PrivacyCCPARequest = metrics.GetOrRegisterMeter("privacy.request.ccpa.specified", registry)
	newMetrics.PrivacyCCPARequestOptOut = metrics.GetOrRegisterMeter("privacy.request.ccpa.opt-out", registry)
	newMetrics.PrivacyCOPPARequest = metrics.GetOrRegisterMeter("privacy.request.coppa", registry)
	newMetrics.PrivacyLMTRequest = metrics.GetOrRegisterMeter("privacy.request.lmt", registry)
	for _, version := range TCFVersions() {
		newMetrics.PrivacyTCFRequestVersion[version] = metrics.GetOrRegisterMeter(fmt.Sprintf("privacy.request.tcf.%s", string(version)), registry)
	}

	return newMetrics
}

// Part of setting up blank metrics, the adapter metrics.
func makeBlankAdapterMetrics(disabledMetrics config.DisabledMetrics) *AdapterMetrics {
	blankMeter := &metrics.NilMeter{}
	newAdapter := &AdapterMetrics{
		NoCookieMeter:     blankMeter,
		ErrorMeters:       make(map[AdapterError]metrics.Meter),
		NoBidMeter:        blankMeter,
		GotBidsMeter:      blankMeter,
		RequestTimer:      &metrics.NilTimer{},
		PriceHistogram:    &metrics.NilHistogram{},
		BidsReceivedMeter: blankMeter,
		PanicMeter:        blankMeter,
		MarkupMetrics:     makeBlankBidMarkupMetrics(),
	}
	if !disabledMetrics.AdapterConnectionMetrics {
		newAdapter.ConnCreated = metrics.NilCounter{}
		newAdapter.ConnReused = metrics.NilCounter{}
		newAdapter.ConnWaitTime = &metrics.NilTimer{}
	}
	if !disabledMetrics.AdapterGDPRRequestBlocked {
		newAdapter.GDPRRequestBlocked = blankMeter
	}
	for _, err := range AdapterErrors() {
		newAdapter.ErrorMeters[err] = blankMeter
	}
	return newAdapter
}

func makeBlankBidMarkupMetrics() map[openrtb_ext.BidType]*MarkupDeliveryMetrics {
	return map[openrtb_ext.BidType]*MarkupDeliveryMetrics{
		openrtb_ext.BidTypeAudio:  makeBlankMarkupDeliveryMetrics(),
		openrtb_ext.BidTypeBanner: makeBlankMarkupDeliveryMetrics(),
		openrtb_ext.BidTypeNative: makeBlankMarkupDeliveryMetrics(),
		openrtb_ext.BidTypeVideo:  makeBlankMarkupDeliveryMetrics(),
	}
}

func makeBlankMarkupDeliveryMetrics() *MarkupDeliveryMetrics {
	return &MarkupDeliveryMetrics{
		AdmMeter:  &metrics.NilMeter{},
		NurlMeter: &metrics.NilMeter{},
	}
}

func registerAdapterMetrics(registry metrics.Registry, adapterOrAccount string, exchange string, am *AdapterMetrics) {
	am.NoCookieMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_cookie_requests", adapterOrAccount, exchange), registry)
	am.NoBidMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests.nobid", adapterOrAccount, exchange), registry)
	am.GotBidsMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests.gotbids", adapterOrAccount, exchange), registry)
	am.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.request_time", adapterOrAccount, exchange), registry)
	am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("%[1]s.%[2]s.prices", adapterOrAccount, exchange), registry, metrics.NewExpDecaySample(1028, 0.015))
	am.MarkupMetrics = map[openrtb_ext.BidType]*MarkupDeliveryMetrics{
		openrtb_ext.BidTypeBanner: makeDeliveryMetrics(registry, adapterOrAccount+"."+exchange, openrtb_ext.BidTypeBanner),
		openrtb_ext.BidTypeVideo:  makeDeliveryMetrics(registry, adapterOrAccount+"."+exchange, openrtb_ext.BidTypeVideo),
		openrtb_ext.BidTypeAudio:  makeDeliveryMetrics(registry, adapterOrAccount+"."+exchange, openrtb_ext.BidTypeAudio),
		openrtb_ext.BidTypeNative: makeDeliveryMetrics(registry, adapterOrAccount+"."+exchange, openrtb_ext.BidTypeNative),
	}
	am.ConnCreated = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.connections_created", adapterOrAccount, exchange), registry)
	am.ConnReused = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.connections_reused", adapterOrAccount, exchange), registry)
	am.ConnWaitTime = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.connection_wait_time", adapterOrAccount, exchange), registry)
	for err := range am.ErrorMeters {
		am.ErrorMeters[err] = metrics.GetOrRegisterMeter(fmt.Sprintf("%s.%s.requests.%s", adapterOrAccount, exchange, err), registry)
	}
	if adapterOrAccount != "adapter" {
		am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), registry)
	}
	am.PanicMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests.panic", adapterOrAccount, exchange), registry)
	am.GDPRRequestBlocked = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.gdpr_request_blocked", adapterOrAccount, exchange), registry)
}

func makeDeliveryMetrics(registry metrics.Registry, prefix string, bidType openrtb_ext.BidType) *MarkupDeliveryMetrics {
	return &MarkupDeliveryMetrics{
		AdmMeter:  metrics.GetOrRegisterMeter(prefix+"."+string(bidType)+".adm_bids_received", registry),
		NurlMeter: metrics.GetOrRegisterMeter(prefix+"."+string(bidType)+".nurl_bids_received", registry),
	}
}

// getAccountMetrics gets or registers the account metrics for account "id".
// There is no getBlankAccountMetrics() as all metrics are generated dynamically.
func (me *Metrics) getAccountMetrics(id string) *accountMetrics {
	var am *accountMetrics
	var ok bool

	me.accountMetricsRWMutex.RLock()
	am, ok = me.accountMetrics[id]
	me.accountMetricsRWMutex.RUnlock()

	if ok {
		return am
	}

	me.accountMetricsRWMutex.Lock()
	// Made sure to use defer as we have two exit points: we want to unlock the mutex as quickly as possible.
	defer me.accountMetricsRWMutex.Unlock()

	am, ok = me.accountMetrics[id]
	if ok {
		// Unlock and return as quickly as possible
		return am
	}
	am = &accountMetrics{}
	am.requestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.requests", id), me.MetricsRegistry)
	am.bidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.bids_received", id), me.MetricsRegistry)
	am.priceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), me.MetricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
	am.adapterMetrics = make(map[openrtb_ext.BidderName]*AdapterMetrics, len(me.exchanges))
	if !me.MetricsDisabled.AccountAdapterDetails {
		for _, a := range me.exchanges {
			am.adapterMetrics[a] = makeBlankAdapterMetrics(me.MetricsDisabled)
			registerAdapterMetrics(me.MetricsRegistry, fmt.Sprintf("account.%s", id), string(a), am.adapterMetrics[a])
		}
	}

	me.accountMetrics[id] = am

	return am
}

// Implement the MetricsEngine interface

// RecordRequest implements a part of the MetricsEngine interface
func (me *Metrics) RecordRequest(labels Labels) {
	me.RequestStatuses[labels.RType][labels.RequestStatus].Mark(1)
	if labels.Source == DemandApp {
		me.AppRequestMeter.Mark(1)
	} else {
		if labels.CookieFlag == CookieFlagNo {
			// NOTE: Old behavior was log me.AMPNoCookieMeter here for AMP requests.
			// AMP is still new and OpenRTB does not do this, so changing to match
			// OpenRTB endpoint
			me.NoCookieMeter.Mark(1)
		}
	}

	// Handle the account metrics now.
	am := me.getAccountMetrics(labels.PubID)
	am.requestMeter.Mark(1)
}

func (me *Metrics) RecordImps(labels ImpLabels) {
	me.ImpMeter.Mark(int64(1))
	if labels.BannerImps {
		me.ImpsTypeBanner.Mark(int64(1))
	}
	if labels.VideoImps {
		me.ImpsTypeVideo.Mark(int64(1))
	}
	if labels.AudioImps {
		me.ImpsTypeAudio.Mark(int64(1))
	}
	if labels.NativeImps {
		me.ImpsTypeNative.Mark(int64(1))
	}
}

func (me *Metrics) RecordLegacyImps(labels Labels, numImps int) {
	me.LegacyImpMeter.Mark(int64(numImps))
}

func (me *Metrics) RecordConnectionAccept(success bool) {
	if success {
		me.ConnectionCounter.Inc(1)
	} else {
		me.ConnectionAcceptErrorMeter.Mark(1)
	}
}

func (me *Metrics) RecordConnectionClose(success bool) {
	if success {
		me.ConnectionCounter.Dec(1)
	} else {
		me.ConnectionCloseErrorMeter.Mark(1)
	}
}

// RecordRequestTime implements a part of the MetricsEngine interface. The calling code is responsible
// for determining the call duration.
func (me *Metrics) RecordRequestTime(labels Labels, length time.Duration) {
	// Only record times for successful requests, as we don't have labels to screen out bad requests.
	if labels.RequestStatus == RequestStatusOK {
		me.RequestTimer.Update(length)
	}
}

// RecordStoredDataFetchTime implements a part of the MetricsEngine interface
func (me *Metrics) RecordStoredDataFetchTime(labels StoredDataLabels, length time.Duration) {
	me.StoredDataFetchTimer[labels.DataType][labels.DataFetchType].Update(length)
}

// RecordStoredDataError implements a part of the MetricsEngine interface
func (me *Metrics) RecordStoredDataError(labels StoredDataLabels) {
	me.StoredDataErrorMeter[labels.DataType][labels.Error].Mark(1)
}

// RecordAdapterPanic implements a part of the MetricsEngine interface
func (me *Metrics) RecordAdapterPanic(labels AdapterLabels) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}
	am.PanicMeter.Mark(1)
}

// RecordAdapterRequest implements a part of the MetricsEngine interface
func (me *Metrics) RecordAdapterRequest(labels AdapterLabels) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}

	aam, ok := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	switch labels.AdapterBids {
	case AdapterBidNone:
		am.NoBidMeter.Mark(1)
		if ok {
			aam.NoBidMeter.Mark(1)
		}
	case AdapterBidPresent:
		am.GotBidsMeter.Mark(1)
		if ok {
			aam.GotBidsMeter.Mark(1)
		}
	default:
		glog.Warningf("No go-metrics logged for AdapterBids value: %s", labels.AdapterBids)
	}
	for errType := range labels.AdapterErrors {
		am.ErrorMeters[errType].Mark(1)
	}

	if labels.CookieFlag == CookieFlagNo {
		am.NoCookieMeter.Mark(1)
	}
}

// Keeps track of created and reused connections to adapter bidders and the time from the
// connection request, to the connection creation, or reuse from the pool across all engines
func (me *Metrics) RecordAdapterConnections(adapterName openrtb_ext.BidderName,
	connWasReused bool,
	connWaitTime time.Duration) {

	if me.MetricsDisabled.AdapterConnectionMetrics {
		return
	}

	am, ok := me.AdapterMetrics[adapterName]
	if !ok {
		glog.Errorf("Trying to log adapter connection metrics for %s: adapter not found", string(adapterName))
		return
	}

	if connWasReused {
		am.ConnReused.Inc(1)
	} else {
		am.ConnCreated.Inc(1)
	}
	am.ConnWaitTime.Update(connWaitTime)
}

func (me *Metrics) RecordDNSTime(dnsLookupTime time.Duration) {
	me.DNSLookupTimer.Update(dnsLookupTime)
}

func (me *Metrics) RecordTLSHandshakeTime(tlsHandshakeTime time.Duration) {
	me.TLSHandshakeTimer.Update(tlsHandshakeTime)
}

// RecordAdapterBidReceived implements a part of the MetricsEngine interface.
// This tracks how many bids from each Bidder use `adm` vs. `nurl.
func (me *Metrics) RecordAdapterBidReceived(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter bid metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}

	// Adapter metrics
	am.BidsReceivedMeter.Mark(1)
	// Account-Adapter metrics
	if aam, ok := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]; ok {
		aam.BidsReceivedMeter.Mark(1)
	}

	if metricsForType, ok := am.MarkupMetrics[bidType]; ok {
		if hasAdm {
			metricsForType.AdmMeter.Mark(1)
		} else {
			metricsForType.NurlMeter.Mark(1)
		}
	} else {
		glog.Errorf("bid/adm metrics map entry does not exist for type %s. This is a bug, and should be reported.", bidType)
	}
	return
}

// RecordAdapterPrice implements a part of the MetricsEngine interface. Generates a histogram of winning bid prices
func (me *Metrics) RecordAdapterPrice(labels AdapterLabels, cpm float64) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter price metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}
	// Adapter metrics
	am.PriceHistogram.Update(int64(cpm))
	// Account-Adapter metrics
	if aam, ok := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]; ok {
		aam.PriceHistogram.Update(int64(cpm))
	}
}

// RecordAdapterTime implements a part of the MetricsEngine interface. Records the adapter response time
func (me *Metrics) RecordAdapterTime(labels AdapterLabels, length time.Duration) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter latency metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}
	// Adapter metrics
	am.RequestTimer.Update(length)
	// Account-Adapter metrics
	if aam, ok := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]; ok {
		aam.RequestTimer.Update(length)
	}
}

// RecordCookieSync implements a part of the MetricsEngine interface. Records a cookie sync request
func (me *Metrics) RecordCookieSync() {
	me.CookieSyncMeter.Mark(1)
}

// RecordAdapterCookieSync implements a part of the MetricsEngine interface. Records a cookie sync adpter sync request and gdpr status
func (me *Metrics) RecordAdapterCookieSync(adapter openrtb_ext.BidderName, gdprBlocked bool) {
	me.CookieSyncGen[adapter].Mark(1)
	if gdprBlocked {
		me.CookieSyncGDPRPrevent[adapter].Mark(1)
	}
}

// RecordUserIDSet implements a part of the MetricsEngine interface. Records a cookie setuid request
func (me *Metrics) RecordUserIDSet(userLabels UserLabels) {
	switch userLabels.Action {
	case RequestActionOptOut:
		me.userSyncOptout.Mark(1)
	case RequestActionErr:
		me.userSyncBadRequest.Mark(1)
	case RequestActionSet:
		doMark(userLabels.Bidder, me.userSyncSet)
	case RequestActionGDPR:
		doMark(userLabels.Bidder, me.userSyncGDPRPrevent)
	}
}

// RecordStoredReqCacheResult implements a part of the MetricsEngine interface. Records the
// cache hits and misses when looking up stored requests
func (me *Metrics) RecordStoredReqCacheResult(cacheResult CacheResult, inc int) {
	me.StoredReqCacheMeter[cacheResult].Mark(int64(inc))
}

// RecordStoredImpCacheResult implements a part of the MetricsEngine interface. Records the
// cache hits and misses when looking up stored impressions.
func (me *Metrics) RecordStoredImpCacheResult(cacheResult CacheResult, inc int) {
	me.StoredImpCacheMeter[cacheResult].Mark(int64(inc))
}

// RecordAccountCacheResult implements a part of the MetricsEngine interface. Records the
// cache hits and misses when looking up accounts.
func (me *Metrics) RecordAccountCacheResult(cacheResult CacheResult, inc int) {
	me.AccountCacheMeter[cacheResult].Mark(int64(inc))
}

// RecordPrebidCacheRequestTime implements a part of the MetricsEngine interface. Records the
// amount of time taken to store the auction result in Prebid Cache.
func (me *Metrics) RecordPrebidCacheRequestTime(success bool, length time.Duration) {
	if success {
		me.PrebidCacheRequestTimerSuccess.Update(length)
	} else {
		me.PrebidCacheRequestTimerError.Update(length)
	}
}

func (me *Metrics) RecordRequestQueueTime(success bool, requestType RequestType, length time.Duration) {
	if requestType == ReqTypeVideo { //remove this check when other request types are supported
		me.RequestsQueueTimer[requestType][success].Update(length)
	}

}

func (me *Metrics) RecordTimeoutNotice(success bool) {
	if success {
		me.TimeoutNotificationSuccess.Mark(1)
	} else {
		me.TimeoutNotificationFailure.Mark(1)
	}
	return
}

func (me *Metrics) RecordRequestPrivacy(privacy PrivacyLabels) {
	if privacy.CCPAProvided {
		me.PrivacyCCPARequest.Mark(1)
		if privacy.CCPAEnforced {
			me.PrivacyCCPARequestOptOut.Mark(1)
		}
	}

	if privacy.COPPAEnforced {
		me.PrivacyCOPPARequest.Mark(1)
	}

	if privacy.GDPREnforced {
		if metric, ok := me.PrivacyTCFRequestVersion[privacy.GDPRTCFVersion]; ok {
			metric.Mark(1)
		} else {
			me.PrivacyTCFRequestVersion[TCFVersionErr].Mark(1)
		}
	}

	if privacy.LMTEnforced {
		me.PrivacyLMTRequest.Mark(1)
	}
	return
}

func (me *Metrics) RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName) {
	if me.MetricsDisabled.AdapterGDPRRequestBlocked {
		return
	}

	am, ok := me.AdapterMetrics[adapterName]
	if !ok {
		glog.Errorf("Trying to log adapter GDPR request blocked metric for %s: adapter not found", string(adapterName))
		return
	}

	am.GDPRRequestBlocked.Mark(1)
}

func doMark(bidder openrtb_ext.BidderName, meters map[openrtb_ext.BidderName]metrics.Meter) {
	met, ok := meters[bidder]
	if ok {
		met.Mark(1)
	} else {
		meters[unknownBidder].Mark(1)
	}
}
