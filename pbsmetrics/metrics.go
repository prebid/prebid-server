package pbsmetrics

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
	"strings"
	"time"
)

// Labels defines the labels that can be attached to the metrics.
type Labels struct {
	Source        DemandSource
	RType         RequestType
	Adapter       openrtb_ext.BidderName
	PubID         string // exchange specific ID, so we cannot compile in values
	Browser       Browser
	CookieFlag    CookieFlag
	RequestStatus RequestStatus
}

// Label typecasting. Se below the type definitions for possible values

// DemandSource : Demand source enumeration
type DemandSource string

// RequestType : Request type enumeration
type RequestType string

// Browser type enumeration
type Browser string

// CookieFlag : User ID cookie exists flag
type CookieFlag string

// RequestStatus : The request/adapter return status
type RequestStatus string

// The demand sources
const (
	DemandWeb     DemandSource = "web"
	DemandApp     DemandSource = "app"
	DemandUnknown DemandSource = "unknown"
)

// The request types (endpoints)
const (
	ReqTypeLegacy RequestType = "legacy"
	ReqTypeORTB2  RequestType = "openrtb2"
	ReqTypeAMP    RequestType = "amp"
)

// Browser flag; at this point we only care about identifying Safari
const (
	BrowserSafari Browser = "safari"
	BrowserOther  Browser = "other"
)

// Cookie flag
const (
	CookieFlagYes     CookieFlag = "2"
	CookieFlagNo      CookieFlag = "1"
	CookieFlagUnknown CookieFlag = "0"
)

// Request/return status
const (
	RequestStatusOK      RequestStatus = "ok"
	RequestStatusErr     RequestStatus = "err"
	RequestStatusNoBid   RequestStatus = "nobid"   // Only for adapters
	RequestStatusTimeout RequestStatus = "timeout" // Only for adapters
)

// UserLabels : Labels for /setuid endpoint
type UserLabels struct {
	Action RequestAction
	Bidder openrtb_ext.BidderName
}

// RequestAction : The setuid request result
type RequestAction string

// /setuid action labels
const (
	RequestActionSet    RequestAction = "set"
	RequestActionOptOut RequestAction = "opt_out"
	RequestActionErr    RequestAction = "err"
)

// MetricsEngine is a generic interface to record PBS metrics into the desired backend
// The first three metrics function fire off once per incoming request, so total metrics
// will equal the total numer of incoming requests. The remaining 5 fire off per outgoing
// request to a bidder adapter, so will record a number of hits per incoming request. The
// two groups should be consistent within themselves, but comparing numbers between groups
// is generally not useful.
type MetricsEngine interface {
	RecordRequest(labels Labels)                    // ignores adapter. only statusOk and statusErr fom status
	RecordTime(labels Labels, length time.Duration) // ignores adapter. only statusOk and statusErr fom status
	RecordAdapterRequest(labels Labels)
	RecordAdapterBidsReceived(labels Labels, bids int64)
	RecordAdapterPrice(labels Labels, cpm float64)
	RecordAdapterTime(labels Labels, length time.Duration)
	RecordCookieSync(labels Labels)        // May ignore all labels
	RecordUserIDSet(userLabels UserLabels) // Function should verify bidder values
}

// NewMetricsEngine reads the configuration and returns the appropriate metrics engine
// for this instance.
func NewMetricsEngine(cfg *config.Configuration, adapterList []openrtb_ext.BidderName) MetricsEngine {
	// Create a list of metrics engines to use.
	// Capacity of 2, as unlikely to have more than 2 metrics backends, and in the case
	// of 1 we won't use the list so it will be garbage collected.
	engineList := make([]MetricsEngine, 0, 2)

	// Initializze the go metrics package seperately in case we get a second backend that uses it.
	// NOTE: it is currently not an error to initialize go-metrics, but not export the data to a backend.
	var goMetrics *Metrics
	goME := strings.ToLower(cfg.Metrics.GoMetrics.Enabled)
	if len(goME) > 0 && goME != "no" && goME != "false" {
		goMetrics := NewMetrics(metrics.NewPrefixedRegistry("prebidserver."), adapterList)
		engineList = append(engineList, goMetrics)
	}

	if cfg.Metrics.Influxdb.Host != "" {
		// Seperate check in case we find another metrics library we want to hook to InfluxDB
		// If we want to support tagging, then we will need something other than the legacy goMetrics
		if goMetrics == nil {
			panic("Configuration error: InfluxDB turned on, but go-metrics is not enabled")
		}
		// Set up the Influx logger
		go influxdb.InfluxDB(
			goMetrics.metricsRegistry,     // metrics registry
			time.Second*10,                // interval
			cfg.Metrics.Influxdb.Host,     // the InfluxDB url
			cfg.Metrics.Influxdb.Database, // your InfluxDB database
			cfg.Metrics.Influxdb.Username, // your InfluxDB user
			cfg.Metrics.Influxdb.Password, // your InfluxDB password
		)
		// Influx is not added to the engine list as goMetrics takes care of it already.
	}

	// Now return the proper metrics engine
	if len(engineList) > 1 {
		return &MultiMetricsEngine{engineList: engineList}
	} else if len(engineList) == 1 {
		return engineList[0]
	}
	return &DummyMetricsEngine{0}
}

// MultiMetricsEngine logs metrics to multiple metrics databases The can be useful in transitioning
// an instance from one engine to another, you can run both in parallel to verify stats match up.
type MultiMetricsEngine struct {
	engineList []MetricsEngine
}

// RecordRequest across all engines
func (me *MultiMetricsEngine) RecordRequest(labels Labels) {
	for _, thisME := range me.engineList {
		thisME.RecordRequest(labels)
	}
}

// RecordTime across all engines
func (me *MultiMetricsEngine) RecordTime(labels Labels, length time.Duration) {
	for _, thisME := range me.engineList {
		thisME.RecordTime(labels, length)
	}
}

// RecordAdapterRequest across all engines
func (me *MultiMetricsEngine) RecordAdapterRequest(labels Labels) {
	for _, thisME := range me.engineList {
		thisME.RecordAdapterRequest(labels)
	}
}

// RecordAdapterBidsReceived across all engines
func (me *MultiMetricsEngine) RecordAdapterBidsReceived(labels Labels, bids int64) {
	for _, thisME := range me.engineList {
		thisME.RecordAdapterBidsReceived(labels, bids)
	}
}

// RecordAdapterPrice across all engines
func (me *MultiMetricsEngine) RecordAdapterPrice(labels Labels, cpm float64) {
	for _, thisME := range me.engineList {
		thisME.RecordAdapterPrice(labels, cpm)
	}
}

// RecordAdapterTime across all engines
func (me *MultiMetricsEngine) RecordAdapterTime(labels Labels, length time.Duration) {
	for _, thisME := range me.engineList {
		thisME.RecordAdapterTime(labels, length)
	}
}

// RecordCookieSync across all engines
func (me *MultiMetricsEngine) RecordCookieSync(labels Labels) {
	for _, thisME := range me.engineList {
		thisME.RecordCookieSync(labels)
	}
}

// RecordUserIDSet across all engines
func (me *MultiMetricsEngine) RecordUserIDSet(userLabels UserLabels) {
	for _, thisME := range me.engineList {
		thisME.RecordUserIDSet(userLabels)
	}
}

// DummyMetricsEngine is a Noop metrics engine in case no metrics are configured. (may also be useful for tests)
type DummyMetricsEngine struct {
	dummy int
}

// RecordRequest as a noop
func (me *DummyMetricsEngine) RecordRequest(labels Labels) {
	return
}

// RecordTime as a noop
func (me *DummyMetricsEngine) RecordTime(labels Labels, length time.Duration) {
	return
}

// RecordAdapterRequest as a noop
func (me *DummyMetricsEngine) RecordAdapterRequest(labels Labels) {
	return
}

// RecordAdapterBidsReceived as a noop
func (me *DummyMetricsEngine) RecordAdapterBidsReceived(labels Labels, bids int64) {
	return
}

// RecordAdapterPrice as a noop
func (me *DummyMetricsEngine) RecordAdapterPrice(labels Labels, cpm float64) {
	return
}

// RecordAdapterTime as a noop
func (me *DummyMetricsEngine) RecordAdapterTime(labels Labels, length time.Duration) {
	return
}

// RecordCookieSync as a noop
func (me *DummyMetricsEngine) RecordCookieSync(labels Labels) {
	return
}

// RecordUserIDSet as a noop
func (me *DummyMetricsEngine) RecordUserIDSet(userLabels UserLabels) {
	return
}
