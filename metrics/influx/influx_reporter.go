package metrics

import (
	"time"

	"github.com/golang/glog"
	coreInflux "github.com/influxdata/influxdb/client/v2"
	"github.com/rcrowley/go-metrics"
)

// This file is largely cannibalized from https://github.com/f0ster/go-metrics-influxdb...
// but has been updated to support Tags, and use the v2 InfluxDB client.

// Reporter periodically sends the metrics from a TaggableRegistry to Influx.
type reporter struct {
	// Client is the Influx client which we use to write data points.
	client coreInflux.Client
	// Database is the name of the Influx datatbase where metrics should go.
	database string
	// Registry stores the Metrics which should be written to Influx.
	registry *taggableRegistry
	// Tags specifies tags which should appear in *every* Measurement written to influx.
	tags map[string]string
}

// Run() starts the reporter.
// It will ping Influx whenever the pingNotifier receives a message.
// It will send any collected metrics whenever the sendNotifier receives a message.
// If done is not nil, a single "true" message will be passed just before this function exits
//
// It blocks until the causeSend channel is closed, so it should be run in its own goroutine.
//
// When the function exits, it will also close the Influx Client stored on the reporter.
func (r *reporter) Run(causePing <- chan time.Time, causeSend <- chan time.Time, done chan <- bool) {
	for sendChannelOpen := true; sendChannelOpen; {
		select {
		case _, sendChannelOpen = <- causeSend:
			if err := r.Send(); err != nil {
				glog.Warningf("Failed to send metrics to InfluxDB. %v", err)
			}
		case <-causePing:
			_, _, err := r.client.Ping(1 * time.Second)
			if err != nil {
				glog.Warningf("Failed to ping InfluxDB. %v.", err)
			}
		}
	}
	r.client.Close();
	if done != nil {
		done <- true
	}
}

// Send() sends the captured metrics to influx.
func (r *reporter) Send() error {
	var pts, err = coreInflux.NewBatchPoints(coreInflux.BatchPointsConfig{
		Database: r.database,
	})
	if err != nil {
		glog.Warningf("Failed to create InfluxDB BatchPoints. %v. Some metrics may be missing", err)
	}

	var tryAddPoint = func(point *coreInflux.Point, err error) {
		if err != nil {
			glog.Warningf("Failed to create InfluxDB Point. %v. Some metrics may be missing", err)
		} else {
			pts.AddPoint(point)
		}
	}

	r.registry.each(func(name string, tags map[string]string, i interface{}) {
		now := time.Now()

		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			tryAddPoint(coreInflux.NewPoint(
				name,
				combineMaps(tags, r.tags),
				map[string]interface{}{
					"value": ms.Count(),
				},
				now))
		case metrics.Gauge:
			ms := metric.Snapshot()
			tryAddPoint(coreInflux.NewPoint(
				name,
				combineMaps(tags, r.tags),
				map[string]interface{}{
					"value": ms.Value(),
				},
				now))
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			tryAddPoint(coreInflux.NewPoint(
				name,
				combineMaps(tags, r.tags),
				map[string]interface{}{
					"value": ms.Value(),
				},
				now))
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			tryAddPoint(coreInflux.NewPoint(
				name,
				combineMaps(tags, r.tags),
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
			tryAddPoint(coreInflux.NewPoint(
				name,
				combineMaps(tags, r.tags),
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
			tryAddPoint(coreInflux.NewPoint(
				name,
				combineMaps(tags, r.tags),
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

	if len(pts.Points()) > 0 {
		return r.client.Write(pts)
	} else {
		return nil
	}
}

func combineMaps(a, b map[string]string) map[string]string {
	for k, v := range b {
		a[k] = v
	}
	return a
}
