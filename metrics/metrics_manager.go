package metrics

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/rcrowley/go-metrics"
	"github.com/rubicon-project/go-metrics-graphite"
	"github.com/vrischmann/go-metrics-influxdb"
	"net"
	"time"
)

const TYPE_COUNTER string = "counter"
const TYPE_INFLUXDB, TYPE_GRAPHITE string = "influxdb", "graphite"

type PBSMetrics interface {
	Setup(metrics.Registry, map[string]adapters.Adapter)
	GetMetrics() PBSMetrics
	IncRequest(int64)
	IncSafariRequest(int64)
	IncAppRequest(int64)
	IncNoCookie(int64)
	IncSafariNoCookie(int64)
	IncError(int64)
	IncCookieSync(int64)
	GetMyAccountMetrics(string) AccountMetrics
	GetMyAdapterMetrics(string) AdapterMetrics
	UpdateRequestTimerSince(time.Time)
}

type AccountMetrics interface {
	IncRequest(int64)
	IncBidsReceived(int64)
	GetMyAdapterMetrics(string) AdapterMetrics
	UpdatePriceHistogram(int64)
}

type AdapterMetrics interface {
	IncRequest(int64)
	IncNoCookie(int64)
	UpdateRequestTimerSince(time.Time)
	IncTimeOut(int64)
	IncError(int64)
	IncBidsReceived(int64)
	UpdatePriceHistogram(int64)
	IncNoBid(int64)
}

func SetupMetrics(settings config.Metrics, exchanges map[string]adapters.Adapter) (PBSMetrics, metrics.Registry) {
	metricsRegistry := initReporting(settings)
	if settings.MetricType == TYPE_COUNTER {
		counterMets := AllCounterMetrics{}
		(&counterMets).Setup(metricsRegistry, exchanges)
		return (&counterMets).GetMetrics(), metricsRegistry
	} else {
		meterMets := AllMeterMetrics{}
		(&meterMets).Setup(metricsRegistry, exchanges)
		return (&meterMets).GetMetrics(), metricsRegistry
	}
}

func initReporting(settings config.Metrics) metrics.Registry {
	var metricsRegistry metrics.Registry
	if settings.Type == TYPE_INFLUXDB {
		metricsRegistry = metrics.NewPrefixedRegistry(settings.Prefix)
		go influxdb.InfluxDB(
			metricsRegistry,               // metrics registry
			time.Second*settings.Interval, // interval
			settings.Host,                 // the InfluxDB url
			settings.Database,             // your InfluxDB database
			settings.Username,             // your InfluxDB user
			settings.Password,             // your InfluxDB password
		)

	} else if settings.Type == TYPE_GRAPHITE {
		metricsRegistry = metrics.NewPrefixedRegistry("")
		addr, err := net.ResolveTCPAddr("tcp", settings.Host)
		if err == nil {
			go graphite.Graphite(
				metricsRegistry,               // metrics registry
				time.Second*settings.Interval, // interval
				settings.Prefix,               // prefix
				addr,                          // graphite host
				settings.ClearCounter, //clear counters after flush
			)
		} else {
			glog.Info(err)
		}
	}
	return metricsRegistry
}
