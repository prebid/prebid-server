package metrics

import (
	"time"

	"github.com/golang/glog"
	influxClient "github.com/influxdata/influxdb/client/v2"
	"github.com/rcrowley/go-metrics"
)

// This file is largely cannibalized from https://github.com/f0ster/go-metrics-influxdb...
// but has been updated to support Tags, and use the v2 InfluxDB client.

type reporter struct {
	registry TaggableRegistry
	interval time.Duration
	addr     string
	database string
	username string
	password string
	tags     map[string]string

	client influxClient.Client
}

// ReportMetrics causes metrics from the given TaggableRegistry to be posted to Influx at each interval d.
//
// Any tags sent in here will be sent on every measurement from this Registry. If any conflict with the
// metric-level tags, the metric-level tags will take precedence.
func ReportMetrics(
	registry TaggableRegistry,
	interval time.Duration,
	addr string,
	database string,
	username string,
	password string,
	tags map[string]string) {

	rep := &reporter{
		registry: registry,
		interval: interval,
		addr:     addr,
		database: database,
		username: username,
		password: password,
		tags:     tags,
	}

	go rep.start()
}

func (r *reporter) start() {
	var err = r.makeClient()
	if (err != nil) {
		glog.Errorf("Failed to create InfluxDB client. %v.", err)
	}
	r.run()
}

func (r *reporter) makeClient() error {
	var newClient, err = influxClient.NewHTTPClient(influxClient.HTTPConfig{
		Addr:     r.addr,
		Username: r.username,
		Password: r.password,
	})

	r.client = newClient
	return err
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(time.Second * 5)

	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				glog.Warningf("Failed to send metrics to InfluxDB. %v", err)
			}
		case <-pingTicker:
			_, _, err := r.client.Ping(1 * time.Second)
			if err != nil {
				glog.Warningf("Failed to ping InfluxDB. %v. Trying to recreate client.", err)

				if err = r.makeClient(); err != nil {
					glog.Warningf("Failed to recreate InfluxDB client. %v", err)
				}
			}
		}
	}
}

func (r *reporter) send() error {
	var pts, err = influxClient.NewBatchPoints(influxClient.BatchPointsConfig{
		Database: r.database,
	})
	if (err != nil) {
		glog.Warningf("Failed to create InfluxDB BatchPoints. %v. Some metrics may be missing", err)
	}

	var tryAddPoint = func(point *influxClient.Point, err error) {
		if (err != nil) {
			glog.Warningf("Failed to create InfluxDB Point. %v. Some metrics may be missing", err)
		} else {
			pts.AddPoint(point)
		}
	}

	r.registry.Each(func(name string, tags map[string]string, i interface{}) {
		now := time.Now()

		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			tryAddPoint(influxClient.NewPoint(
				name,
				combineMaps(r.tags, tags),
				map[string]interface{}{
					"value": ms.Count(),
				},
				now))
		case metrics.Gauge:
			ms := metric.Snapshot()
			tryAddPoint(influxClient.NewPoint(
				name,
				combineMaps(r.tags, tags),
				map[string]interface{}{
					"value": ms.Value(),
				},
				now))
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			tryAddPoint(influxClient.NewPoint(
				name,
				combineMaps(r.tags, tags),
				map[string]interface{}{
					"value": ms.Value(),
				},
				now))
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			tryAddPoint(influxClient.NewPoint(
				name,
				combineMaps(r.tags, tags),
				map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
				},
				now))
		case metrics.Meter:
			ms := metric.Snapshot()
			tryAddPoint(influxClient.NewPoint(
				name,
				combineMaps(r.tags, tags),
				map[string]interface{}{
					"count": ms.Count(),
					"m1":    ms.Rate1(),
					"m5":    ms.Rate5(),
					"m15":   ms.Rate15(),
					"mean":  ms.RateMean(),
				},
				now))
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			tryAddPoint(influxClient.NewPoint(
				name,
				combineMaps(r.tags, tags),
				map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
					"m1":       ms.Rate1(),
					"m5":       ms.Rate5(),
					"m15":      ms.Rate15(),
					"meanrate": ms.RateMean(),
				},
				now))
		}
	})

	return r.client.Write(pts)
}

func combineMaps(a, b map[string]string) map[string]string {
	for k, v := range b {
		a[k] = v
	}
	return a
}
