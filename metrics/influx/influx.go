package metrics

import (
	"os"
	"time"

	"github.com/golang/glog"
	coreInflux "github.com/influxdata/influxdb/client/v2"
	coreMetrics "github.com/prebid/prebid-server/metrics"

	"context"
	"github.com/prebid/prebid-server/pbs"
	"github.com/rcrowley/go-metrics"
	"strconv"
)

const DATABASE = "test_data"

// These are the names of Measurements which we export to Influx.
const (
	AUCTION_REQUEST_COUNT    = "prebidserver.auction_request_count"
	AUCTION_RESPONSE_COUNT   = "prebidserver.auction_response_count"
	AUCTION_REQUEST_DURATION = "prebidserver.auction_request_duration_nanos"

	BIDDER_REQUEST_COUNT    = "prebidserver.bidder_request_count"
	BIDDER_RESPONSE_COUNT   = "prebidserver.bidder_response_count"
	BIDDER_REQUEST_DURATION = "prebidserver.bidder_request_duration_nanos"
	BID_PRICES              = "prebidserver.bid_response_cpm_cents"
	BID_COUNT               = "prebidserver.bid_count"

	COOKIESYNC_REQUEST_COUNT = "prebidserver.cookiesync_request_count"
)

// PBSInflux is the struct which implements PBSMetrics backed by InfluxDB.
//
// You probably don't want to use this directly. Use NewInfluxMetrics() instead.
type PBSInflux struct {
	Registry *TaggableRegistry
}

// NewInfluxMetrics returns a PBSMetrics which logs data to InfluxDB through the given Client.
func NewInfluxMetrics(client coreInflux.Client) coreMetrics.PBSMetrics {
	var registry = &TaggableRegistry{
		Delegate: metrics.NewRegistry(),
	}

	var hostname, err = os.Hostname()
	if err != nil {
		glog.Warningf("Failed to determine hostname. Deafulting to \"unknown\" for metrics: %s", err.Error())
		hostname = "unknown"
	}

	var reporter = Reporter{
		Client:   client,
		Database: DATABASE,
		Interval: 10 * time.Second,
		Registry: registry,
		Tags: map[string]string{
			"hostname": hostname,
		},
	}

	go reporter.run()

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
func (influx *PBSInflux) StartAuctionRequest(requestInfo *coreMetrics.AuctionRequestInfo) coreMetrics.ServerRequestFollowups {
	var followupTags = map[string]string{
		"account_id": requestInfo.AccountId,
	}

	var initialTags = combineMaps(map[string]string{
		"source":     requestInfo.RequestSource.String(),
		"has_cookie": strconv.FormatBool(requestInfo.HasCookie),
	}, followupTags)

	influx.Registry.GetOrRegisterMeter(AUCTION_REQUEST_COUNT, initialTags).Mark(1)

	return &InfluxAuctionRequestFollowups{
		Influx:    influx,
		Tags:      followupTags,
		StartTime: time.Now(),
	}
}

type InfluxAuctionRequestFollowups struct {
	Influx    *PBSInflux
	Tags      map[string]string
	StartTime time.Time
}

func (f *InfluxAuctionRequestFollowups) Completed(err error) {
	if err == nil {
		f.Influx.Registry.GetOrRegisterTimer(AUCTION_REQUEST_DURATION, f.Tags).UpdateSince(f.StartTime)
		f.Influx.Registry.GetOrRegisterMeter(AUCTION_RESPONSE_COUNT, f.WithResponseTypeTag("success")).Mark(1)
	} else {
		f.Influx.Registry.GetOrRegisterMeter(AUCTION_RESPONSE_COUNT, f.WithResponseTypeTag("error")).Mark(1)
	}
}

func (followups *InfluxAuctionRequestFollowups) WithResponseTypeTag(value string) map[string]string {
	return combineMaps(map[string]string{"type": value}, followups.Tags)
}

// InfluxBidderRequestFollowups is the Influx implementation of the BidderRequestFollowups interface.
type InfluxBidderRequestFollowups struct {
	Influx    *PBSInflux
	Tags      map[string]string
	StartTime time.Time
}

func (f *InfluxBidderRequestFollowups) BidderResponded(bids pbs.PBSBidSlice, err error) {
	f.Influx.Registry.GetOrRegisterTimer(BIDDER_REQUEST_DURATION, f.Tags).UpdateSince(f.StartTime)

	if err != nil {
		switch err {
		case context.DeadlineExceeded:
			f.Influx.Registry.GetOrRegisterMeter(BIDDER_RESPONSE_COUNT, f.WithResponseTypeTag("timeout")).Mark(1)
		default:
			f.Influx.Registry.GetOrRegisterMeter(BIDDER_RESPONSE_COUNT, f.WithResponseTypeTag("error")).Mark(1)
		}
	} else {
		f.Influx.Registry.GetOrRegisterMeter(BIDDER_RESPONSE_COUNT, f.WithResponseTypeTag("success")).Mark(1)
		f.Influx.Registry.GetOrRegisterMeter(BID_COUNT, f.Tags).Mark(int64(len(bids)))
		for _, bid := range bids {
			f.Influx.Registry.GetOrRegisterHistogram(BID_PRICES, f.Tags).Update(int64(bid.Price * 1000))
		}
	}
}

func (followups *InfluxBidderRequestFollowups) WithResponseTypeTag(value string) map[string]string {
	return combineMaps(map[string]string{"type": value}, followups.Tags)
}

// BidderStartedRequest implements part of the PBSMetrics interface.
//
// This function tracks:
//
//   1. The number of requests to this bidder.
//   2. The number of times this bidder responded with bids, had an error, or was too slow for the timeout.
//   3. A price distribution for all the bids
func (influx *PBSInflux) StartBidRequest(
	auctionRequestInfo *coreMetrics.AuctionRequestInfo,
	bidRequestInfo *coreMetrics.BidRequestInfo) coreMetrics.BidderRequestFollowups {

	var followupTags = map[string]string{
		"account_id":  auctionRequestInfo.AccountId,
		"bidder_code": bidRequestInfo.Bidder.BidderCode,
	}

	var requestTags = combineMaps(map[string]string{
		"has_cookie": strconv.FormatBool(auctionRequestInfo.HasCookie),
	}, followupTags)

	influx.Registry.GetOrRegisterMeter(BIDDER_REQUEST_COUNT, requestTags).Mark(1)

	return &InfluxBidderRequestFollowups{
		Tags:      followupTags,
		StartTime: time.Now(),
	}
}
