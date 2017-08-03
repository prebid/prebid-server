package metrics

import (
	"context"
	"errors"
	coreMetrics "github.com/prebid/prebid-server/metrics"
	"github.com/rcrowley/go-metrics"
	"strconv"
	"testing"
)

func TestAuctionEvents(t *testing.T) {
	registry := taggableRegistry{metrics.NewRegistry()}
	influx := &pbsInflux{&registry}

	auc1Info := &coreMetrics.AuctionRequestInfo{
		AccountId:     "pub",
		Browser:       coreMetrics.UNKNOWN,
		Source:        coreMetrics.APP,
		HasCookie:     true,
	}
	doAuctionStart(t, influx, auc1Info).Completed(nil)
	if influx.registry.getOrRegisterMeter(AUCTION_RESPONSE_COUNT, getAucEndTags(auc1Info, nil)).Snapshot().Count() != 1 {
		t.Error("Failed to log the successful auction completed response.")
	}
	if influx.registry.getOrRegisterTimer(AUCTION_REQUEST_DURATION, getAucDurationTags(auc1Info)).Snapshot().Count() != 1 {
		t.Error("Failed to log a duration for the auction request.")
	}

	auc2Info := &coreMetrics.AuctionRequestInfo{
		AccountId:     "pub",
		Browser:       coreMetrics.UNKNOWN,
		Source: coreMetrics.APP,
		HasCookie:     true,
	}
	doAuctionStart(t, influx, auc2Info).Completed(errors.New("Some failure"))
	if influx.registry.getOrRegisterMeter(AUCTION_RESPONSE_COUNT, getAucEndTags(auc2Info, errors.New("some error"))).Snapshot().Count() != 1 {
		t.Error("Failed to log the failed auction completed response.")
	}

	doAuctionStart(t, influx, &coreMetrics.AuctionRequestInfo{
		AccountId:     "pub",
		Browser:       coreMetrics.SAFARI,
		Source:        coreMetrics.DESKTOP,
		HasCookie:     false,
	})
}

func TestBidderWithCookie(t *testing.T) {
	registry := taggableRegistry{delegate: metrics.NewRegistry()}
	influx := &pbsInflux{registry: &registry}

	aucInfo := &coreMetrics.AuctionRequestInfo{
		AccountId:     "pub",
		Browser:       coreMetrics.UNKNOWN,
		Source:        coreMetrics.DESKTOP,
		HasCookie:     true,
	}

	bidInfo1 := &coreMetrics.BidRequestInfo{
		BidderCode: "bidder1",
		HasCookie: true,
	}
	var count int64
	doBidderStart(t, influx, aucInfo, bidInfo1).BidderResponded(nil, errors.New("Internal bidder error"))
	count = influx.registry.getOrRegisterMeter(BIDDER_RESPONSE_COUNT, getBidEndTagsWithType(aucInfo, bidInfo1, errors.New("some error"))).Snapshot().Count()
	if count != 1 {
		t.Errorf("Failed Bidder Completed Failure metric. Expected %d, got %d.", 1, count)
	}
	count = influx.registry.getOrRegisterTimer(BIDDER_REQUEST_DURATION, getBidEndTags(aucInfo, bidInfo1)).Snapshot().Count()
	if count != 0 {
		t.Errorf("Failed Bidder Duration Failure metric: Expected %d, got %d.", 0, count)
	}
	count = influx.registry.getOrRegisterMeter(BID_COUNT, getBidEndTagsWithType(aucInfo, bidInfo1, errors.New("some error"))).Snapshot().Count()
	if count != 0 {
		t.Errorf("Failed Bid count Failure metric: Expected %d, got %d.", 0, count)
	}
}

func TestBidderNoCookie(t *testing.T) {
	registry := taggableRegistry{delegate: metrics.NewRegistry()}
	influx := &pbsInflux{registry: &registry}

	aucInfo := &coreMetrics.AuctionRequestInfo{
		AccountId:     "pub",
		Browser:       coreMetrics.SAFARI,
		Source:        coreMetrics.DESKTOP,
		HasCookie:     true,
	}
	bidInfo2 := &coreMetrics.BidRequestInfo{
		BidderCode: "bidder1",
		HasCookie: false,
	}
	mockResponse := []float64{0.07, 0.08}
	doBidderStart(t, influx, aucInfo, bidInfo2).BidderResponded(mockResponse, nil)
	var count int64
	count = influx.registry.getOrRegisterMeter(BIDDER_RESPONSE_COUNT, getBidEndTagsWithType(aucInfo, bidInfo2, nil)).Snapshot().Count()
	if count != 1 {
		t.Errorf("Failed Auction Completed Successfully metric: Expected %d, got %d.", 1, count)
	}
	count = influx.registry.getOrRegisterTimer(BIDDER_REQUEST_DURATION, getBidEndTags(aucInfo, bidInfo2)).Snapshot().Count()
	if count != 1 {
		t.Errorf("Failed Bidder Duration Success metric: Expected %d, got %d.", 1, count)
	}
	count = influx.registry.getOrRegisterMeter(BID_COUNT, getBidEndTags(aucInfo, bidInfo2)).Snapshot().Count()
	if count != 2 {
		t.Errorf("Failed Bid count Success metric: Expected %d, got %d", 2, count)
	}
}

func TestBidderSkipNoCookie(t *testing.T) {
	registry := taggableRegistry{delegate: metrics.NewRegistry()}
	influx := &pbsInflux{registry: &registry}

	aucInfo := &coreMetrics.AuctionRequestInfo{
		AccountId:     "pub",
		Browser:       coreMetrics.UNKNOWN,
		Source:        coreMetrics.DESKTOP,
		HasCookie:     true,
	}
	bidInfo := &coreMetrics.BidRequestInfo{
		BidderCode: "bidder1",
		HasCookie: false,
	}
	doBidderStart(t, influx, aucInfo, bidInfo).BidderSkipped()

	var count int64
	tags := getBidEndTags(aucInfo, bidInfo)
	tags["type"] = "skipped_no_cookie"
	count = influx.registry.getOrRegisterMeter(BIDDER_RESPONSE_COUNT, tags).Snapshot().Count()
	if count != 1 {
		t.Errorf("Failed Log Bidder Skipped User Without Cookie: Expected %d, got %d.", 1, count)
	}
}

func TestCookieSyncEvents(t *testing.T) {
	registry := taggableRegistry{delegate: metrics.NewRegistry()}
	influx := &pbsInflux{registry: &registry}

	influx.StartCookieSyncRequest()

	count := influx.registry.getOrRegisterMeter(COOKIESYNC_REQUEST_COUNT, nil).Snapshot().Count()
	if count != 1 {
		t.Errorf("Failed to log cookiesync event. Expected %d, got %d", 1, count)
	}

}

func TestRespTypeForAuctionParsing(t *testing.T) {
	if makeRespTypeForAuction(nil) != "success" {
		t.Errorf("Successful auctions should count events as tag type=\"success\". Got %s", makeRespTypeForAuction(nil))
	}

	if makeRespTypeForAuction(errors.New("Any error")) != "error" {
		t.Errorf("Failed auctions should count events as tag type=\"error\". Got %s", makeRespTypeForAuction(errors.New("Any error")))
	}
}

func TestRespTypeForBidderParsing(t *testing.T) {
	if makeRespTypeForBidder(nil) != "success" {
		t.Errorf("Successful bidders should count events as tag type=\"success\". Got %s", makeRespTypeForBidder(nil))
	}

	if makeRespTypeForBidder(errors.New("Any error")) != "error" {
		t.Errorf("Bidders with generic errors should count events as tag type=\"error\". Got %s", makeRespTypeForBidder(errors.New("Any error")))
	}

	timeout := context.DeadlineExceeded

	if makeRespTypeForBidder(timeout) != "timeout" {
		t.Errorf("Bidders with generic errors should count events as tag type=\"error\". Got %s", makeRespTypeForBidder(timeout))
	}
}

// Get a map of the tags which we expect to exist on AUCTION_REQUEST_COUNT events
func getAucStartTags(reqInfo *coreMetrics.AuctionRequestInfo) map[string]string {
	return map[string]string{
		"account_id": reqInfo.AccountId,
		"source":     reqInfo.Source.String(),
		"browser":    reqInfo.Browser.String(),
		"has_cookie": strconv.FormatBool(reqInfo.HasCookie),
	}
}

// Get a map of the tags which we expect to exist on AUCTION_RESPONSE_COUNT events
func getAucEndTags(reqInfo *coreMetrics.AuctionRequestInfo, err error) map[string]string {
	return map[string]string{
		"account_id": reqInfo.AccountId,
		"type":       makeRespTypeForAuction(err),
	}
}

// Get a map of the tags which we expect to exist on AUCTION_REQUEST_DURATION events
func getAucDurationTags(reqInfo *coreMetrics.AuctionRequestInfo) map[string]string {
	return map[string]string{
		"account_id": reqInfo.AccountId,
	}
}

func doAuctionStart(t *testing.T, influx *pbsInflux, reqInfo *coreMetrics.AuctionRequestInfo) coreMetrics.AuctionRequestFollowups {
	followups := influx.StartAuctionRequest(reqInfo)
	expectedTags := getAucStartTags(reqInfo)
	numRequests := influx.registry.getOrRegisterMeter(AUCTION_REQUEST_COUNT, expectedTags).Snapshot().Count()
	if numRequests != 1 {
		t.Errorf("Expected 1 AuctionRequest event. Got %d", numRequests)
	}
	return followups
}

func doBidderStart(
	t *testing.T,
	influx *pbsInflux,
	reqInfo *coreMetrics.AuctionRequestInfo,
	bidInfo *coreMetrics.BidRequestInfo) coreMetrics.BidderRequestFollowups {
	followups := influx.StartBidderRequest(reqInfo, bidInfo)
	expectedTags := getBidStartTags(reqInfo, bidInfo)
	numRequests := influx.registry.getOrRegisterMeter(BIDDER_REQUEST_COUNT, expectedTags).Snapshot().Count()
	if numRequests != 1 {
		t.Errorf("Expected 1 BidderRequestStart counter event. Got %d", numRequests)
	}
	return followups
}

func getBidStartTags(reqInfo *coreMetrics.AuctionRequestInfo, bidInfo *coreMetrics.BidRequestInfo) map[string]string {
	return map[string]string{
		"account_id":  reqInfo.AccountId,
		"bidder_code": bidInfo.BidderCode,
		"has_cookie":  strconv.FormatBool(bidInfo.HasCookie),
	}
}

func getBidEndTagsWithType(reqInfo *coreMetrics.AuctionRequestInfo, bidInfo *coreMetrics.BidRequestInfo, err error) map[string]string {
	tags := getBidEndTags(reqInfo, bidInfo)
	tags["type"] = makeRespTypeForBidder(err)
	return tags
}

func getBidEndTags(reqInfo *coreMetrics.AuctionRequestInfo, bidInfo *coreMetrics.BidRequestInfo) map[string]string {
	return map[string]string{
		"account_id":  reqInfo.AccountId,
		"bidder_code": bidInfo.BidderCode,
	}
}
