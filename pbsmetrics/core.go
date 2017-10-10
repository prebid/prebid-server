package pbsmetrics

import (
	"sync"
	"github.com/rcrowley/go-metrics"
	"fmt"
	"time"
	"github.com/prebid/prebid-server/config"
	"github.com/vrischmann/go-metrics-influxdb"
)

const (
	USERSYNC_OPT_OUT     = "usersync.opt_outs"
	USERSYNC_BAD_REQUEST = "usersync.bad_requests"
	USERSYNC_SUCCESS     = "usersync.%s.sets"
)

type DomainMetrics struct {
	RequestMeter metrics.Meter
}

type AccountMetrics struct {
	RequestMeter      metrics.Meter
	BidsReceivedMeter metrics.Meter
	PriceHistogram    metrics.Histogram
	// store account by adapter metrics. Type is map[PBSBidder.BidderCode]
	AdapterMetrics map[string]*AdapterMetrics
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

type UserSyncMetrics struct {
	registry        metrics.Registry
	BadRequestMeter metrics.Meter
	OptOutMeter     metrics.Meter
	successMeters   *sync.Map  // This is a *map[string]metrics.Meter
}

func (u *UserSyncMetrics) SuccessMeter(bidder string) metrics.Meter {
	meter, loaded := u.successMeters.LoadOrStore(bidder, metrics.NewMeter())
	if !loaded {
		u.registry.Register(fmt.Sprintf(USERSYNC_SUCCESS, bidder), meter)
	}
	return meter.(metrics.Meter)
}

type Metrics struct{
	metricsRegistry      metrics.Registry
	RequestMeter        metrics.Meter
	AppRequestMeter     metrics.Meter
	NoCookieMeter       metrics.Meter
	SafariRequestMeter  metrics.Meter
	SafariNoCookieMeter metrics.Meter
	ErrorMeter          metrics.Meter
	InvalidMeter        metrics.Meter
	RequestTimer        metrics.Timer
	CookieSyncMeter     metrics.Meter
	UserSyncMetrics     *UserSyncMetrics

	AdapterMetrics      map[string]*AdapterMetrics

	accountMetrics        map[string]*AccountMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex

	exchanges []string
}

// Export begins exporting all the metrics to the database. This blocks indefinitely, so it should
// probably be run inside a goroutine.
func (m *Metrics) Export(cfg *config.Configuration) {
	influxdb.InfluxDB(
		m.metricsRegistry,      // metrics registry
		time.Second*10,         // interval
		cfg.Metrics.Host,       // the InfluxDB url
		cfg.Metrics.Database,   // your InfluxDB database
		cfg.Metrics.Username,   // your InfluxDB user
		cfg.Metrics.Password,   // your InfluxDB password
	)
}

func (m *Metrics) GetAccountMetrics(id string) *AccountMetrics {
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
		am.AdapterMetrics = makeExchangeMetrics(fmt.Sprintf("account.%s", id), m.exchanges, m.metricsRegistry)
		m.accountMetrics[id] = am
	}
	m.accountMetricsRWMutex.Unlock()

	return am
}

func NewMetrics(exchanges []string) *Metrics {
	registry := metrics.NewPrefixedRegistry("prebidserver.")
	return &Metrics{
		metricsRegistry: registry,
		RequestMeter: metrics.GetOrRegisterMeter("requests", registry),
		AppRequestMeter: metrics.GetOrRegisterMeter("app_requests", registry),
		NoCookieMeter: metrics.GetOrRegisterMeter("no_cookie_requests", registry),
		SafariRequestMeter: metrics.GetOrRegisterMeter("safari_requests", registry),
		SafariNoCookieMeter: metrics.GetOrRegisterMeter("safari_no_cookie_requests", registry),
		ErrorMeter: metrics.GetOrRegisterMeter("error_requests", registry),
		InvalidMeter: metrics.GetOrRegisterMeter("invalid_requests", registry),
		RequestTimer: metrics.GetOrRegisterTimer("request_time", registry),
		CookieSyncMeter: metrics.GetOrRegisterMeter("cookie_sync_requests", registry),
		AdapterMetrics: makeExchangeMetrics("adapter", exchanges, registry),
		UserSyncMetrics: &UserSyncMetrics{
			registry: registry,
			BadRequestMeter: metrics.GetOrRegisterMeter(USERSYNC_BAD_REQUEST, registry),
			OptOutMeter: metrics.GetOrRegisterMeter(USERSYNC_OPT_OUT, registry),
			successMeters: &sync.Map{},
		},

		accountMetrics: make(map[string]*AccountMetrics),
		exchanges: exchanges,
	}
}

func makeExchangeMetrics(adapterOrAccount string, exchanges []string, registry metrics.Registry) map[string]*AdapterMetrics {
	var adapterMetrics = make(map[string]*AdapterMetrics)
	for _, exchange := range exchanges {
		a := AdapterMetrics{}
		a.NoCookieMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_cookie_requests", adapterOrAccount, exchange), registry)
		a.ErrorMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.error_requests", adapterOrAccount, exchange), registry)
		a.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests", adapterOrAccount, exchange), registry)
		a.NoBidMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_bid_requests", adapterOrAccount, exchange), registry)
		a.TimeoutMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.timeout_requests", adapterOrAccount, exchange), registry)
		a.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.request_time", adapterOrAccount, exchange), registry)
		a.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("%[1]s.%[2]s.prices", adapterOrAccount, exchange), registry, metrics.NewExpDecaySample(1028, 0.015))
		if adapterOrAccount != "adapter" {
			a.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), registry)
		}

		adapterMetrics[exchange] = &a
	}
	return adapterMetrics
}
