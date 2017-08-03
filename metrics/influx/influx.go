package metrics

import (
	"os"
	"time"

	"github.com/golang/glog"
	coreInflux "github.com/influxdata/influxdb/client/v2"
	coreMetrics "github.com/prebid/prebid-server/metrics"

	"context"
	"github.com/rcrowley/go-metrics"
	"strconv"
)

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
// Users outside the package should use NewInfluxMetrics() instead.
type pbsInflux struct {
	registry *taggableRegistry
}

// NewInfluxMetrics returns a PBSMetrics which logs data to InfluxDB through the given Client.
func NewInfluxMetrics(client coreInflux.Client, database string) coreMetrics.PBSMetrics {
	var registry = &taggableRegistry{
		delegate: metrics.NewRegistry(),
	}

	var hostname, err = os.Hostname()
	if err != nil {
		glog.Warningf("Failed to determine hostname. Deafulting to \"unknown\" for metrics: %s", err.Error())
		hostname = "unknown"
	}

	var reporter = reporter{
		client:   client,
		database: database,
		registry: registry,
		tags: map[string]string{
			"hostname": hostname,
		},
	}

	go reporter.Run(time.Tick(1*time.Second), time.Tick(time.Second*5), nil)

	var influxMetrics = &pbsInflux{
		registry: registry,
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
func (influx *pbsInflux) StartAuctionRequest(requestInfo *coreMetrics.AuctionRequestInfo) coreMetrics.AuctionRequestFollowups {
	var followupTags = map[string]string{
		"account_id": requestInfo.AccountId,
	}

	var initialTags = combineMaps(map[string]string{
		"source":     requestInfo.RequestSource.String(),
		"has_cookie": strconv.FormatBool(requestInfo.HasCookie),
	}, followupTags)

	influx.registry.getOrRegisterMeter(AUCTION_REQUEST_COUNT, initialTags).Mark(1)

	return &influxAuctionRequestFollowups{
		Influx:    influx,
		Tags:      followupTags,
		StartTime: time.Now(),
	}
}

type influxAuctionRequestFollowups struct {
	Influx    *pbsInflux
	Tags      map[string]string
	StartTime time.Time
}

// makeRespTypeForAuction determines which "type" Tag value we use for AUCTION_RESPONSE_COUNT events.
func makeRespTypeForAuction(err error) string {
	if err == nil {
		return "success"
	} else {
		return "error"
	}
}

func (f *influxAuctionRequestFollowups) Completed(err error) {
	f.Influx.registry.getOrRegisterMeter(AUCTION_RESPONSE_COUNT, f.WithResponseTypeTag(makeRespTypeForAuction(err))).Mark(1)

	if err == nil {
		f.Influx.registry.getOrRegisterTimer(AUCTION_REQUEST_DURATION, f.Tags).UpdateSince(f.StartTime)
	}
}

func (f *influxAuctionRequestFollowups) WithResponseTypeTag(value string) map[string]string {
	return combineMaps(map[string]string{"type": value}, f.Tags)
}

// influxBidderRequestFollowups is the Influx implementation of the BidderRequestFollowups interface.
type influxBidderRequestFollowups struct {
	Influx    *pbsInflux
	Tags      map[string]string
	StartTime time.Time
}

func (f *influxBidderRequestFollowups) BidderSkipped() {
	f.Influx.registry.getOrRegisterMeter(BIDDER_RESPONSE_COUNT, f.WithResponseTypeTag("skipped_no_cookie")).Mark(1)
}

func (f *influxBidderRequestFollowups) BidderResponded(bidPrices []float64, err error) {
	f.Influx.registry.getOrRegisterMeter(BIDDER_RESPONSE_COUNT, f.WithResponseTypeTag(makeRespTypeForBidder(err))).Mark(1)

	if err == nil {
		f.Influx.registry.getOrRegisterTimer(BIDDER_REQUEST_DURATION, f.Tags).UpdateSince(f.StartTime)

		f.Influx.registry.getOrRegisterMeter(BID_COUNT, f.Tags).Mark(int64(len(bidPrices)))
		for _, bidPrice := range bidPrices {
			var histogram = f.Influx.registry.getOrRegisterHistogram(BID_PRICES, f.Tags, metrics.NewExpDecaySample(1028, 0.015))
			histogram.Update(int64(bidPrice * 1000))
		}
	}
}

// makeRespTypeForBidder determines which "type" Tag value we attach to BIDDER_RESPONSE_COUNT counts
func makeRespTypeForBidder(err error) string {
	if err == nil {
		return "success"
	} else {
		switch err {
		case context.DeadlineExceeded:
			return "timeout"
		default:
			return "error"
		}
	}
}

func (f *influxBidderRequestFollowups) WithResponseTypeTag(value string) map[string]string {
	return combineMaps(map[string]string{"type": value}, f.Tags)
}

// BidderStartedRequest implements part of the PBSMetrics interface.
//
// This function tracks:
//
//   1. The number of requests to this bidder.
//   2. The number of times this bidder responded with bids, had an error, or was too slow for the timeout.
//   3. A price distribution for all the bids
func (influx *pbsInflux) StartBidderRequest(
	auctionRequestInfo *coreMetrics.AuctionRequestInfo,
	bidRequestInfo *coreMetrics.BidRequestInfo) coreMetrics.BidderRequestFollowups {

	var followupTags = map[string]string{
		"account_id":  auctionRequestInfo.AccountId,
		"bidder_code": bidRequestInfo.BidderCode,
	}

	var requestTags = combineMaps(map[string]string{
		"has_cookie": strconv.FormatBool(bidRequestInfo.HasCookie),
	}, followupTags)

	influx.registry.getOrRegisterMeter(BIDDER_REQUEST_COUNT, requestTags).Mark(1)

	return &influxBidderRequestFollowups{
		Influx:    influx,
		Tags:      followupTags,
		StartTime: time.Now(),
	}
}

// StartCookieSyncRequest implements part of the PBSMetrics interface.
func (influx *pbsInflux) StartCookieSyncRequest() {
	influx.registry.getOrRegisterMeter(COOKIESYNC_REQUEST_COUNT, nil).Mark(1)
}
