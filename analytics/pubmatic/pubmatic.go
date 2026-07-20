package pubmatic

import (
	"runtime/debug"
	"sync"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/pubmatic/mhttp"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
)

type RequestType string

const (
	COOKIE_SYNC        RequestType = "/cookie_sync"
	AUCTION            RequestType = "/openrtb2/auction"
	VIDEO              RequestType = "/openrtb2/video"
	SETUID             RequestType = "/set_uid"
	AMP                RequestType = "/openrtb2/amp"
	NOTIFICATION_EVENT RequestType = "/event"
)

var ow HTTPLogger
var once sync.Once

// Module that can perform transactional logging
type HTTPLogger struct {
	cfg      config.PubMaticWL
	hostName string
}

// LogAuctionObject prepares the owlogger url and send it to logger endpoint
func (ow HTTPLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	var rCtx *models.RequestCtx
	defer func() {
		if r := recover(); r != nil {
			if rCtx != nil {
				rCtx.MetricsEngine.RecordOpenWrapServerPanicStats(ow.hostName, "LogAuctionObject")
				glog.Errorf("stacktrace:[%s], error:[%v], pubid:[%d], profid:[%d], ver:[%d]", string(debug.Stack()), r, rCtx.PubID, rCtx.ProfileID, rCtx.VersionID)
				return
			}
			glog.Errorf("stacktrace:[%s], error:[%v]", string(debug.Stack()), r)
		}
	}()

	rCtx = GetRequestCtx(ao.HookExecutionOutcome)
	if rCtx == nil {
		// glog.Errorf("Failed to get the request context for AuctionObject - [%v]", ao)
		// add this log once complete header-bidding code is migrated to modules
		return
	}

	if rCtx.LoggerDisabled {
		// logger disabled explicitly for publisher,profile request
		return
	}

	var orignalMaxBidResponse *openrtb2.BidResponse
	if sdkutils.IsSdkBiddingEndpoint(rCtx.Endpoint) {
		orignalMaxBidResponse = new(openrtb2.BidResponse)
		*orignalMaxBidResponse = *ao.Response
	}

	err := RestoreBidResponse(rCtx, *ao)
	if err != nil {
		glog.Errorf("Failed to restore bid response for pub:[%d], profile:[%d], version:[%d], err:[%s].", rCtx.PubID, rCtx.ProfileID, rCtx.VersionID, err.Error())
	}

	loggerURL, headers := GetLogAuctionObjectAsURL(*ao, rCtx, false, false)
	if loggerURL == "" {
		glog.Errorf("Failed to prepare the owlogger for pub:[%d], profile:[%d], version:[%d].",
			rCtx.PubID, rCtx.ProfileID, rCtx.VersionID)
		return
	}

	go send(rCtx, loggerURL, headers, mhttp.NewMultiHttpContext())

	if sdkutils.IsSdkBiddingEndpoint(rCtx.Endpoint) {
		ao.Response = orignalMaxBidResponse
	}

	if sdkutils.IsGoogleSDKResponseRejected(rCtx, *ao) {
		ao.Response = rCtx.GoogleSDK.RejectedBidResponse
	}
	setWakandaObject(rCtx, ao, loggerURL)
}

// Writes VideoObject to file
func (ow HTTPLogger) LogVideoObject(vo *analytics.VideoObject) {
}

// Logs SetUIDObject to file
func (ow HTTPLogger) LogSetUIDObject(so *analytics.SetUIDObject) {
}

// Logs CookieSyncObject to file
func (ow HTTPLogger) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
}

// Logs AmpObject to file
func (ow HTTPLogger) LogAmpObject(ao *analytics.AmpObject) {
}

// Logs NotificationEvent to file
func (ow HTTPLogger) LogNotificationEventObject(ne *analytics.NotificationEvent) {
}

// Shutdown - no op since the analytic module already implements system signal handling
// and trying to close a closed channel will cause panic
func (ow HTTPLogger) Shutdown() {
	glog.Info("[Pubmatic] Shutdown")
}

// Method to initialize the analytic module
func NewHTTPLogger(cfg config.PubMaticWL) analytics.Module {
	once.Do(func() {
		mhttp.Init(cfg.MaxClients, cfg.MaxConnections, cfg.MaxCalls, cfg.RespTimeout)

		ow = HTTPLogger{
			cfg:      cfg,
			hostName: openwrap.GetHostName(),
		}
	})

	return ow
}
