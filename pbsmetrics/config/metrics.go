package config

import (
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	prometheusmetrics "github.com/prebid/prebid-server/pbsmetrics/prometheus"
	metrics "github.com/rcrowley/go-metrics"
	influxdb "github.com/vrischmann/go-metrics-influxdb"
)

// NewMetricsEngine reads the configuration and returns the appropriate metrics engine
// for this instance.
func NewMetricsEngine(cfg *config.Configuration, adapterList []openrtb_ext.BidderName) *DetailedMetricsEngine {
	// Create a list of metrics engines to use.
	// Capacity of 2, as unlikely to have more than 2 metrics backends, and in the case
	// of 1 we won't use the list so it will be garbage collected.
	engineList := make(MultiMetricsEngine, 0, 2)
	returnEngine := DetailedMetricsEngine{}

	if cfg.Metrics.Influxdb.Host != "" {
		// Currently use go-metrics as the metrics piece for influx
		returnEngine.GoMetrics = pbsmetrics.NewMetrics(metrics.NewPrefixedRegistry("prebidserver."), adapterList)
		engineList = append(engineList, returnEngine.GoMetrics)
		// Set up the Influx logger
		go influxdb.InfluxDB(
			returnEngine.GoMetrics.MetricsRegistry, // metrics registry
			time.Second*10,                         // interval
			cfg.Metrics.Influxdb.Host,              // the InfluxDB url
			cfg.Metrics.Influxdb.Database,          // your InfluxDB database
			cfg.Metrics.Influxdb.Username,          // your InfluxDB user
			cfg.Metrics.Influxdb.Password,          // your InfluxDB password
		)
		// Influx is not added to the engine list as goMetrics takes care of it already.
	}
	if cfg.Metrics.Prometheus.Port != 0 {
		// Set up the Prometheus metrics.
		returnEngine.PrometheusMetrics = prometheusmetrics.NewMetrics(cfg.Metrics.Prometheus)
		engineList = append(engineList, returnEngine.PrometheusMetrics)
	}

	// Now return the proper metrics engine
	if len(engineList) > 1 {
		returnEngine.MetricsEngine = &engineList
	} else if len(engineList) == 1 {
		returnEngine.MetricsEngine = engineList[0]
	} else {
		returnEngine.MetricsEngine = &DummyMetricsEngine{}
	}

	return &returnEngine
}

// DetailedMetricsEngine is a MultiMetricsEngine that preserves links to unerlying metrics engines.
type DetailedMetricsEngine struct {
	pbsmetrics.MetricsEngine
	GoMetrics         *pbsmetrics.Metrics
	PrometheusMetrics *prometheusmetrics.Metrics
}

// MultiMetricsEngine logs metrics to multiple metrics databases The can be useful in transitioning
// an instance from one engine to another, you can run both in parallel to verify stats match up.
type MultiMetricsEngine []pbsmetrics.MetricsEngine

// RecordRequest across all engines
func (me *MultiMetricsEngine) RecordRequest(labels pbsmetrics.Labels) {
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
func (me *MultiMetricsEngine) RecordImps(labels pbsmetrics.Labels, numImps int) {
	for _, thisME := range *me {
		thisME.RecordImps(labels, numImps)
	}
}

// RecordRequestTime across all engines
func (me *MultiMetricsEngine) RecordRequestTime(labels pbsmetrics.Labels, length time.Duration) {
	for _, thisME := range *me {
		thisME.RecordRequestTime(labels, length)
	}
}

// RecordAdapterPanic across all engines
func (me *MultiMetricsEngine) RecordAdapterPanic(labels pbsmetrics.AdapterLabels) {
	for _, thisME := range *me {
		thisME.RecordAdapterPanic(labels)
	}
}

// RecordAdapterRequest across all engines
func (me *MultiMetricsEngine) RecordAdapterRequest(labels pbsmetrics.AdapterLabels) {
	for _, thisME := range *me {
		thisME.RecordAdapterRequest(labels)
	}
}

// RecordAdapterBidReceived across all engines
func (me *MultiMetricsEngine) RecordAdapterBidReceived(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	for _, thisME := range *me {
		thisME.RecordAdapterBidReceived(labels, bidType, hasAdm)
	}
}

// RecordAdapterPrice across all engines
func (me *MultiMetricsEngine) RecordAdapterPrice(labels pbsmetrics.AdapterLabels, cpm float64) {
	for _, thisME := range *me {
		thisME.RecordAdapterPrice(labels, cpm)
	}
}

// RecordAdapterTime across all engines
func (me *MultiMetricsEngine) RecordAdapterTime(labels pbsmetrics.AdapterLabels, length time.Duration) {
	for _, thisME := range *me {
		thisME.RecordAdapterTime(labels, length)
	}
}

// RecordCookieSync across all engines
func (me *MultiMetricsEngine) RecordCookieSync(labels pbsmetrics.Labels) {
	for _, thisME := range *me {
		thisME.RecordCookieSync(labels)
	}
}

// RecordAdapterCookieSync across all engines
func (me *MultiMetricsEngine) RecordAdapterCookieSync(adapter openrtb_ext.BidderName, gdprBlocked bool) {
	for _, thisME := range *me {
		thisME.RecordAdapterCookieSync(adapter, gdprBlocked)
	}
}

// RecordUserIDSet across all engines
func (me *MultiMetricsEngine) RecordUserIDSet(userLabels pbsmetrics.UserLabels) {
	for _, thisME := range *me {
		thisME.RecordUserIDSet(userLabels)
	}
}

// DummyMetricsEngine is a Noop metrics engine in case no metrics are configured. (may also be useful for tests)
type DummyMetricsEngine struct{}

// RecordRequest as a noop
func (me *DummyMetricsEngine) RecordRequest(labels pbsmetrics.Labels) {
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
func (me *DummyMetricsEngine) RecordImps(labels pbsmetrics.Labels, numImps int) {
	return
}

// RecordRequestTime as a noop
func (me *DummyMetricsEngine) RecordRequestTime(labels pbsmetrics.Labels, length time.Duration) {
	return
}

// RecordAdapterPanic as a noop
func (me *DummyMetricsEngine) RecordAdapterPanic(labels pbsmetrics.AdapterLabels) {
	return
}

// RecordAdapterRequest as a noop
func (me *DummyMetricsEngine) RecordAdapterRequest(labels pbsmetrics.AdapterLabels) {
	return
}

// RecordAdapterBidReceived as a noop
func (me *DummyMetricsEngine) RecordAdapterBidReceived(labels pbsmetrics.AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	return
}

// RecordAdapterPrice as a noop
func (me *DummyMetricsEngine) RecordAdapterPrice(labels pbsmetrics.AdapterLabels, cpm float64) {
	return
}

// RecordAdapterTime as a noop
func (me *DummyMetricsEngine) RecordAdapterTime(labels pbsmetrics.AdapterLabels, length time.Duration) {
	return
}

// RecordCookieSync as a noop
func (me *DummyMetricsEngine) RecordCookieSync(labels pbsmetrics.Labels) {
	return
}

// RecordAdapterCookieSync as a noop
func (me *DummyMetricsEngine) RecordAdapterCookieSync(adapter openrtb_ext.BidderName, gdprBlocked bool) {
	return
}

// RecordUserIDSet as a noop
func (me *DummyMetricsEngine) RecordUserIDSet(userLabels pbsmetrics.UserLabels) {
	return
}
