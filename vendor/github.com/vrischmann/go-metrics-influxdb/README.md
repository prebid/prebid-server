go-metrics-influxdb
===================

This is a reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library which will post the metrics to [InfluxDB](https://influxdb.com/).

This version adds a measurement for the metrics, moves the histogram bucket names into tags, similar to the behavior of hitograms in telegraf, and aligns all metrics in a batch on the same timestamp.

Additionally, metrics can be aligned to the beginning of a bucket as defined by the interval.

Setting align to true will cause the timestamp to be truncated down to the nearest even integral of the reporting interval.

For example, if the interval is 30 seconds, tiemstamps will be aligned on :00 and :30 for every reporting interval.

This also maps to a similar option in Telegraf.

Note
----

This is only compatible with InfluxDB 0.9+.

Usage
-----

```go
import "github.com/jregovic/go-metrics-influxdb"

go influxdb.InfluxDBWithTags(
    metrics.DefaultRegistry,    // metrics registry
    time.Second * 10,           // interval
    metricsHost,                // the InfluxDB url
    database,                   // your InfluxDB database
    measurement,                // your measurement
    metricsuser,                // your InfluxDB user
    metricspass,                // your InfluxDB password
    tags,                       // your tag set as map[string]string
    aligntimestamps             // align the timestamps
)
```

License
-------

go-metrics-influxdb is licensed under the MIT license. See the LICENSE file for details.
