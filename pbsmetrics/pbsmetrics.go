package pbsmetrics

import (
	"fmt"
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
	RequestTimer        metrics.Timer
	// Metrics for OpenRTB requests specifically. So we can track what % of RequestsMeter are OpenRTB
	// and know when legacy requests have been abandoned.
	ORTBRequestMeter metrics.Meter

	AdapterMetrics map[openrtb_ext.BidderName]*AdapterMetrics

	exchanges []openrtb_ext.BidderName
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
		RequestTimer:        blankTimer(0),
		ORTBRequestMeter:    blankMeter(0),

		AdapterMetrics: make(map[openrtb_ext.BidderName]*AdapterMetrics, len(exchanges)),

		exchanges: exchanges,
	}
	for _, a := range exchanges {
		newMetrics.AdapterMetrics[a] = makeBlankAdapterMetrics(registry, a)
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
	newMetrics.SafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", registry)
	newMetrics.ErrorMeter = metrics.GetOrRegisterMeter("error_requests", registry)
	newMetrics.NoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", registry)
	newMetrics.AppRequestMeter = metrics.GetOrRegisterMeter("app_requests", registry)
	newMetrics.SafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", registry)
	newMetrics.RequestTimer = metrics.GetOrRegisterTimer("request_time", registry)
	newMetrics.ORTBRequestMeter = metrics.GetOrRegisterMeter("ortb_requests", registry)

	for _, a := range exchanges {
		registerAdapterMetrics(registry, "adapter", string(a), newMetrics.AdapterMetrics[a])
	}
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
