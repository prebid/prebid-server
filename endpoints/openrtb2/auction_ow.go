package openrtb2

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/pubmatic"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// recordRejectedBids records the rejected bids and respective rejection reason code
func recordRejectedBids(pubID string, seatNonBids []openrtb_ext.SeatNonBid, metricEngine metrics.MetricsEngine, label metrics.Labels) {

	var found bool
	var codeLabel string
	reasonCodeMap := make(map[openrtb3.NoBidReason]string)
	if len(seatNonBids) == 0 && label.RequestStatus == "badinput" {
		metricEngine.RecordBadRequest(string(label.RType), label.PubID, openrtb3.NoBidReason(18).Ptr())
	}

	for _, seatNonbid := range seatNonBids {
		for _, nonBid := range seatNonbid.NonBid {
			if codeLabel, found = reasonCodeMap[openrtb3.NoBidReason(nonBid.StatusCode)]; !found {
				codeLabel = strconv.FormatInt(int64(nonBid.StatusCode), 10)
				reasonCodeMap[openrtb3.NoBidReason(nonBid.StatusCode)] = codeLabel
			}
			metricEngine.RecordRejectedBids(pubID, seatNonbid.Seat, codeLabel)
		}
	}
}

func UpdateResponseExtOW(w http.ResponseWriter, bidResponse *openrtb2.BidResponse, ao analytics.AuctionObject) {
	defer func() {
		if r := recover(); r != nil {
			response, err := json.Marshal(bidResponse)
			if err != nil {
				glog.Error("response:" + string(response) + ". err: " + err.Error() + ". stacktrace:" + string(debug.Stack()))
				return
			}
			glog.Error("response:" + string(response) + ". stacktrace:" + string(debug.Stack()))
		}
	}()

	if bidResponse == nil {
		return
	}

	rCtx := pubmatic.GetRequestCtx(ao.HookExecutionOutcome)
	if rCtx == nil {
		return
	}

	//Send owlogger in response only in case of debug mode
	if rCtx.Debug && !rCtx.LoggerDisabled {
		var originalBidResponse *openrtb2.BidResponse
		if sdkutils.IsSdkBiddingEndpoint(rCtx.Endpoint) {
			originalBidResponse = new(openrtb2.BidResponse)
			*originalBidResponse = *bidResponse
			pubmatic.RestoreBidResponse(rCtx, ao)
		}

		owlogger, _ := pubmatic.GetLogAuctionObjectAsURL(ao, rCtx, false, true)
		if sdkutils.IsSdkBiddingEndpoint(rCtx.Endpoint) {
			*bidResponse = *originalBidResponse
		}
		if len(bidResponse.Ext) == 0 {
			bidResponse.Ext = []byte("{}")
		}
		if updatedExt, err := jsonparser.Set([]byte(bidResponse.Ext), []byte(strconv.Quote(owlogger)), "owlogger"); err == nil {
			bidResponse.Ext = updatedExt
		}
	} else if rCtx.Endpoint == models.EndpointAppLovinMax || rCtx.Endpoint == models.EndpointUnityLevelPlay || rCtx.Endpoint == models.EndpointAPS {
		bidResponse.Ext = nil
		if rCtx.AppLovinMax.Reject || rCtx.UnityLevelPlay.Reject {
			w.WriteHeader(http.StatusNoContent)
		}
	}

	if rCtx.WakandaDebug != nil && rCtx.WakandaDebug.IsEnable() {
		rCtx.WakandaDebug.SetHTTPResponseWriter(w)
	}
}

func removeDefaultBidsFromSeatNonBid(seatNonBid *openrtb_ext.SeatNonBidBuilder, ao *analytics.AuctionObject) {
	if seatNonBid == nil {
		return
	}

	for seat, nonBids := range *seatNonBid {
		// First pass: count number of bids per ImpID
		impCount := make(map[string]int)
		for _, bid := range nonBids {
			impCount[bid.ImpId]++
		}

		// Second pass: remove default bids with StatusCode == 0 only if that ImpID has >1 bid
		cleanedNonSeatBids := make([]openrtb_ext.NonBid, 0, len(nonBids))
		for _, bid := range nonBids {
			if impCount[bid.ImpId] > 1 && bid.StatusCode == 0 {
				glog.V(3).Infof("Removing Default bid from seatNonBid of seat %s and impid %s", seat, bid.ImpId)
				continue
			}
			cleanedNonSeatBids = append(cleanedNonSeatBids, bid)
		}
		(*seatNonBid)[seat] = cleanedNonSeatBids
	}
	ao.SeatNonBid = seatNonBid.Get()
}

func getGoogleSDKRejectedResponse(response *openrtb2.BidResponse, ao analytics.AuctionObject) *openrtb2.BidResponse {
	rCtx := pubmatic.GetRequestCtx(ao.HookExecutionOutcome)
	if response == nil || rCtx == nil || rCtx.Endpoint != models.EndpointGoogleSDK {
		return response
	}

	if !rCtx.GoogleSDK.Reject && response.NBR == nil {
		return response
	}

	ext := []byte("{}")
	if rCtx.Debug {
		// Copy only "owlogger" from original Ext if it exists
		if owLoggerVal, _, _, err := jsonparser.Get(response.Ext, models.LoggerKey); err == nil {
			if updatedExt, err := jsonparser.Set(ext, []byte(strconv.Quote(string(owLoggerVal))), models.LoggerKey); err == nil {
				ext = updatedExt
			}
		}
	}
	//append processing time
	processingTimeValue := time.Since(rCtx.GoogleSDK.StartTime).Milliseconds()
	if updatedExt, err := jsonparser.Set(ext, []byte(strconv.FormatInt(processingTimeValue, 10)), models.ProcessingTime); err == nil {
		ext = updatedExt
	}

	rCtx.GoogleSDK.RejectedBidResponse = &openrtb2.BidResponse{
		ID:  response.ID,
		NBR: response.NBR,
		Ext: ext,
	}
	return rCtx.GoogleSDK.RejectedBidResponse
}

func isAPSIntegration(ao analytics.AuctionObject) bool {
	rCtx := pubmatic.GetRequestCtx(ao.HookExecutionOutcome)
	if rCtx != nil && rCtx.Endpoint == models.EndpointAPS {
		return true
	}
	return false
}
