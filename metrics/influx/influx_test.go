package metrics

import (
	"testing"
	"github.com/rcrowley/go-metrics"
	coreMetrics "github.com/prebid/prebid-server/metrics"
	"strconv"
	"errors"
)

func getAucStartTags(reqInfo *coreMetrics.AuctionRequestInfo) map[string]string {
	return map[string]string{
		"account_id": reqInfo.AccountId,
		"source": reqInfo.RequestSource.String(),
		"has_cookie": strconv.FormatBool(reqInfo.HasCookie),
	}
}

func expectedResponseType(error bool) string {
	if error {
		return "error"
	} else {
		return "success"
	}
}

func getAucEndTags(reqInfo *coreMetrics.AuctionRequestInfo, errored bool) map[string]string {
	return map[string]string{
		"account_id": reqInfo.AccountId,
		"type": expectedResponseType(errored),
	}
}

func doAuctionStart(t *testing.T, influx *pbsInflux, reqInfo *coreMetrics.AuctionRequestInfo) coreMetrics.AuctionRequestFollowups {
	followups := influx.StartAuctionRequest(reqInfo)
	expectedTags := getAucStartTags(reqInfo)
	numRequests := influx.Registry.getOrRegisterMeter(AUCTION_REQUEST_COUNT, expectedTags).Snapshot().Count()
	if numRequests != 1 {
		t.Errorf("Expected 1 AuctionRequest event. Got %d", numRequests)
	}
	return followups
}

func TestAuctionCounters(t *testing.T) {
	registry := taggableRegistry{metrics.NewRegistry()}
	influx := &pbsInflux{&registry}

	auc1Info := &coreMetrics.AuctionRequestInfo{
		AccountId: "pub",
		RequestSource: coreMetrics.SAFARI,
		HasCookie: true,
	}
	doAuctionStart(t, influx, auc1Info).Completed(nil)
	if influx.Registry.getOrRegisterMeter(AUCTION_RESPONSE_COUNT, getAucEndTags(auc1Info, false)).Snapshot().Count() != 1 {
		t.Error("Failed to log the successful auction completed response.")
	}

	auc2Info := &coreMetrics.AuctionRequestInfo{
		AccountId: "pub",
		RequestSource: coreMetrics.APP,
		HasCookie: true,
	}
	doAuctionStart(t, influx, auc2Info).Completed(errors.New("Some failure"))
	if influx.Registry.getOrRegisterMeter(AUCTION_RESPONSE_COUNT, getAucEndTags(auc1Info, true)).Snapshot().Count() != 1 {
		t.Error("Failed to log the failed auction completed response.")
	}

	doAuctionStart(t, influx, &coreMetrics.AuctionRequestInfo{
		AccountId: "pub",
		RequestSource: coreMetrics.OTHER,
		HasCookie: false,
	})
}
