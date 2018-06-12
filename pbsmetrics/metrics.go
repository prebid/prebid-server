package pbsmetrics

import (
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
)

// Labels defines the labels that can be attached to the metrics.
type Labels struct {
	Source        DemandSource
	RType         RequestType
	PubID         string // exchange specific ID, so we cannot compile in values
	Browser       Browser
	CookieFlag    CookieFlag
	RequestStatus RequestStatus
}

// AdapterLabels defines the labels that can be attached to the adapter metrics.
type AdapterLabels struct {
	Source        DemandSource
	RType         RequestType
	Adapter       openrtb_ext.BidderName
	PubID         string // exchange specific ID, so we cannot compile in values
	Browser       Browser
	CookieFlag    CookieFlag
	AdapterStatus AdapterStatus
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

// RequestStatus : The request return status
type RequestStatus string

// AdapterStatus : The radapter execution status
type AdapterStatus string

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

func requestTypes() []RequestType {
	return []RequestType{
		ReqTypeLegacy,
		ReqTypeORTB2,
		ReqTypeAMP,
	}
}

// Browser flag; at this point we only care about identifying Safari
const (
	BrowserSafari Browser = "safari"
	BrowserOther  Browser = "other"
)

// Cookie flag
const (
	CookieFlagYes     CookieFlag = "exists"
	CookieFlagNo      CookieFlag = "no"
	CookieFlagUnknown CookieFlag = "unknown"
)

// Request/return status
const (
	RequestStatusOK       RequestStatus = "ok"
	RequestStatusBadInput RequestStatus = "badinput"
	RequestStatusErr      RequestStatus = "err"
)

func requestStatuses() []RequestStatus {
	return []RequestStatus{
		RequestStatusOK,
		RequestStatusBadInput,
		RequestStatusErr,
	}
}

// Adapter execution status
const (
	AdapterStatusOK      AdapterStatus = "ok"
	AdapterStatusErr     AdapterStatus = "err"
	AdapterStatusNoBid   AdapterStatus = "nobid"
	AdapterStatusTimeout AdapterStatus = "timeout"
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
	RequestActionGDPR   RequestAction = "gdpr"
	RequestActionErr    RequestAction = "err"
)

// MetricsEngine is a generic interface to record PBS metrics into the desired backend
// The first three metrics function fire off once per incoming request, so total metrics
// will equal the total numer of incoming requests. The remaining 5 fire off per outgoing
// request to a bidder adapter, so will record a number of hits per incoming request. The
// two groups should be consistent within themselves, but comparing numbers between groups
// is generally not useful.
type MetricsEngine interface {
	RecordConnectionAccept(success bool)
	RecordConnectionClose(success bool)
	RecordRequest(labels Labels)                           // ignores adapter. only statusOk and statusErr fom status
	RecordImps(labels Labels, numImps int)                 // ignores adapter. only statusOk and statusErr fom status
	RecordRequestTime(labels Labels, length time.Duration) // ignores adapter. only statusOk and statusErr fom status
	RecordAdapterRequest(labels AdapterLabels)
	RecordAdapterBidsReceived(labels AdapterLabels, bids int64)
	// This records whether or not a bid of a particular type uses `adm` or `nurl`.
	// Since the legacy endpoints don't have a bid type, it can only count bids from OpenRTB and AMP.
	RecordAdapterBidAdm(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool)
	RecordAdapterPrice(labels AdapterLabels, cpm float64)
	RecordAdapterTime(labels AdapterLabels, length time.Duration)
	RecordCookieSync(labels Labels)        // May ignore all labels
	RecordUserIDSet(userLabels UserLabels) // Function should verify bidder values
}

// NewMetricsEngine reads the configuration and returns the appropriate metrics engine
// for this instance.
func NewMetricsEngine(cfg *config.Configuration, adapterList []openrtb_ext.BidderName) MetricsEngine {
	// Create a list of metrics engines to use.
	// Capacity of 2, as unlikely to have more than 2 metrics backends, and in the case
	// of 1 we won't use the list so it will be garbage collected.
	engineList := make(MultiMetricsEngine, 0, 2)

	if cfg.Metrics.Influxdb.Host != "" {
		// Currently use go-metrics as the metrics piece for influx
		goMetrics := NewMetrics(metrics.NewPrefixedRegistry("prebidserver."), adapterList)
		engineList = append(engineList, goMetrics)
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
		return &engineList
	} else if len(engineList) == 1 {
		return engineList[0]
	}
	return &DummyMetricsEngine{}
}

// MultiMetricsEngine logs metrics to multiple metrics databases The can be useful in transitioning
// an instance from one engine to another, you can run both in parallel to verify stats match up.
type MultiMetricsEngine []MetricsEngine

// RecordRequest across all engines
func (me *MultiMetricsEngine) RecordRequest(labels Labels) {
	for _, thisME := range *me {
		thisME.RecordRequest(labels)
	}
}

func (me *MultiMetricsEngine) RecordConnectionAccept(success bool) {
	for _, thisME := range *me {
		thisME.RecordConnectionAccept(success)
	}
}

func (me *MultiMetricsEngine) RecordConnectionClose(success bool) {
	for _, thisME := range *me {
		thisME.RecordConnectionClose(success)
	}
}

// RecordImps across all engines
func (me *MultiMetricsEngine) RecordImps(labels Labels, numImps int) {
	for _, thisME := range *me {
		thisME.RecordImps(labels, numImps)
	}
}

// RecordRequestTime across all engines
func (me *MultiMetricsEngine) RecordRequestTime(labels Labels, length time.Duration) {
	for _, thisME := range *me {
		thisME.RecordRequestTime(labels, length)
	}
}

// RecordAdapterRequest across all engines
func (me *MultiMetricsEngine) RecordAdapterRequest(labels AdapterLabels) {
	for _, thisME := range *me {
		thisME.RecordAdapterRequest(labels)
	}
}

// RecordAdapterBidsReceived across all engines
func (me *MultiMetricsEngine) RecordAdapterBidsReceived(labels AdapterLabels, bids int64) {
	for _, thisME := range *me {
		thisME.RecordAdapterBidsReceived(labels, bids)
	}
}

// RecordAdapterBidAdm across all engines
func (me *MultiMetricsEngine) RecordAdapterBidAdm(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	for _, thisME := range *me {
		thisME.RecordAdapterBidAdm(labels, bidType, hasAdm)
	}
}

// RecordAdapterPrice across all engines
func (me *MultiMetricsEngine) RecordAdapterPrice(labels AdapterLabels, cpm float64) {
	for _, thisME := range *me {
		thisME.RecordAdapterPrice(labels, cpm)
	}
}

// RecordAdapterTime across all engines
func (me *MultiMetricsEngine) RecordAdapterTime(labels AdapterLabels, length time.Duration) {
	for _, thisME := range *me {
		thisME.RecordAdapterTime(labels, length)
	}
}

// RecordCookieSync across all engines
func (me *MultiMetricsEngine) RecordCookieSync(labels Labels) {
	for _, thisME := range *me {
		thisME.RecordCookieSync(labels)
	}
}

// RecordUserIDSet across all engines
func (me *MultiMetricsEngine) RecordUserIDSet(userLabels UserLabels) {
	for _, thisME := range *me {
		thisME.RecordUserIDSet(userLabels)
	}
}

// DummyMetricsEngine is a Noop metrics engine in case no metrics are configured. (may also be useful for tests)
type DummyMetricsEngine struct{}

// RecordRequest as a noop
func (me *DummyMetricsEngine) RecordRequest(labels Labels) {
	return
}

// RecordConnectionAccept as a noop
func (me *DummyMetricsEngine) RecordConnectionAccept(success bool) {
	return
}

// RecordConnectionClose as a noop
func (me *DummyMetricsEngine) RecordConnectionClose(success bool) {
	return
}

// RecordImps as a noop
func (me *DummyMetricsEngine) RecordImps(labels Labels, numImps int) {
	return
}

// RecordRequestTime as a noop
func (me *DummyMetricsEngine) RecordRequestTime(labels Labels, length time.Duration) {
	return
}

// RecordAdapterRequest as a noop
func (me *DummyMetricsEngine) RecordAdapterRequest(labels AdapterLabels) {
	return
}

// RecordAdapterBidsReceived as a noop
func (me *DummyMetricsEngine) RecordAdapterBidsReceived(labels AdapterLabels, bids int64) {
	return
}

// RecordAdapterBidAdm as a noop
func (me *DummyMetricsEngine) RecordAdapterBidAdm(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	return
}

// RecordAdapterPrice as a noop
func (me *DummyMetricsEngine) RecordAdapterPrice(labels AdapterLabels, cpm float64) {
	return
}

// RecordAdapterTime as a noop
func (me *DummyMetricsEngine) RecordAdapterTime(labels AdapterLabels, length time.Duration) {
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
