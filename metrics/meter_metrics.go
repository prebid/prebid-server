package metrics

import (
	"fmt"
	"github.com/prebid/prebid-server/adapters"
	"github.com/rcrowley/go-metrics"
	"sync"
	"time"
)

type DomainMeterMetrics struct {
	RequestMeter metrics.Meter
}

type AccountMeterMetrics struct {
	RequestMeter      metrics.Meter
	BidsReceivedMeter metrics.Meter
	PriceHistogram    metrics.Histogram
	// store account by adapter metrics. Type is map[PBSBidder.BidderCode]
	AdapterMetrics map[string]*AdapterMeterMetrics
}

type AdapterMeterMetrics struct {
	NoCookieMeter     metrics.Meter
	ErrorMeter        metrics.Meter
	NoBidMeter        metrics.Meter
	TimeoutMeter      metrics.Meter
	RequestMeter      metrics.Meter
	RequestTimer      metrics.Timer
	PriceHistogram    metrics.Histogram
	BidsReceivedMeter metrics.Meter
}

type AllMeterMetrics struct {
	exchanges            map[string]adapters.Adapter
	metricsRegistry      metrics.Registry
	mRequestMeter        metrics.Meter
	mAppRequestMeter     metrics.Meter
	mNoCookieMeter       metrics.Meter
	mSafariRequestMeter  metrics.Meter
	mSafariNoCookieMeter metrics.Meter
	mErrorMeter          metrics.Meter
	mInvalidMeter        metrics.Meter
	mRequestTimer        metrics.Timer
	mCookieSyncMeter     metrics.Meter

	adapterMetrics map[string]*AdapterMeterMetrics

	accountMetrics        map[string]*AccountMeterMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex
}

func (m *AllMeterMetrics) Setup(metricsRegistry metrics.Registry, exchanges map[string]adapters.Adapter) {
	m.metricsRegistry = metricsRegistry
	m.exchanges = exchanges
	m.mRequestMeter = metrics.GetOrRegisterMeter("requests", metricsRegistry)
	m.mAppRequestMeter = metrics.GetOrRegisterMeter("app_requests", metricsRegistry)
	m.mNoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", metricsRegistry)
	m.mSafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", metricsRegistry)
	m.mSafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", metricsRegistry)
	m.mErrorMeter = metrics.GetOrRegisterMeter("error_requests", metricsRegistry)
	m.mInvalidMeter = metrics.GetOrRegisterMeter("invalid_requests", metricsRegistry)
	m.mRequestTimer = metrics.GetOrRegisterTimer("request_time", metricsRegistry)
	m.mCookieSyncMeter = metrics.GetOrRegisterMeter("cookie_sync_requests", metricsRegistry)

	m.accountMetrics = make(map[string]*AccountMeterMetrics)
	m.adapterMetrics = m.makeExchangeMetrics("adapter")
}

func (m *AllMeterMetrics) makeExchangeMetrics(adapterOrAccount string) map[string]*AdapterMeterMetrics {
	var adapterMetrics = make(map[string]*AdapterMeterMetrics)
	for exchange := range m.exchanges {
		a := AdapterMeterMetrics{}
		a.NoCookieMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_cookie_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.ErrorMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.error_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.NoBidMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_bid_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.TimeoutMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.timeout_requests", adapterOrAccount, exchange), m.metricsRegistry)
		a.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.request_time", adapterOrAccount, exchange), m.metricsRegistry)
		a.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("%[1]s.%[2]s.prices", adapterOrAccount, exchange), m.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		if adapterOrAccount != "adapter" {
			a.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), m.metricsRegistry)
		}

		adapterMetrics[exchange] = &a
	}
	return adapterMetrics
}

func (m *AllMeterMetrics) GetMyAccountMetrics(id string) AccountMetrics {
	var am *AccountMeterMetrics
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
		am = &AccountMeterMetrics{}
		am.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.requests", id), m.metricsRegistry)
		am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.bids_received", id), m.metricsRegistry)
		am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), m.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		am.AdapterMetrics = m.makeExchangeMetrics(fmt.Sprintf("account.%s", id)) //TODO: Is reinitialization necessary
		m.accountMetrics[id] = am
	}
	m.accountMetricsRWMutex.Unlock()

	return am
}

func (m *AllMeterMetrics) GetMyAdapterMetrics(bidderCode string) AdapterMetrics {
	return m.adapterMetrics[bidderCode]
}

func (m *AllMeterMetrics) UpdateRequestTimerSince(start time.Time) {
	m.mRequestTimer.UpdateSince(start)
}

func (m *AllMeterMetrics) GetMetrics() PBSMetrics {
	return m
}

func (m *AllMeterMetrics) IncRequest(i int64) {
	m.mRequestMeter.Mark(i)
}

func (m *AllMeterMetrics) IncSafariRequest(i int64) {
	m.mSafariRequestMeter.Mark(i)
}

func (m *AllMeterMetrics) IncAppRequest(i int64) {
	m.mAppRequestMeter.Mark(i)
}

func (m *AllMeterMetrics) IncNoCookie(i int64) {
	m.mNoCookieMeter.Mark(i)
}

func (m *AllMeterMetrics) IncSafariNoCookie(i int64) {
	m.mSafariNoCookieMeter.Mark(i)
}

func (m *AllMeterMetrics) IncError(i int64) {
	m.mErrorMeter.Mark(i)
}

func (m *AllMeterMetrics) IncCookieSync(i int64) {
	m.mCookieSyncMeter.Mark(i)
}

func (am *AccountMeterMetrics) GetMyAdapterMetrics(bidderCode string) AdapterMetrics {
	return am.AdapterMetrics[bidderCode]
}

func (am *AccountMeterMetrics) IncBidsReceived(i int64) {
	am.BidsReceivedMeter.Mark(i)
}

func (am *AccountMeterMetrics) IncRequest(i int64) {
	am.RequestMeter.Mark(i)
}

func (am *AccountMeterMetrics) UpdatePriceHistogram(cpm int64) {
	am.PriceHistogram.Update(cpm)
}

func (am *AdapterMeterMetrics) IncNoBid(i int64) {
	am.NoBidMeter.Mark(i)
}

func (am *AdapterMeterMetrics) IncBidsReceived(i int64) {
	am.BidsReceivedMeter.Mark(i)
}

func (am *AdapterMeterMetrics) UpdateRequestTimerSince(start time.Time) {
	am.RequestTimer.UpdateSince(start)
}

func (am *AdapterMeterMetrics) IncError(i int64) {
	am.ErrorMeter.Mark(i)
}

func (am *AdapterMeterMetrics) IncTimeOut(i int64) {
	am.TimeoutMeter.Mark(i)
}

func (am *AdapterMeterMetrics) UpdatePriceHistogram(cpm int64) {
	am.PriceHistogram.Update(cpm)
}

func (am *AdapterMeterMetrics) IncRequest(i int64) {
	am.RequestMeter.Mark(i)
}
func (am *AdapterMeterMetrics) IncNoCookie(i int64) {
	am.NoCookieMeter.Mark(i)
}
