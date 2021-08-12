package sharethrough

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy/ccpa"
)

const defaultTmax = 10000 // 10 sec

type StrAdSeverParams struct {
	Pkey               string
	BidID              string
	GPID               string
	ConsentRequired    bool
	ConsentString      string
	USPrivacySignal    string
	InstantPlayCapable bool
	Iframe             bool
	Height             uint64
	Width              uint64
	TheTradeDeskUserId string
	SharethroughUserId string
}

type StrOpenRTBInterface interface {
	requestFromOpenRTB(openrtb2.Imp, *openrtb2.BidRequest, string) (*adapters.RequestData, error)
	responseToOpenRTB([]byte, *adapters.RequestData) (*adapters.BidderResponse, []error)
}

type StrAdServerUriInterface interface {
	buildUri(StrAdSeverParams) string
	parseUri(string) (*StrAdSeverParams, error)
}

type UserAgentParsers struct {
	ChromeVersion    *regexp.Regexp
	ChromeiOSVersion *regexp.Regexp
	SafariVersion    *regexp.Regexp
}

type ButlerRequestBody struct {
	BlockedAdvDomains []string `json:"badv,omitempty"`
	MaxTimeout        int64    `json:"tmax"`
	Deadline          string   `json:"deadline"`
	BidFloor          float64  `json:"bidfloor,omitempty"`
}

type StrUriHelper struct {
	BaseURI string
	Clock   ClockInterface
}

type StrBodyHelper struct {
	Clock ClockInterface
}

type StrOpenRTBTranslator struct {
	UriHelper        StrAdServerUriInterface
	Util             UtilityInterface
	UserAgentParsers UserAgentParsers
}

func (s StrOpenRTBTranslator) requestFromOpenRTB(imp openrtb2.Imp, request *openrtb2.BidRequest, domain string) (*adapters.RequestData, error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("Origin", domain)
	headers.Add("Referer", request.Site.Page)
	headers.Add("X-Forwarded-For", request.Device.IP)
	headers.Add("User-Agent", request.Device.UA)

	var strImpExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &strImpExt); err != nil {
		return nil, err
	}
	var strImpParams openrtb_ext.ExtImpSharethrough
	if err := json.Unmarshal(strImpExt.Bidder, &strImpParams); err != nil {
		return nil, err
	}

	pKey := strImpParams.Pkey
	userInfo := s.Util.parseUserInfo(request.User)
	height, width := s.Util.getPlacementSize(imp, strImpParams)

	jsonBody, err := (StrBodyHelper{Clock: s.Util.getClock()}).buildBody(request, strImpParams)
	if err != nil {
		return nil, err
	}

	var gpid string
	if strImpParams.Data != nil && strImpParams.Data.PBAdSlot != "" {
		gpid = strImpParams.Data.PBAdSlot
	}

	usPolicySignal := ""
	if usPolicy, err := ccpa.ReadFromRequest(request); err == nil {
		usPolicySignal = usPolicy.Consent
	}

	return &adapters.RequestData{
		Method: "POST",
		Uri: s.UriHelper.buildUri(StrAdSeverParams{
			Pkey:               pKey,
			BidID:              imp.ID,
			GPID:               gpid,
			ConsentRequired:    s.Util.gdprApplies(request),
			ConsentString:      userInfo.Consent,
			USPrivacySignal:    usPolicySignal,
			Iframe:             strImpParams.Iframe,
			Height:             uint64(height),
			Width:              uint64(width),
			InstantPlayCapable: s.Util.canAutoPlayVideo(request.Device.UA, s.UserAgentParsers),
			TheTradeDeskUserId: userInfo.TtdUid,
			SharethroughUserId: userInfo.StxUid,
		}),
		Body:    jsonBody,
		Headers: headers,
	}, nil
}

func (s StrOpenRTBTranslator) responseToOpenRTB(strRawResp []byte, btlrReq *adapters.RequestData) (*adapters.BidderResponse, []error) {
	var errs []error

	var strResp openrtb_ext.ExtImpSharethroughResponse
	if err := json.Unmarshal(strRawResp, &strResp); err != nil {
		return nil, []error{&errortypes.BadInput{Message: "Unable to parse response JSON"}}
	}
	bidResponse := adapters.NewBidderResponse()

	bidResponse.Currency = "USD"
	typedBid := &adapters.TypedBid{BidType: openrtb_ext.BidTypeBanner}

	if len(strResp.Creatives) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "No creative provided"})
		return nil, errs
	}
	creative := strResp.Creatives[0]

	btlrParams, parseHBUriErr := s.UriHelper.parseUri(btlrReq.Uri)
	if parseHBUriErr != nil {
		errs = append(errs, &errortypes.BadInput{Message: parseHBUriErr.Error()})
		return nil, errs
	}

	adm, admErr := s.Util.getAdMarkup(strRawResp, strResp, btlrParams)
	if admErr != nil {
		errs = append(errs, &errortypes.BadServerResponse{Message: admErr.Error()})
		return nil, errs
	}

	bid := &openrtb2.Bid{
		AdID:   strResp.AdServerRequestID,
		ID:     strResp.BidID,
		ImpID:  btlrParams.BidID,
		Price:  creative.CPM,
		CID:    creative.Metadata.CampaignKey,
		CrID:   creative.Metadata.CreativeKey,
		DealID: creative.Metadata.DealID,
		AdM:    adm,
		H:      int64(btlrParams.Height),
		W:      int64(btlrParams.Width),
	}

	typedBid.Bid = bid
	bidResponse.Bids = append(bidResponse.Bids, typedBid)

	return bidResponse, errs
}

func (h StrBodyHelper) buildBody(request *openrtb2.BidRequest, strImpParams openrtb_ext.ExtImpSharethrough) (body []byte, err error) {
	timeout := request.TMax
	if timeout == 0 {
		timeout = defaultTmax
	}

	body, err = json.Marshal(ButlerRequestBody{
		BlockedAdvDomains: request.BAdv,
		MaxTimeout:        timeout,
		Deadline:          h.Clock.now().Add(time.Duration(timeout) * time.Millisecond).Format(time.RFC3339Nano),
		BidFloor:          strImpParams.BidFloor,
	})

	return
}

func (h StrUriHelper) buildUri(params StrAdSeverParams) string {
	v := url.Values{}
	v.Set("placement_key", params.Pkey)
	v.Set("bidId", params.BidID)
	if params.GPID != "" {
		v.Set("gpid", params.GPID)
	}
	v.Set("consent_required", fmt.Sprintf("%t", params.ConsentRequired))
	v.Set("consent_string", params.ConsentString)
	if params.USPrivacySignal != "" {
		v.Set("us_privacy", params.USPrivacySignal)
	}
	if params.TheTradeDeskUserId != "" {
		v.Set("ttduid", params.TheTradeDeskUserId)
	}
	if params.SharethroughUserId != "" {
		v.Set("stxuid", params.SharethroughUserId)
	}

	v.Set("instant_play_capable", fmt.Sprintf("%t", params.InstantPlayCapable))
	v.Set("stayInIframe", fmt.Sprintf("%t", params.Iframe))
	v.Set("height", strconv.FormatUint(params.Height, 10))
	v.Set("width", strconv.FormatUint(params.Width, 10))

	v.Set("adRequestAt", h.Clock.now().Format(time.RFC3339Nano))
	v.Set("supplyId", supplyId)
	v.Set("strVersion", strconv.FormatInt(strVersion, 10))

	return h.BaseURI + "?" + v.Encode()
}

func (h StrUriHelper) parseUri(uri string) (*StrAdSeverParams, error) {
	btlrUrl, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	params := btlrUrl.Query()
	height, err := strconv.ParseUint(params.Get("height"), 10, 64)
	if err != nil {
		return nil, err
	}

	width, err := strconv.ParseUint(params.Get("width"), 10, 64)
	if err != nil {
		return nil, err
	}

	stayInIframe, err := strconv.ParseBool(params.Get("stayInIframe"))
	if err != nil {
		stayInIframe = false
	}

	consentRequired, err := strconv.ParseBool(params.Get("consent_required"))
	if err != nil {
		consentRequired = false
	}

	return &StrAdSeverParams{
		Pkey:            params.Get("placement_key"),
		BidID:           params.Get("bidId"),
		Iframe:          stayInIframe,
		Height:          height,
		Width:           width,
		ConsentRequired: consentRequired,
		ConsentString:   params.Get("consent_string"),
	}, nil
}
