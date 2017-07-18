package main

import (
	coreInflux "github.com/influxdata/influxdb/client/v2"
	pbsMetrics "github.com/prebid/prebid-server/metrics"
	pbsInflux "github.com/prebid/prebid-server/metrics/influx"
)

func main() {
	var influxClient, err = coreInflux.NewHTTPClient(coreInflux.HTTPConfig{
		Addr: "http://52.170.44.44:8086",
		Username: "root",
		Password: "root",
	})

	if (err != nil) {
		panic("Couldn't create Influx client.");
	}


	var metrics = pbsInflux.NewInfluxMetrics(influxClient)
	metrics.ServerStartedRequest(&pbsMetrics.RequestInfo{
		Publisher: "phony-pub",
		IsApp:     false,
		IsSafari:  true,
	})

	for i := 0; i < 1000000000; i++ {
		metrics.ServerStartedRequest(&pbsMetrics.RequestInfo{
			Publisher: "phony-pub",
			IsApp:     false,
			IsSafari:  true,
		})

		metrics.ServerStartedRequest(&pbsMetrics.RequestInfo{
			Publisher: "phony-pub2",
			IsApp:     false,
			IsSafari:  true,
		})
	}
}
