package analytics

import (
	"fmt"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
	"sync"
)

type DomainMeterMetrics struct {
	RequestMeter metrics.Meter
}

type AccountMetrics struct {
	RequestMeter      metrics.Meter
	BidsReceivedMeter metrics.Meter
	PriceHistogram    metrics.Histogram
	// store account by adapter pbsMetrics. Type is map[PBSBidder.BidderCode]
	AdapterMetrics map[openrtb_ext.BidderName]*AdapterMetrics
}

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

type PBSMetrics struct {
	exchanges            []openrtb_ext.BidderName
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

	adapterMetrics map[openrtb_ext.BidderName]*AdapterMetrics

	accountMetrics        map[openrtb_ext.BidderName]*AccountMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex
}

func (m *PBSMetrics) setup(metricsRegistry metrics.Registry, adapters []openrtb_ext.BidderName) {
	m.metricsRegistry = metricsRegistry
	m.exchanges = adapters
	m.mRequestMeter = metrics.GetOrRegisterMeter("requests", metricsRegistry)
	m.mAppRequestMeter = metrics.GetOrRegisterMeter("app_requests", metricsRegistry)
	m.mNoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", metricsRegistry)
	m.mSafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", metricsRegistry)
	m.mSafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", metricsRegistry)
	m.mErrorMeter = metrics.GetOrRegisterMeter("error_requests", metricsRegistry)
	m.mInvalidMeter = metrics.GetOrRegisterMeter("invalid_requests", metricsRegistry)
	m.mRequestTimer = metrics.GetOrRegisterTimer("request_time", metricsRegistry)
	m.mCookieSyncMeter = metrics.GetOrRegisterMeter("cookie_sync_requests", metricsRegistry)

	m.accountMetrics = make(map[openrtb_ext.BidderName]*AccountMetrics)
	m.adapterMetrics = m.makeExchangeMetrics("adapter")
}

func (m *PBSMetrics) makeExchangeMetrics(adapterOrAccount string) map[openrtb_ext.BidderName]*AdapterMetrics {
	var adapterMetrics = make(map[openrtb_ext.BidderName]*AdapterMetrics)
	for _, exchange := range m.exchanges {
		a := AdapterMetrics{}
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

func (m *PBSMetrics) GetMyAccountMetrics(id openrtb_ext.BidderName) *AccountMetrics {
	var am *AccountMetrics
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
		am = &AccountMetrics{}
		am.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.requests", id), m.metricsRegistry)
		am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.bids_received", id), m.metricsRegistry)
		am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), m.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		am.AdapterMetrics = m.makeExchangeMetrics(fmt.Sprintf("account.%s", id)) //TODO: Is reinitialization necessary
		m.accountMetrics[id] = am
	}
	m.accountMetricsRWMutex.Unlock()

	return am
}
