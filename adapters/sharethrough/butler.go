package sharethrough

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

type StrAdSeverParams struct {
	Pkey               string
	BidID              string
	ConsentRequired    bool
	ConsentString      string
	InstantPlayCapable bool
	Iframe             bool
	Height             uint64
	Width              uint64
	TheTradeDeskUserId string
}

type StrOpenRTBInterface interface {
	requestFromOpenRTB(openrtb.Imp, *openrtb.BidRequest, string) (*adapters.RequestData, error)
	responseToOpenRTB(openrtb_ext.ExtImpSharethroughResponse, *adapters.RequestData) (*adapters.BidderResponse, []error)
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

type StrUriHelper struct {
	BaseURI string
}

type StrOpenRTBTranslator struct {
	UriHelper        StrAdServerUriInterface
	Util             UtilityInterface
	UserAgentParsers UserAgentParsers
}

func (s StrOpenRTBTranslator) requestFromOpenRTB(imp openrtb.Imp, request *openrtb.BidRequest, domain string) (*adapters.RequestData, error) {
	headers := http.Header{}
	headers.Add("Content-Type", "text/plain;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("Origin", domain)
	headers.Add("X-Forwarded-For", request.Device.IP)
	headers.Add("User-Agent", request.Device.UA)

	var strImpExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &strImpExt); err != nil {
		return nil, err
	}
	var strImpParams openrtb_ext.ExtImpSharethroughExt
	if err := json.Unmarshal(strImpExt.Bidder, &strImpParams); err != nil {
		return nil, err
	}

	pKey := strImpParams.Pkey
	userInfo := s.Util.parseUserExt(request.User)

	var height, width uint64
	if len(strImpParams.IframeSize) >= 2 {
		height, width = uint64(strImpParams.IframeSize[0]), uint64(strImpParams.IframeSize[1])
	} else {
		height, width = s.Util.getPlacementSize(imp.Banner.Format)
	}

	return &adapters.RequestData{
		Method: "POST",
		Uri: s.UriHelper.buildUri(StrAdSeverParams{
			Pkey:               pKey,
			BidID:              imp.ID,
			ConsentRequired:    s.Util.gdprApplies(request),
			ConsentString:      userInfo.Consent,
			Iframe:             strImpParams.Iframe,
			Height:             height,
			Width:              width,
			InstantPlayCapable: s.Util.canAutoPlayVideo(request.Device.UA, s.UserAgentParsers),
			TheTradeDeskUserId: userInfo.TtdUid,
		}),
		Body:    nil,
		Headers: headers,
	}, nil
}

func (s StrOpenRTBTranslator) responseToOpenRTB(strResp openrtb_ext.ExtImpSharethroughResponse, btlrReq *adapters.RequestData) (*adapters.BidderResponse, []error) {
	var errs []error
	bidResponse := adapters.NewBidderResponse()

	bidResponse.Currency = "USD"
	typedBid := &adapters.TypedBid{BidType: openrtb_ext.BidTypeNative}

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

	adm, admErr := s.Util.getAdMarkup(strResp, btlrParams)
	if admErr != nil {
		errs = append(errs, &errortypes.BadServerResponse{Message: admErr.Error()})
		return nil, errs
	}

	bid := &openrtb.Bid{
		AdID:   strResp.AdServerRequestID,
		ID:     strResp.BidID,
		ImpID:  btlrParams.BidID,
		Price:  creative.CPM,
		CID:    creative.Metadata.CampaignKey,
		CrID:   creative.Metadata.CreativeKey,
		DealID: creative.Metadata.DealID,
		AdM:    adm,
		H:      btlrParams.Height,
		W:      btlrParams.Width,
	}

	typedBid.Bid = bid
	bidResponse.Bids = append(bidResponse.Bids, typedBid)

	return bidResponse, errs
}

func (h StrUriHelper) buildUri(params StrAdSeverParams) string {
	v := url.Values{}
	v.Set("placement_key", params.Pkey)
	v.Set("bidId", params.BidID)
	v.Set("consent_required", fmt.Sprintf("%t", params.ConsentRequired))
	v.Set("consent_string", params.ConsentString)
	if params.TheTradeDeskUserId != "" {
		v.Set("ttduid", params.TheTradeDeskUserId)
	}

	v.Set("instant_play_capable", fmt.Sprintf("%t", params.InstantPlayCapable))
	v.Set("stayInIframe", fmt.Sprintf("%t", params.Iframe))
	v.Set("height", strconv.FormatUint(params.Height, 10))
	v.Set("width", strconv.FormatUint(params.Width, 10))

	v.Set("supplyId", supplyId)
	v.Set("strVersion", strVersion)

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
