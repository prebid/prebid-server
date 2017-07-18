package metrics

import (
	"os"
	"time"

	"github.com/golang/glog"
	coreInflux "github.com/influxdata/influxdb/client/v2"
	pbsmetrics "github.com/prebid/prebid-server/metrics"

	"github.com/rcrowley/go-metrics"
)

const DATABASE = "test_data"

// PBSInflux is the struct which implements PBSMetrics backed by InfluxDB.
//
// You probably don't want to use this directly. Use NewInfluxMetrics() instead.
type PBSInflux struct {
	Registry *TaggableRegistry
}

// NewInfluxMetrics returns a PBSMetrics which logs data to an InfluxDB instance through the given Client.
func NewInfluxMetrics(client coreInflux.Client) pbsmetrics.PBSMetrics {
	var registry = &TaggableRegistry{
		Delegate: metrics.NewRegistry(),
	}

	var hostname, err = os.Hostname()
	if err != nil {
		glog.Warningf("Failed to determine hostname. Deafulting to \"unknown\" for metrics: %s", err.Error())
		hostname = "unknown"
	}

	var reporter = Reporter{
		Client: client,
		Database: DATABASE,
		Interval: 1 * time.Second,
		Registry: registry,
		Tags: map[string]string{
			"hostname": hostname,
		},
	};

	go reporter.run();

	var influxMetrics = &PBSInflux{
		Registry: registry,
	}

	return influxMetrics
}

// ServerStartedRequest implements part of the PBSMetrics interface.
//
// This function tracks:
//
//   1. The number of requests which come to prebid-server.
//   2. The number of requests which end in some sort of error.
//   3. Performance metrics for how long it takes prebid-server to respond.
func (influx *PBSInflux) ServerStartedRequest(requestInfo *pbsmetrics.RequestInfo) pbsmetrics.ServerRequestFollowups {
	var tags = map[string]string{"account_id": requestInfo.Publisher}

	influx.Registry.GetOrRegisterMeter("prebidserver/request_count", tags).Mark(1)
	var requestStartTime = time.Now()

	return &pbsmetrics.ServerRequestFollowupsPrototype{
		CompletedImpl: func() {
			influx.Registry.GetOrRegisterTimer("prebidserver/request_duration", tags).UpdateSince(requestStartTime)
		},

		FailedImpl: func() {
			influx.Registry.GetOrRegisterMeter("prebidserver/request_errors", tags).Mark(1)
		},
	}
}

// BidderStartedRequest implements part of the PBSMetrics interface.
//
// This function tracks:
//
//   1. The number of requests to this bidder.
//   2. The number of times this bidder responded with bids, had an error, or was too slow for the timeout.
//   3. A price distribution for all the bids
func (influx *PBSInflux) BidderStartedRequest(requestInfo pbsmetrics.BidderRequestInfo) pbsmetrics.BidderRequestFollowups {

	return nil
}
