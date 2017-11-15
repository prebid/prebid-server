package metrics

import (
	"fmt"
	"github.com/prebid/prebid-server/adapters"
	"github.com/rcrowley/go-metrics"
	"sync"
	"time"
)

type DomainCounterMetrics struct {
	RequestCounter metrics.Counter
}

type AccountCounterMetrics struct {
	RequestCounter      metrics.Counter
	BidsReceivedCounter metrics.Counter
	PriceHistogram      metrics.Histogram
	// store account by adapter metrics. Type is map[PBSBidder.BidderCode]
	AdapterMetrics map[string]*AdapterCounterMetrics
}

type AdapterCounterMetrics struct {
	NoCookieCounter     metrics.Counter
	ErrorCounter        metrics.Counter
	NoBidCounter        metrics.Counter
	TimeoutCounter      metrics.Counter
	RequestCounter      metrics.Counter
	RequestTimer        metrics.Timer
	PriceHistogram      metrics.Histogram
	BidsReceivedCounter metrics.Counter
}

type AllCounterMetrics struct {
	exchanges              map[string]adapters.Adapter
	metricsRegistry        metrics.Registry
	mRequestCounter        metrics.Counter
	mAppRequestCounter     metrics.Counter
	mNoCookieCounter       metrics.Counter
	mSafariRequestCounter  metrics.Counter
	mSafariNoCookieCounter metrics.Counter
	mErrorCounter          metrics.Counter
	mInvalidCounter        metrics.Counter
	mRequestTimer          metrics.Timer
	mCookieSyncCounter     metrics.Counter

	adapterMetrics map[string]*AdapterCounterMetrics

	accountMetrics        map[string]*AccountCounterMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex
}

func (m *AllCounterMetrics) Setup(metricsRegistry metrics.Registry, exchanges map[string]adapters.Adapter) {
	m.metricsRegistry = metricsRegistry
	m.exchanges = exchanges
	m.mRequestCounter = metrics.GetOrRegisterCounter("requests", metricsRegistry)
	m.mAppRequestCounter = metrics.GetOrRegisterCounter("app_requests", metricsRegistry)
	m.mNoCookieCounter = metrics.GetOrRegisterCounter("no_cookie_requests", metricsRegistry)
	m.mSafariRequestCounter = metrics.GetOrRegisterCounter("safari_requests", metricsRegistry)
	m.mSafariNoCookieCounter = metrics.GetOrRegisterCounter("safari_no_cookie_requests", metricsRegistry)
	m.mErrorCounter = metrics.GetOrRegisterCounter("error_requests", metricsRegistry)
	m.mInvalidCounter = metrics.GetOrRegisterCounter("invalid_requests", metricsRegistry)
	m.mRequestTimer = metrics.GetOrRegisterTimer("request_time", metricsRegistry)
	m.mCookieSyncCounter = metrics.GetOrRegisterCounter("cookie_sync_requests", metricsRegistry)

	m.accountMetrics = make(map[string]*AccountCounterMetrics)
	m.adapterMetrics = m.makeExchangeMetrics("adapter")
}

func (m *AllCounterMetrics) makeExchangeMetrics(adapterOrAccount string) map[string]*AdapterCounterMetrics {
	var adapterMetrics = make(map[string]*AdapterCounterMetrics)
	for exchange := range m.exchanges {
		a := AdapterCounterMetrics{}
		a.NoCookieCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.no_cookie_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.ErrorCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.error_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.RequestCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.NoBidCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.no_bid_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.TimeoutCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.timeout_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.request_time", adapterOrAccount, exchange), m.metricsRegistry)
		a.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("%[1]s.%[2]s.prices", adapterOrAccount, exchange), m.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		if adapterOrAccount != "adapter" {
			a.BidsReceivedCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), m.metricsRegistry)
		}

		adapterMetrics[exchange] = &a
	}
	return adapterMetrics
}

func (m *AllCounterMetrics) GetMyAccountMetrics(id string) AccountMetrics {
	var am *AccountCounterMetrics
	var ok bool

	m.accountMetricsRWMutex.RLock()
	am, ok = m.accountMetrics[id]
	m.accountMetricsRWMutex.RUnlock()

	if ok {
		return am
	}

	m.accountMetricsRWMutex.Lock()
	am, ok = m.accountMetrics[id]
	if !ok {
		am = &AccountCounterMetrics{}
		am.RequestCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("account.%s.requests", id), m.metricsRegistry)
		am.BidsReceivedCounter = metrics.GetOrRegisterCounter(fmt.Sprintf("account.%s.bids_received", id), m.metricsRegistry)
		am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), m.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		am.AdapterMetrics = m.makeExchangeMetrics(fmt.Sprintf("account.%s", id)) //TODO: Is reinitialization necessary
		m.accountMetrics[id] = am
	}
	m.accountMetricsRWMutex.Unlock()

	return am
}

func (m *AllCounterMetrics) IncRequest(i int64) {
	m.mRequestCounter.Inc(i)
}

func (m *AllCounterMetrics) IncSafariRequest(i int64) {
	m.mSafariRequestCounter.Inc(i)
}

func (m *AllCounterMetrics) IncAppRequest(i int64) {
	m.mAppRequestCounter.Inc(i)
}

func (m *AllCounterMetrics) IncNoCookie(i int64) {
	m.mNoCookieCounter.Inc(i)
}

func (m *AllCounterMetrics) GetMetrics() PBSMetrics {
	return m
}

func (m *AllCounterMetrics) IncSafariNoCookie(i int64) {
	m.mSafariNoCookieCounter.Inc(i)
}

func (m *AllCounterMetrics) IncError(i int64) {
	m.mErrorCounter.Inc(i)
}

func (m *AllCounterMetrics) IncCookieSync(i int64) {
	m.mCookieSyncCounter.Inc(i)
}

func (m *AllCounterMetrics) UpdateRequestTimerSince(start time.Time) {
	m.mRequestTimer.UpdateSince(start)
}

func (m *AllCounterMetrics) GetMyAdapterMetrics(bidderCode string) AdapterMetrics {
	return m.adapterMetrics[bidderCode]
}

func (am *AccountCounterMetrics) GetMyAdapterMetrics(bidderCode string) AdapterMetrics {
	return am.AdapterMetrics[bidderCode]
}

func (am *AccountCounterMetrics) UpdatePriceHistogram(cpm int64) {
	am.PriceHistogram.Update(cpm)
}

func (am *AccountCounterMetrics) IncRequest(i int64) {
	am.RequestCounter.Inc(i)
}

func (am *AccountCounterMetrics) IncBidsReceived(i int64) {
	am.BidsReceivedCounter.Inc(i)
}

func (am *AdapterCounterMetrics) IncBidsReceived(i int64) {
	am.BidsReceivedCounter.Inc(i)
}

func (am *AdapterCounterMetrics) UpdateRequestTimerSince(start time.Time) {
	am.RequestTimer.UpdateSince(start)
}

func (am *AdapterCounterMetrics) IncTimeOut(i int64) {
	am.TimeoutCounter.Inc(i)
}

func (am *AdapterCounterMetrics) IncError(i int64) {
	am.ErrorCounter.Inc(i)
}

func (am *AdapterCounterMetrics) UpdatePriceHistogram(cpm int64) {
	am.PriceHistogram.Update(cpm)
}

func (am *AdapterCounterMetrics) IncNoBid(i int64) {
	am.NoBidCounter.Inc(i)
}

func (am *AdapterCounterMetrics) IncRequest(i int64) {
	am.RequestCounter.Inc(i)
}

func (am *AdapterCounterMetrics) IncNoCookie(i int64) {
	am.NoCookieCounter.Inc(i)
}
