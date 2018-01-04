package pbsmetrics

import (
	"sync"
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
)

type Metrics struct {
	metricsRegistry     metrics.Registry
	RequestMeter        metrics.Meter
	AppRequestMeter     metrics.Meter
	NoCookieMeter       metrics.Meter
	SafariRequestMeter  metrics.Meter
	SafariNoCookieMeter metrics.Meter
	ErrorMeter          metrics.Meter
	InvalidMeter        metrics.Meter
	RequestTimer        metrics.Timer
	CookieSyncMeter     metrics.Meter
	// UserSyncMetrics     *UserSyncMetrics

	// AdapterMetrics map[string]*AdapterMetrics

	// accountMetrics        map[string]*AccountMetrics
	accountMetricsRWMutex sync.RWMutex

	exchanges []openrtb_ext.BidderName
}

// Create a new Metrics object with all blank metrics object. This may also be useful for
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
		InvalidMeter:        blankMeter(0),
		RequestTimer:        blankTimer(0),
		CookieSyncMeter:     blankMeter(0),

		// AdapterMetrics: make(map[string]*AdapterMetrics, len(exchanges)),

		// accountMetrics: make(map[string]*AccountMetrics),

		exchanges: exchanges,
	}

	return newMetrics
}

// Create a new Metrics object with needed metrics defined. In time we may develop to the point
// where Metrics contains all the metrics we might want to record, and then we build the actual
// metrics object to contain only the metrics we are interested in. This would allow for debug
// mode metrics. The code would allways try to record the metrics, but effectively noop if we are
// using a blank meter/timer.

func NewMetrics(registry metrics.Registry, exchanges []openrtb_ext.BidderName) *Metrics {
	newMetrics := NewBlankMetrics(registry, exchanges)
	newMetrics.RequestMeter = metrics.GetOrRegisterMeter("requests", registry)

	return newMetrics
}

// Set up blank metrics objects so we can add/subtract active metrics without refactoring a lot of code.
// This will be useful when removing endpoints, we can just run will the blank metrics function
// rather than loading legacy metrics that never get filled.
// This will also eventually let us configure metrics, such as setting a limited set of metrics
// for a production instance, and then expanding again when we need more debugging.

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
