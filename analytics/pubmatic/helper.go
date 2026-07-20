package pubmatic

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/pubmatic/mhttp"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/wakanda"
)

const parseUrlFormat = "json"

// PrepareLoggerURL returns the url for OW logger call
func PrepareLoggerURL(wlog *WloggerRecord, loggerURL string, gdprEnabled int) string {
	if wlog == nil {
		return ""
	}
	v := url.Values{}

	jsonString, err := json.Marshal(wlog.record)
	if err != nil {
		return ""
	}

	v.Set(models.WLJSON, string(jsonString))
	v.Set(models.WLPUBID, strconv.Itoa(wlog.PubID))
	if gdprEnabled == 1 {
		v.Set(models.WLGDPR, strconv.Itoa(gdprEnabled))
	}
	queryString := v.Encode()

	finalLoggerURL := loggerURL + "?" + queryString
	return finalLoggerURL
}

// getGdprEnabledFlag returns gdpr flag set in the partner config
func getGdprEnabledFlag(partnerConfigMap map[int]map[string]string) int {
	gdpr := 0
	if val := partnerConfigMap[models.VersionLevelConfigID][models.GDPR_ENABLED]; val != "" {
		gdpr, _ = strconv.Atoi(val)
	}
	return gdpr
}

// send function will send the owlogger to analytics endpoint
var send = func(rCtx *models.RequestCtx, url string, headers http.Header, mhc mhttp.MultiHttpContextInterface) {
	startTime := time.Now()
	hc, _ := mhttp.NewHttpCall(url, "")

	for k, v := range headers {
		if len(v) != 0 {
			hc.AddHeader(k, v[0])
		}
	}

	if rCtx.KADUSERCookie != nil {
		hc.AddCookie(models.KADUSERCOOKIE, rCtx.KADUSERCookie.Value)
	}

	mhc.AddHttpCall(hc)
	_, erc := mhc.Execute()
	if erc != 0 {
		glog.Errorf("Failed to send the owlogger for pub:[%d], profile:[%d], version:[%d].",
			rCtx.PubID, rCtx.ProfileID, rCtx.VersionID)

		// we will not record at version level in prometheus metric
		rCtx.MetricsEngine.RecordPublisherWrapperLoggerFailure(rCtx.PubIDStr)
		return
	}
	rCtx.MetricsEngine.RecordSendLoggerDataTime(time.Since(startTime))
	// TODO: this will increment HB specific metric (ow_pbs_sshb_*), verify labels
}

// decodeAPSAdmGzipBase64 reverses APS bid.adm encoding (gzip + base64.StdEncoding) from sdk/aps.ApplyAPSResponse.
func decodeAPSAdmGzipBase64(adm string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(adm)
	if err != nil {
		return nil, err
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

// RestoreBidResponse restores the original bid response for AppLovinMax / APS / other SDK paths from wrapper signal or compressed adm.
func RestoreBidResponse(rctx *models.RequestCtx, ao analytics.AuctionObject) error {
	if !sdkutils.IsSdkBiddingEndpoint(rctx.Endpoint) {
		return nil
	}

	if rctx.AppLovinMax.Reject || rctx.GoogleSDK.Reject || rctx.UnityLevelPlay.Reject || rctx.APS.Reject {
		return nil
	}

	if ao.Response.NBR != nil {
		return nil
	}

	if len(ao.Response.SeatBid) == 0 || len(ao.Response.SeatBid[0].Bid) == 0 {
		return errors.New("seatbid or bid not found in the response")
	}

	orignalResponse := &openrtb2.BidResponse{}
	if rctx.Endpoint == models.EndpointAppLovinMax {
		signalData := map[string]string{}
		if err := json.Unmarshal(ao.Response.SeatBid[0].Bid[0].Ext, &signalData); err != nil {
			return err
		}

		if val, ok := signalData[models.SignalData]; !ok || val == "" {
			return errors.New("signal data not found in the response")
		}

		if err := json.Unmarshal([]byte(signalData[models.SignalData]), orignalResponse); err != nil {
			return err
		}
	}

	if rctx.Endpoint == models.EndpointGoogleSDK {
		renderingData, err := jsonparser.GetString(ao.Response.SeatBid[0].Bid[0].Ext, "sdk_rendered_ad", "rendering_data")
		if err = json.Unmarshal([]byte(renderingData), orignalResponse); err != nil {
			return err
		}
	}

	if rctx.Endpoint == models.EndpointUnityLevelPlay {
		if err := json.Unmarshal([]byte(ao.Response.SeatBid[0].Bid[0].AdM), orignalResponse); err != nil {
			return err
		}
	}

	if rctx.Endpoint == models.EndpointAPS {
		payload, err := decodeAPSAdmGzipBase64(ao.Response.SeatBid[0].Bid[0].AdM)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(payload, orignalResponse); err != nil {
			return err
		}
	}
	*ao.Response = *orignalResponse
	return nil
}

func (wlog *WloggerRecord) logProfileMetaData(rctx *models.RequestCtx) {
	wlog.ProfileType = rctx.ProfileType
	wlog.ProfileTypePlatform = rctx.ProfileTypePlatform
	wlog.AppPlatform = rctx.AppPlatform
	if rctx.AppIntegrationPath != nil && *rctx.AppIntegrationPath >= 0 {
		wlog.AppIntegrationPath = rctx.AppIntegrationPath
	}
	if rctx.AppSubIntegrationPath != nil && *rctx.AppSubIntegrationPath >= 0 {
		wlog.AppSubIntegrationPath = rctx.AppSubIntegrationPath
	}
}

func setWakandaObject(rCtx *models.RequestCtx, ao *analytics.AuctionObject, loggerURL string) {
	if rCtx.WakandaDebug != nil && rCtx.WakandaDebug.IsEnable() {
		setWakandaWinningBidFlag(rCtx.WakandaDebug, ao.Response)
		parseURL, err := url.Parse(loggerURL)
		if err != nil {
			glog.Errorf("Failed to parse loggerURL while setting wakanda object err: %s", err.Error())
		}
		if parseURL != nil {
			jsonParam := parseURL.Query().Get(parseUrlFormat)
			rCtx.WakandaDebug.SetLogger(json.RawMessage(jsonParam))
		}
		bytes, err := json.Marshal(ao.Response)
		if err != nil {
			glog.Errorf("Failed to marshal ao.Response while setting wakanda object err: %s", err.Error())
		}
		rCtx.WakandaDebug.SetHTTPResponseBodyWriter(string(bytes))
		rCtx.WakandaDebug.SetOpenRTB(ao.RequestWrapper.BidRequest)
		rCtx.WakandaDebug.WriteLogToFiles()
	}
}

// setWakandaWinningBidFlag will set WinningBid flag to true if we are getting any positive bid in response
func setWakandaWinningBidFlag(wakandaDebug wakanda.WakandaDebug, response *openrtb2.BidResponse) {
	if wakandaDebug != nil && response != nil {
		if len(response.SeatBid) > 0 &&
			len(response.SeatBid[0].Bid) > 0 &&
			response.SeatBid[0].Bid[0].Price > 0 {
			wakandaDebug.SetWinningBid(true)
		}
	}
}
