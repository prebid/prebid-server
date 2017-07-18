package main

import (
	"errors"
	coreInflux "github.com/influxdata/influxdb/client/v2"
	pbsMetrics "github.com/prebid/prebid-server/metrics"
	pbsInflux "github.com/prebid/prebid-server/metrics/influx"
)

func main() {
	var influxClient, err = coreInflux.NewHTTPClient(coreInflux.HTTPConfig{
		Addr:     "http://52.170.44.44:8086",
		Username: "root",
		Password: "root",
	})

	if err != nil {
		panic("Couldn't create Influx client.")
	}

	var metrics = pbsInflux.NewInfluxMetrics(influxClient)

	for i := 0; i < 1000000000; i++ {
		var auctionFollowups1 = metrics.StartAuctionRequest(&pbsMetrics.AuctionRequestInfo{
			AccountId:     "phony-pub",
			RequestSource: pbsMetrics.SAFARI,
			HasCookie:     true,
		})

		var auctionFollowups2 = metrics.StartAuctionRequest(&pbsMetrics.AuctionRequestInfo{
			AccountId:     "phony-pub2",
			RequestSource: pbsMetrics.APP,
			HasCookie:     true,
		})

		auctionFollowups1.Completed(nil)
		auctionFollowups2.Completed(errors.New("Some failure occurred."))
	}
}
