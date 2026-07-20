package openwrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	v25 "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/endpoints/legacy/openrtb/v25"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/aps"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/googlesdk"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/unitylevelplay"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/wakanda"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/usersync"
	uuid "github.com/satori/go.uuid"
)

const (
	OpenWrapAuction      = "/pbs/openrtb2/auction"
	OpenWrapV25          = "/openrtb/2.5"
	OpenWrapV25Video     = "/openrtb/2.5/video"
	OpenWrapOpenRTBVideo = "/video/openrtb"
	OpenWrapVAST         = "/video/vast"
	OpenWrapJSON         = "/video/json"
	OpenWrapAmp          = "/amp"
)

func (m OpenWrap) handleEntrypointHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (result hookstage.HookResult[hookstage.EntrypointPayload], err error) {
	queryParams := payload.Request.URL.Query()
	source := queryParams.Get("source") //source query param to identify /openrtb2/auction type

	rCtx := models.RequestCtx{}
	var endpoint string
	var requestExtWrapper models.RequestExtWrapper
	defer func() {
		if result.Reject {
			if rCtx.PubIDStr == "" {
				rCtx.PubIDStr = "0"
			}
			m.metricEngine.RecordBadRequests(endpoint, rCtx.PubIDStr, getPubmaticErrorCode(openrtb3.NoBidReason(result.NbrCode)))
			if glog.V(models.LogLevelDebug) {
				glog.Infof("[bad_request] pubid:[%d] profid:[%d] endpoint:[%s] nbr:[%d] query_params:[%s] body:[%s]",
					rCtx.PubID, rCtx.ProfileID, rCtx.Endpoint, result.NbrCode, queryParams.Encode(), string(payload.Body))
			}
			return
		}
		result.ModuleContext = make(hookstage.ModuleContext)
		result.ModuleContext["rctx"] = rCtx
	}()

	rCtx.Sshb = queryParams.Get("sshb")
	//Do not execute the module for requests processed in SSHB(8001)
	if rCtx.Sshb == models.Enabled {
		rCtx.VastUnWrap.Enabled = getVastUnwrapperEnable(payload.Request.Context(), models.VastUnwrapperEnableKey)
		rCtx.DeviceCtx.IP = models.GetIP(payload.Request)
		rCtx.DeviceCtx.UA = payload.Request.Header.Get("User-Agent")
		return result, nil
	}
	endpoint = GetEndpoint(payload.Request.URL.Path, source, queryParams.Get(models.Agent))
	if endpoint == models.EndpointHybrid {
		rCtx.Endpoint = models.EndpointHybrid
		return result, nil
	}

	originalRequestBody := payload.Body

	if endpoint == models.EndpointAppLovinMax {
		rCtx.MetricsEngine = m.metricEngine
		// updating body locally to access updated fields from signal
		payload.Body = updateAppLovinMaxRequest(payload.Body, &rCtx)
		result.ChangeSet.AddMutation(func(ep hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			ep.Body = payload.Body
			return ep, nil
		}, hookstage.MutationUpdate, "update-max-app-lovin-request")
	}

	if endpoint == models.EndpointGoogleSDK {
		rCtx.MetricsEngine = m.metricEngine
		// Update fields from signal
		payload.Body = googlesdk.ModifyRequestWithGoogleSDKParams(payload.Body, &rCtx, m.features)
		result.ChangeSet.AddMutation(func(ep hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			ep.Body = payload.Body
			return ep, nil
		}, hookstage.MutationUpdate, "update-google-sdk-request")
	}

	if endpoint == models.EndpointUnityLevelPlay {
		rCtx.MetricsEngine = m.metricEngine
		// Update fields from signal
		ulp := unitylevelplay.NewLevelPlay(m.metricEngine)
		payload.Body = ulp.ModifyRequestWithUnityLevelPlayParams(payload.Body, &rCtx)
		result.ChangeSet.AddMutation(func(ep hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			ep.Body = payload.Body
			return ep, nil
		}, hookstage.MutationUpdate, "update-unity-level-play-request")
	}

	// Publisher id from raw body after any SDK merge (e.g. Google SDK injects app.publisher.id from signal).
	pubIdStr, _, _, errs := getAccountIdFromRawRequest(false, nil, payload.Body)
	if len(errs) > 0 {
		result.Reject = true
		result.NbrCode = int(nbr.InvalidPublisherID)
		result.Errors = append(result.Errors, errs[0].Error())
		return result, errs[0]
	}

	if endpoint == models.EndpointAPS {
		var errAps error
		var nbrCode openrtb3.NoBidReason
		payload.Body, errAps, nbrCode = enrichApsRequest(payload.Body, m.cache, m.metricEngine, pubIdStr)
		if errAps != nil {
			result.Reject = true
			glog.Errorf("aps_slot_mapping: %v", errAps)
			result.NbrCode = int(nbrCode)
			result.Errors = append(result.Errors, errAps.Error())
			return result, errAps
		}
		aps := aps.NewAPS(m.metricEngine)
		payload.Body = aps.ModifyRequestWithAPSParams(payload.Body, &rCtx)
		result.ChangeSet.AddMutation(func(ep hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			ep.Body = payload.Body
			return ep, nil
		}, hookstage.MutationUpdate, "update-aps-request")
	}

	// init default for all modules
	result.Reject = true

	requestExtWrapper, err = GetRequestWrapper(payload, result, endpoint)
	if err != nil {
		result.NbrCode = int(nbr.InvalidRequestWrapperExtension)
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	if requestExtWrapper.ProfileId <= 0 {
		result.NbrCode = int(nbr.InvalidProfileID)
		result.Errors = append(result.Errors, "ErrMissingProfileID")
		return result, err
	}

	requestDebug, _ := jsonparser.GetBoolean(payload.Body, "ext", "prebid", "debug")
	// SDK entrypoint handlers above decode signal into rCtx.SignalRequest; preserve it across re-init.
	signalRequest := rCtx.SignalRequest
	rCtx = models.RequestCtx{
		StartTime:          time.Now().Unix(),
		Debug:              queryParams.Get(models.Debug) == "1" || requestDebug,
		ProfileID:          requestExtWrapper.ProfileId,
		DisplayID:          requestExtWrapper.VersionId,
		DisplayVersionID:   requestExtWrapper.VersionId,
		SupportDeals:       requestExtWrapper.SupportDeals,
		ABTestConfig:       requestExtWrapper.ABTestConfig,
		SSAuction:          requestExtWrapper.SSAuctionFlag,
		SummaryDisable:     requestExtWrapper.SumryDisableFlag,
		LoggerImpressionID: requestExtWrapper.LoggerImpressionID,
		ClientConfigFlag:   requestExtWrapper.ClientConfigFlag,
		SSAI:               requestExtWrapper.SSAI,
		AdruleFlag:         requestExtWrapper.Video.AdruleFlag,
		IsCTVRequest:       models.IsCTVAPIRequest(payload.Request.URL.Path),
		DeviceCtx: models.DeviceCtx{UA: payload.Request.Header.Get("User-Agent"),
			IP: models.GetIP(payload.Request)},
		TrackerEndpoint:           m.cfg.Tracker.Endpoint,
		VideoErrorTrackerEndpoint: m.cfg.Tracker.VideoErrorTrackerEndpoint,
		Aliases:                   make(map[string]string),
		ImpBidCtx:                 make(map[string]models.ImpCtx),
		PrebidBidderCode:          make(map[string]string),
		BidderResponseTimeMillis:  make(map[string]int),
		ProfileIDStr:              strconv.Itoa(requestExtWrapper.ProfileId),
		Endpoint:                  endpoint,
		MetricsEngine:             m.metricEngine,
		DCName:                    m.cfg.Server.DCName,
		SeatNonBids:               make(map[string][]openrtb_ext.NonBid),
		ParsedUidCookie:           usersync.ReadCookie(payload.Request, usersync.Base64Decoder{}, &config.HostCookie{}),
		TMax:                      m.cfg.Timeout.MaxTimeout,
		Method:                    payload.Request.Method,
		ResponseFormat:            strings.ToLower(strings.TrimSpace(queryParams.Get(models.ResponseFormatKey))),
		RedirectURL:               queryParams.Get(models.OWRedirectURLKey),
		WakandaDebug: &wakanda.Debug{
			Config: m.cfg.Wakanda,
		},
		SendBurl:                        getSendBurl(payload.Body, endpoint),
		ImpCountingMethodEnabledBidders: make(map[string]struct{}),
		GoogleSDK:                       models.GoogleSDK{StartTime: time.Now()},
	}
	rCtx.SignalRequest = signalRequest

	if rCtx.IsCTVRequest {
		// SSAuction will be always 1 for CTV request
		rCtx.SSAuction = 1

		rCtx.ImpAdPodConfig = make(map[string][]models.PodConfig)
	}

	if payload.Request.Header["Cookie"] != nil {
		rCtx.Cookies = payload.Request.Header["Cookie"]
	}

	// only http.ErrNoCookie is returned, we can ignore it
	rCtx.UidCookie, _ = payload.Request.Cookie(models.UidCookieName)
	rCtx.KADUSERCookie, _ = payload.Request.Cookie(models.KADUSERCOOKIE)
	if originCookie, _ := payload.Request.Cookie("origin"); originCookie != nil {
		rCtx.OriginCookie = originCookie.Value
	}

	if rCtx.LoggerImpressionID == "" {
		rCtx.LoggerImpressionID = uuid.NewV4().String()
	}

	rCtx.PubID, err = strconv.Atoi(pubIdStr)
	if err != nil {
		result.NbrCode = int(nbr.InvalidPublisherID)
		result.Errors = append(result.Errors, "ErrInvalidPublisherID")
		return result, fmt.Errorf("invalid publisher id : %v", err)
	}
	rCtx.PubIDStr = pubIdStr
	rCtx.WakandaDebug.EnableIfRequired(pubIdStr, rCtx.ProfileIDStr)
	if rCtx.WakandaDebug.IsEnable() {
		rCtx.WakandaDebug.SetHTTPRequestData(payload.Request, originalRequestBody)
	}

	result.Reject = false

	if rCtx.IsCTVRequest {
		result.ChangeSet.AddMutation(func(ep hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			ep.Request.Body = io.NopCloser(bytes.NewBuffer(ep.Body))
			return ep, nil
		}, hookstage.MutationUpdate, "update-request-body")
	}

	return result, nil
}

func GetRequestWrapper(payload hookstage.EntrypointPayload, result hookstage.HookResult[hookstage.EntrypointPayload], endpoint string) (models.RequestExtWrapper, error) {
	var requestExtWrapper models.RequestExtWrapper
	var err error
	switch endpoint {
	case models.EndpintInappVideo:
		requestExtWrapper, err = v25.ConvertVideoToAuctionRequest(payload, &result)
	case models.EndpointAMP:
		requestExtWrapper, err = models.GetQueryParamRequestExtWrapper(payload.Request)
	case models.EndpointV25:
		fallthrough
	case models.EndpointVideo, models.EndpointORTB, models.EndpointVAST, models.EndpointJson:
		requestExtWrapper, err = models.GetRequestExtWrapper(payload.Body, "ext", "wrapper")
	case models.EndpointAppLovinMax:
		fallthrough
	case models.EndpointWebS2S:
		fallthrough
	default:
		requestExtWrapper, err = models.GetRequestExtWrapper(payload.Body)
	}

	return requestExtWrapper, err
}

func GetEndpoint(path, source string, agent string) string {
	switch path {
	case hookexecution.EndpointAuction:
		switch source {
		case "pbjs":
			return models.EndpointWebS2S
		case "owsdk":
			switch agent {
			case models.AppLovinMaxAgent:
				return models.EndpointAppLovinMax
			case models.GoogleSDKAgent:
				return models.EndpointGoogleSDK
			case models.UnityLevelPlayAgent:
				return models.EndpointUnityLevelPlay
			case models.ApsAgent:
				return models.EndpointAPS
			}
			return models.EndpointV25
		default:
			return models.EndpointHybrid
		}
	case OpenWrapAuction:
		return models.EndpointHybrid
	case OpenWrapV25:
		return models.EndpointV25
	case OpenWrapV25Video:
		return models.EndpintInappVideo
	case OpenWrapAmp:
		return models.EndpointAMP
	case OpenWrapOpenRTBVideo:
		return models.EndpointORTB
	case OpenWrapVAST:
		return models.EndpointVAST
	case OpenWrapJSON:
		return models.EndpointJson
	}
	return ""
}

func getSendBurl(request []byte, endpoint string) bool {
	if sdkutils.IsSdkBiddingEndpoint(endpoint) {
		return true
	}

	//ignore error, default is false
	sendBurl, _ := jsonparser.GetBoolean(request, "ext", "prebid", "bidderparams", "pubmatic", "sendburl")
	return sendBurl
}
