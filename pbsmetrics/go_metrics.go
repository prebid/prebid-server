package pbsmetrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
)

// Metrics is the legacy Metrics object (go-metrics) expanded to also satisfy the MetricsEngine interface
type Metrics struct {
	metricsRegistry     metrics.Registry
	RequestMeter        metrics.Meter
	AppRequestMeter     metrics.Meter
	NoCookieMeter       metrics.Meter
	SafariRequestMeter  metrics.Meter
	SafariNoCookieMeter metrics.Meter
	ErrorMeter          metrics.Meter
	RequestTimer        metrics.Timer
	// Metrics for OpenRTB requests specifically. So we can track what % of RequestsMeter are OpenRTB
	// and know when legacy requests have been abandoned.
	ORTBRequestMeter   metrics.Meter
	AmpRequestMeter    metrics.Meter
	AmpNoCookieMeter   metrics.Meter
	CookieSyncMeter    metrics.Meter
	userSyncOptout     metrics.Meter
	userSyncBadRequest metrics.Meter
	userSyncSet        map[openrtb_ext.BidderName]metrics.Meter

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
	ErrorMeter        metrics.Meter
	NoBidMeter        metrics.Meter
	TimeoutMeter      metrics.Meter
	RequestMeter      metrics.Meter
	RequestTimer      metrics.Timer
	PriceHistogram    metrics.Histogram
	BidsReceivedMeter metrics.Meter
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
func NewBlankMetrics(registry metrics.Registry, exchanges []openrtb_ext.BidderName) *Metrics {
	newMetrics := &Metrics{
		metricsRegistry:     registry,
		RequestMeter:        blankMeter(0),
		AppRequestMeter:     blankMeter(0),
		NoCookieMeter:       blankMeter(0),
		SafariRequestMeter:  blankMeter(0),
		SafariNoCookieMeter: blankMeter(0),
		ErrorMeter:          blankMeter(0),
		RequestTimer:        blankTimer(0),
		ORTBRequestMeter:    blankMeter(0),
		AmpRequestMeter:     blankMeter(0),
		AmpNoCookieMeter:    blankMeter(0),
		CookieSyncMeter:     blankMeter(0),
		userSyncOptout:      blankMeter(0),
		userSyncBadRequest:  blankMeter(0),
		userSyncSet:         make(map[openrtb_ext.BidderName]metrics.Meter),

		AdapterMetrics: make(map[openrtb_ext.BidderName]*AdapterMetrics, len(exchanges)),
		accountMetrics: make(map[string]*accountMetrics),

		exchanges: exchanges,
	}
	for _, a := range exchanges {
		newMetrics.AdapterMetrics[a] = makeBlankAdapterMetrics(registry, a)
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
	newMetrics.RequestMeter = metrics.GetOrRegisterMeter("requests", registry)
	newMetrics.SafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", registry)
	newMetrics.ErrorMeter = metrics.GetOrRegisterMeter("error_requests", registry)
	newMetrics.NoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", registry)
	newMetrics.AppRequestMeter = metrics.GetOrRegisterMeter("app_requests", registry)
	newMetrics.SafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", registry)
	newMetrics.RequestTimer = metrics.GetOrRegisterTimer("request_time", registry)
	newMetrics.ORTBRequestMeter = metrics.GetOrRegisterMeter("ortb_requests", registry)
	newMetrics.AmpRequestMeter = metrics.GetOrRegisterMeter("amp_requests", registry)
	newMetrics.AmpNoCookieMeter = metrics.GetOrRegisterMeter("amp_no_cookie_requests", registry)
	newMetrics.CookieSyncMeter = metrics.GetOrRegisterMeter("cookie_sync_requests", registry)
	newMetrics.userSyncBadRequest = metrics.GetOrRegisterMeter("usersync.bad_requests", registry)
	newMetrics.userSyncOptout = metrics.GetOrRegisterMeter("usersync.opt_outs", registry)
	for _, a := range exchanges {
		newMetrics.userSyncSet[a] = metrics.GetOrRegisterMeter(fmt.Sprintf("usersync.%s.sets", string(a)), registry)
		registerAdapterMetrics(registry, "adapter", string(a), newMetrics.AdapterMetrics[a])
	}
	newMetrics.userSyncSet[unknownBidder] = metrics.GetOrRegisterMeter("usersync.unknown.sets", registry)
	return newMetrics
}

// Part of setting up blank metrics, the adapter metrics.
func makeBlankAdapterMetrics(registry metrics.Registry, exchanges openrtb_ext.BidderName) *AdapterMetrics {
	newAdapter := &AdapterMetrics{
		NoCookieMeter:     blankMeter(0),
		ErrorMeter:        blankMeter(0),
		NoBidMeter:        blankMeter(0),
		TimeoutMeter:      blankMeter(0),
		RequestMeter:      blankMeter(0),
		RequestTimer:      blankTimer(0),
		PriceHistogram:    blankHistogram(0),
		BidsReceivedMeter: blankMeter(0),
	}
	return newAdapter
}

func registerAdapterMetrics(registry metrics.Registry, adapterOrAccount string, exchange string, am *AdapterMetrics) {
	am.NoCookieMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_cookie_requests", adapterOrAccount, exchange), registry)
	am.ErrorMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.error_requests", adapterOrAccount, exchange), registry)
	am.NoBidMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_bid_requests", adapterOrAccount, exchange), registry)
	am.TimeoutMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.timeout_requests", adapterOrAccount, exchange), registry)
	am.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests", adapterOrAccount, exchange), registry)
	am.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.request_time", adapterOrAccount, exchange), registry)
	am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("%[1]s.%[2]s.prices", adapterOrAccount, exchange), registry, metrics.NewExpDecaySample(1028, 0.015))
	if adapterOrAccount != "adapter" {
		am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), registry)
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
	am.requestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.requests", id), me.metricsRegistry)
	am.bidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.bids_received", id), me.metricsRegistry)
	am.priceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), me.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
	am.adapterMetrics = make(map[openrtb_ext.BidderName]*AdapterMetrics, len(me.exchanges))
	for _, a := range me.exchanges {
		am.adapterMetrics[a] = makeBlankAdapterMetrics(me.metricsRegistry, a)
		registerAdapterMetrics(me.metricsRegistry, fmt.Sprintf("account.%s", id), string(a), am.adapterMetrics[a])
	}

	me.accountMetrics[id] = am

	return am
}

// Implement the MetricsEngine interface

// RecordRequest implements a part of the MetricsEngine interface
func (me *Metrics) RecordRequest(labels Labels) {
	me.RequestMeter.Mark(1)
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
	switch labels.RType {
	case ReqTypeORTB2:
		me.ORTBRequestMeter.Mark(1)
	case ReqTypeAMP:
		me.AmpRequestMeter.Mark(1)
	}
	if labels.RequestStatus == RequestStatusErr {
		me.ErrorMeter.Mark(1)
	}
	// Handle the account metrics now.
	am := me.getAccountMetrics(labels.PubID)
	am.requestMeter.Mark(1)
}

// RecordRequestTime implements a part of the MetricsEngine interface. The calling code is responsible
// for determining the call duration.
func (me *Metrics) RecordRequestTime(labels Labels, length time.Duration) {
	// Only record times for successful requests, as we don't have labels to screen out bad requests.
	if labels.RequestStatus == RequestStatusOK {
		me.RequestTimer.Update(length)
	}
}

// RecordAdapterRequest implements a part of the MetricsEngine interface
func (me *Metrics) RecordAdapterRequest(labels AdapterLabels) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}
	// Adapter metrics
	am.RequestMeter.Mark(1)
	// Account-Adapter metrics
	aam := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	aam.RequestMeter.Mark(1)

	switch labels.AdapterStatus {
	case AdapterStatusErr:
		am.ErrorMeter.Mark(1)
	case AdapterStatusNoBid:
		am.NoBidMeter.Mark(1)
	case AdapterStatusTimeout:
		am.TimeoutMeter.Mark(1)
	}
	if labels.CookieFlag == CookieFlagNo {
		am.NoCookieMeter.Mark(1)
	}
}

// RecordAdapterBidsReceived implements a part of the MetricsEngine interface. This tracks the number of bids received
// from a bidder.
func (me *Metrics) RecordAdapterBidsReceived(labels AdapterLabels, bids int64) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter metrics on %s: adapter metrics not found", string(labels.Adapter))
		return
	}
	// Adapter metrics
	am.BidsReceivedMeter.Mark(bids)
	// Account-Adapter metrics
	aam := me.getAccountMetrics(labels.PubID).adapterMetrics[labels.Adapter]
	aam.BidsReceivedMeter.Mark(bids)
}

// RecordAdapterPrice implements a part of the MetricsEngine interface. Generates a histogram of winning bid prices
func (me *Metrics) RecordAdapterPrice(labels AdapterLabels, cpm float64) {
	am, ok := me.AdapterMetrics[labels.Adapter]
	if !ok {
		glog.Errorf("Trying to run adapter metrics on %s: adapter metrics not found", string(labels.Adapter))
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
		glog.Errorf("Trying to run adapter metrics on %s: adapter metrics not found", string(labels.Adapter))
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

// RecordUserIDSet implements a part of the MetricsEngine interface. Records a cookie setuid request
func (me *Metrics) RecordUserIDSet(userLabels UserLabels) {
	switch userLabels.Action {
	case RequestActionOptOut:
		me.userSyncOptout.Mark(1)
		return
	case RequestActionErr:
		me.userSyncBadRequest.Mark(1)
		return
	case RequestActionSet:
		met, ok := me.userSyncSet[userLabels.Bidder]
		if ok {
			met.Mark(1)
		} else {
			me.userSyncSet[unknownBidder].Mark(1)
		}

	}
}

// Set up blank metrics objects so we can add/subtract active metrics without refactoring a lot of code.
// This will be useful when removing endpoints, we can just run will the blank metrics function
// rather than loading legacy metrics that never get filled.
// This will also eventually let us configure metrics, such as setting a limited set of metrics
// for a production instance, and then expanding again when we need more debugging.

// A blank metrics Meter type
type blankMeter int

func (m blankMeter) Count() int64 {
	return 0
}

func (m blankMeter) Mark(i int64) {
	return
}

func (m blankMeter) Rate1() float64 {
	return 0.0
}

func (m blankMeter) Rate5() float64 {
	return 0.0
}

func (m blankMeter) Rate15() float64 {
	return 0.0
}

func (m blankMeter) RateMean() float64 {
	return 0.0
}

func (m blankMeter) Snapshot() metrics.Meter {
	return m
}

func (m blankMeter) Stop() {
	return
}

// A blank metrics Timer type
type blankTimer int

func (t blankTimer) Count() int64 {
	return 0
}

func (t blankTimer) Max() int64 {
	return 0
}

func (t blankTimer) Mean() float64 {
	return 0.0
}

func (t blankTimer) Min() int64 {
	return 0
}

func (t blankTimer) Percentile(p float64) float64 {
	return 0.0
}

func (t blankTimer) Percentiles(p []float64) []float64 {
	return p
}

func (t blankTimer) Rate1() float64 {
	return 0.0
}

func (t blankTimer) Rate5() float64 {
	return 0.0
}

func (t blankTimer) Rate15() float64 {
	return 0.0
}

func (t blankTimer) RateMean() float64 {
	return 0.0
}

func (t blankTimer) Snapshot() metrics.Timer {
	return t
}

func (t blankTimer) StdDev() float64 {
	return 0.0
}

func (t blankTimer) Stop() {
	return
}

func (t blankTimer) Sum() int64 {
	return 0
}

func (t blankTimer) Time(f func()) {
	return
}

func (t blankTimer) Update(tt time.Duration) {
	return
}

func (t blankTimer) UpdateSince(time.Time) {
	return
}

func (t blankTimer) Variance() float64 {
	return 0.0
}

// a blank metrics Histogram type
type blankHistogram int

func (h blankHistogram) Clear() {
	return
}

func (h blankHistogram) Count() int64 {
	return 0
}

func (h blankHistogram) Max() int64 {
	return 0
}

func (h blankHistogram) Mean() float64 {
	return 0.0
}

func (h blankHistogram) Min() int64 {
	return 0
}

func (h blankHistogram) Percentile(f float64) float64 {
	return 0.0
}

func (h blankHistogram) Percentiles(p []float64) []float64 {
	return p
}

func (h blankHistogram) Sample() metrics.Sample {
	return blankSample(0)
}

func (h blankHistogram) Snapshot() metrics.Histogram {
	return h
}

func (h blankHistogram) StdDev() float64 {
	return 0.0
}

func (h blankHistogram) Sum() int64 {
	return 0
}

func (h blankHistogram) Update(int64) {
	return
}

func (h blankHistogram) Variance() float64 {
	return 0.0
}

// Need a blank sample for the Histogram
type blankSample int

func (h blankSample) Clear() {
	return
}

func (h blankSample) Count() int64 {
	return 0
}

func (h blankSample) Max() int64 {
	return 0
}

func (h blankSample) Mean() float64 {
	return 0.0
}

func (h blankSample) Min() int64 {
	return 0
}

func (h blankSample) Percentile(f float64) float64 {
	return 0.0
}

func (h blankSample) Percentiles(p []float64) []float64 {
	return p
}

func (h blankSample) Size() int {
	return 0
}

func (h blankSample) Snapshot() metrics.Sample {
	return h
}

func (h blankSample) StdDev() float64 {
	return 0.0
}

func (h blankSample) Sum() int64 {
	return 0
}

func (h blankSample) Update(int64) {
	return
}

func (h blankSample) Values() []int64 {
	return []int64{}
}

func (h blankSample) Variance() float64 {
	return 0.0
}
