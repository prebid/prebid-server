package analytics

import (
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
	"net"
	"time"
)

const (
	FILE     = "file"
	GRAPHITE = "graphite"
	INFLUXDB = "influxdb"
)

type Analytics struct {
	Loggers []AnalyticsLogger
}

func (a *Analytics) Setup(settings []config.Metrics, adapters []openrtb_ext.BidderName) {
	loggers := make([]AnalyticsLogger, 0)
	for _, setting := range settings {
		switch setting.Type {
		case FILE:
		case GRAPHITE:
			//TODO: Validate configured settings
			loggers = append(loggers, new(GraphiteLogger).Setup(setting, adapters))

		case INFLUXDB:
			//TODO: Validate configured settings
			loggers = append(loggers, new(InfluxDBLogger).Setup(setting, adapters))
		}
	}
}

func (a *Analytics) LogTransaction(to *TransactionObject) {
	for _, logger := range a.Loggers {
		logger.LogTransaction(to)
	}
}

//Log to graphite
type GraphiteLogger struct {
	pbsMetrics PBSMetrics
}

//configure graphite
func (g *GraphiteLogger) Setup(setting config.Metrics, adapters []openrtb_ext.BidderName) *GraphiteLogger {
	metricsRegistry := metrics.NewPrefixedRegistry("")
	g.pbsMetrics.setup(metricsRegistry, adapters)

	addr, err := net.ResolveTCPAddr("tcp", setting.Host)
	if err == nil {
		go graphite.Graphite(
			metricsRegistry,                             // pbsMetrics registry
			time.Second*time.Duration(setting.Interval), // interval
			setting.Prefix,                              // prefix
			addr,                                        // graphite host
		)
	}
	return g
}

//implementation of AnalyticsLogger to send data to graphite
func (gl *GraphiteLogger) LogTransaction(to *TransactionObject) {
	extractAndUpdateMetrics(to, &gl.pbsMetrics)
}

//Log to InfluxDB
type InfluxDBLogger struct {
	pbsMetrics PBSMetrics
}

//configure InfluxDB
func (i *InfluxDBLogger) Setup(setting config.Metrics, adapters []openrtb_ext.BidderName) *InfluxDBLogger {
	metricsRegistry := metrics.NewPrefixedRegistry(setting.Prefix)
	i.pbsMetrics.setup(metricsRegistry, adapters)
	go influxdb.InfluxDB(
		metricsRegistry,                             // pbsMetrics registry
		time.Second*time.Duration(setting.Interval), // interval
		setting.Host,                                // the InfluxDB url
		setting.Database,                            // your InfluxDB database
		setting.Username,                            // your InfluxDB user
		setting.Password,                            // your InfluxDB password
	)
	return i
}

//implementation of AnalyticsLogger to send data to InfluxDB
func (il *InfluxDBLogger) LogTransaction(to *TransactionObject) {
	extractAndUpdateMetrics(to, &il.pbsMetrics)
}

func extractAndUpdateMetrics(to *TransactionObject, pbsm *PBSMetrics) {

}
