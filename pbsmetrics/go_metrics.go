package pbsmetrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/openrtb_ext"
	metrics "github.com/rcrowley/go-metrics"
)

// Metrics is the legacy Metrics object (go-metrics) expanded to also satisfy the MetricsEngine interface
type Metrics struct {
	MetricsRegistry            metrics.Registry
	ConnectionCounter          metrics.Counter
	ConnectionAcceptErrorMeter metrics.Meter
	ConnectionCloseErrorMeter  metrics.Meter
	ImpMeter                   metrics.Meter
	AppRequestMeter            metrics.Meter
	NoCookieMeter              metrics.Meter
	SafariRequestMeter         metrics.Meter
	SafariNoCookieMeter        metrics.Meter
	RequestTimer               metrics.Timer
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

	AdapterMetrics map[openrtb_ext.BidderName]*AdapterMetrics
	// Don't export accountMetrics because we need helper functions here to insure its properly populated dynamically
	accountMetrics        map[string]*accountMetrics
	accountMetricsRWMutex sync.RWMutex
	userSyncRwMutex       sync.RWMutex

	exchanges []openrtb_ext.BidderName
}

// AdapterMetrics houses the metrics for a particular adapter
type AdapterMetrics struct {
	NoCookieMeter     metrics.Meter
	ErrorMeters       map[AdapterError]metrics.Meter
	NoBidMeter        metrics.Meter
	GotBidsMeter      metrics.Meter
	RequestTimer      metrics.Timer
	PriceHistogram    metrics.Histogram
	BidsReceivedMeter metrics.Meter
	PanicMeter        metrics.Meter
	MarkupMetrics     map[openrtb_ext.BidType]*MarkupDeliveryMetrics
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
func NewBlankMetrics(registry metrics.Registry, exchanges []openrtb_ext.BidderName) *Metrics {
	blankMeter := &metrics.NilMeter{}
	newMetrics := &Metrics{
		MetricsRegistry:            registry,
		RequestStatuses:            make(map[RequestType]map[RequestStatus]metrics.Meter),
		ConnectionCounter:          metrics.NilCounter{},
		ConnectionAcceptErrorMeter: blankMeter,
		ConnectionCloseErrorMeter:  blankMeter,
		ImpMeter:                   blankMeter,
		AppRequestMeter:            blankMeter,
		NoCookieMeter:              blankMeter,
		SafariRequestMeter:         blankMeter,
		SafariNoCookieMeter:        blankMeter,
		RequestTimer:               &metrics.NilTimer{},
		AmpNoCookieMeter:           blankMeter,
		CookieSyncMeter:            blankMeter,
		CookieSyncGen:              make(map[openrtb_ext.BidderName]metrics.Meter),
		CookieSyncGDPRPrevent:      make(map[openrtb_ext.BidderName]metrics.Meter),
		userSyncOptout:             blankMeter,
		userSyncBadRequest:         blankMeter,
		userSyncSet:                make(map[openrtb_ext.BidderName]metrics.Meter),
		userSyncGDPRPrevent:        make(map[openrtb_ext.BidderName]metrics.Meter),

		AdapterMetrics: make(map[openrtb_ext.BidderName]*AdapterMetrics, len(exchanges)),
		accountMetrics: make(map[string]*accountMetrics),

		exchanges: exchanges,
	}
	for _, a := range exchanges {
		newMetrics.AdapterMetrics[a] = makeBlankAdapterMetrics()
	}

	for _, t := range RequestTypes() {
		newMetrics.RequestStatuses[t] = make(map[RequestStatus]metrics.Meter)
		for _, s := range RequestStatuses() {
			newMetrics.RequestStatuses[t][s] = blankMeter
		}
	}

	return newMetrics
}

// NewMetrics creates a new Metrics object with needed metrics defined. In time we may develop to the point
// where Metrics contains all the metrics we might want to record, and then we build the actual
// metrics object to contain only the metrics we are interested in. This would allow for debug
// mode metrics. The code would allways try to record the metrics, but effectively noop if we are
// using a blank meter/timer.
func NewMetrics(registry metrics.Registry, exchanges []openrtb_ext.BidderName) *Metrics {
	newMetrics := NewBlankMetrics(registry, exchanges)
	newMetrics.ConnectionCounter = metrics.GetOrRegisterCounter("active_connections", registry)
	newMetrics.ConnectionAcceptErrorMeter = metrics.GetOrRegisterMeter("connection_accept_errors", registry)
	newMetrics.ConnectionCloseErrorMeter = metrics.GetOrRegisterMeter("connection_close_errors", registry)
	newMetrics.ImpMeter = metrics.GetOrRegisterMeter("imps_requested", registry)
	newMetrics.SafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", registry)
	newMetrics.NoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", registry)
	newMetrics.AppRequestMeter = metrics.GetOrRegisterMeter("app_requests", registry)
	newMetrics.SafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", registry)
	newMetrics.RequestTimer = metrics.GetOrRegisterTimer("request_time", registry)
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
	newMetrics.userSyncSet[unknownBidder] = metrics.GetOrRegisterMeter("usersync.unknown.sets", registry)
	newMetrics.userSyncGDPRPrevent[unknownBidder] = metrics.GetOrRegisterMeter("usersync.unknown.gdpr_prevent", registry)
	return newMetrics
}

// Part of setting up blank metrics, the adapter metrics.
func makeBlankAdapterMetrics() *AdapterMetrics {
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
	for err := range am.ErrorMeters {
		am.ErrorMeters[err] = metrics.GetOrRegisterMeter(fmt.Sprintf("%s.%s.requests.%s", adapterOrAccount, exchange, err), registry)
	}
	if adapterOrAccount != "adapter" {
		am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), registry)
	}
	am.PanicMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests.panic", adapterOrAccount, exchange), registry)
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
	for _, a := range me.exchanges {
		am.adapterMetrics[a] = makeBlankAdapterMetrics()
		registerAdapterMetrics(me.MetricsRegistry, fmt.Sprintf("account.%s", id), string(a), am.adapterMetrics[a])
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
		if labels.Browser == BrowserSafari {
			me.SafariRequestMeter.Mark(1)
			if labels.CookieFlag == CookieFlagNo {
				me.SafariNoCookieMeter.Mark(1)
			}
		}
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

func (me *Metrics) RecordImps(labels Labels, numImps int) {
	me.ImpMeter.Mark(int64(numImps))
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

	aam := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	switch labels.AdapterBids {
	case AdapterBidNone:
		am.NoBidMeter.Mark(1)
		aam.NoBidMeter.Mark(1)
	case AdapterBidPresent:
		am.GotBidsMeter.Mark(1)
		aam.GotBidsMeter.Mark(1)
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
	aam := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	aam.BidsReceivedMeter.Mark(1)

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
	aam := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	aam.PriceHistogram.Update(int64(cpm))
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
	aam := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	aam.RequestTimer.Update(length)
}

// RecordCookieSync implements a part of the MetricsEngine interface. Records a cookie sync request
func (me *Metrics) RecordCookieSync(labels Labels) {
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

func doMark(bidder openrtb_ext.BidderName, meters map[openrtb_ext.BidderName]metrics.Meter) {
	met, ok := meters[bidder]
	if ok {
		met.Mark(1)
	} else {
		meters[unknownBidder].Mark(1)
	}
}
