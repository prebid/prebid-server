package metrics

import (
	"os"
	"time"

	"github.com/golang/glog"
	// coreInflux "github.com/influxdata/influxdb/client/v2"
	pbsmetrics "github.com/prebid/prebid-server/metrics"

	"github.com/rcrowley/go-metrics"
)

const DATABASE = "test_data"

// PBSInflux is the struct which implements PBSMetrics backed by InfluxDB.
//
// You probably don't want to use this directly. Use NewInfluxMetrics() instead.
type PBSInflux struct {
	registry *TaggableRegistry
}

// NewInfluxMetrics returns a PBSMetrics which logs data to an InfluxDB instance through the given Client.
func NewInfluxMetrics() pbsmetrics.PBSMetrics {
	var registry = TaggableRegistry{
		delegate: metrics.NewRegistry(),
	}

	var hostname, err = os.Hostname()
	if err != nil {
		glog.Warningf("Failed to determine hostname. Deafulting to \"unknown\" for metrics: %s", err.Error())
		hostname = "unknown"
	}

	ReportMetrics(registry, 1 * time.Second, "http://52.170.44.44:8086", DATABASE, "test", "test", map[string]string{"hostname": hostname})

	var influxMetrics = &PBSInflux{
		registry: &registry,
	}

	return influxMetrics
}

type InfluxServerRequestFollowups struct {
	completed func()
	failed    func()
}

func (followups *InfluxServerRequestFollowups) Completed() {
	followups.completed()
}

func (followups *InfluxServerRequestFollowups) Failed() {
	followups.failed()
}

// ServerStartedRequest implements part of the PBSMetrics interface.
//
// This function tracks:
//
//   1. The number of requests which come to prebid-server.
//   2. The number of requests which end in some sort of error.
//   3. Performance metrics for how long it takes prebid-server to respond.
func (influx *PBSInflux) ServerStartedRequest(requestInfo *pbsmetrics.RequestInfo) pbsmetrics.ServerRequestFollowups {
	var tags = map[string]string{"publisher": requestInfo.Publisher}

	influx.registry.GetOrRegisterMeter("prebidserver/request_count", tags).Mark(1)
	var requestStartTime = time.Now()

	return &InfluxServerRequestFollowups{
		completed: func() {
			influx.registry.GetOrRegisterTimer("prebidserver/request_duration", tags).UpdateSince(requestStartTime)
		},

		failed: func() {
			influx.registry.GetOrRegisterMeter("prebidserver/request_errors", tags).Mark(1)
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
