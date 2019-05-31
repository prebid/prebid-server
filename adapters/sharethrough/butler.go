package sharethrough

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
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
}

type StrOpenRTBInterface interface {
	requestFromOpenRTB(openrtb.Imp, *openrtb.BidRequest) (*adapters.RequestData, error)
	responseToOpenRTB(openrtb_ext.ExtImpSharethroughResponse, *adapters.RequestData) (*adapters.BidderResponse, []error)
}

type StrAdServerUriInterface interface {
	buildUri(StrAdSeverParams, *openrtb.App) string
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

func (s StrOpenRTBTranslator) requestFromOpenRTB(imp openrtb.Imp, request *openrtb.BidRequest) (*adapters.RequestData, error) {
	headers := http.Header{}
	headers.Add("Content-Type", "text/plain;charset=utf-8")
	headers.Add("Accept", "application/json")

	var extBtlrParams openrtb_ext.ExtImpSharethroughExt
	if err := json.Unmarshal(imp.Ext, &extBtlrParams); err != nil {
		return nil, err
	}

	pKey := extBtlrParams.Bidder.Pkey

	var height, width uint64
	if len(extBtlrParams.Bidder.IframeSize) >= 2 {
		height, width = uint64(extBtlrParams.Bidder.IframeSize[0]), uint64(extBtlrParams.Bidder.IframeSize[1])
	} else {
		height, width = s.Util.getPlacementSize(imp.Banner.Format)
	}

	return &adapters.RequestData{
		Method: "POST",
		Uri: s.UriHelper.buildUri(StrAdSeverParams{
			Pkey:               pKey,
			BidID:              imp.ID,
			ConsentRequired:    s.Util.gdprApplies(request),
			ConsentString:      s.Util.gdprConsentString(request),
			Iframe:             extBtlrParams.Bidder.Iframe,
			Height:             height,
			Width:              width,
			InstantPlayCapable: s.Util.canAutoPlayVideo(request.Device.UA, s.UserAgentParsers),
		}, request.App),
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

func (h StrUriHelper) buildUri(params StrAdSeverParams, app *openrtb.App) string {
	v := url.Values{}
	v.Set("placement_key", params.Pkey)
	v.Set("bidId", params.BidID)
	v.Set("consent_required", fmt.Sprintf("%t", params.ConsentRequired))
	v.Set("consent_string", params.ConsentString)

	v.Set("instant_play_capable", fmt.Sprintf("%t", params.InstantPlayCapable))
	v.Set("stayInIframe", fmt.Sprintf("%t", params.Iframe))
	v.Set("height", strconv.FormatUint(params.Height, 10))
	v.Set("width", strconv.FormatUint(params.Width, 10))

	var version string

	if app != nil {
		// Skipping error handling here because it should fall through to unknown in the flow
		version, _ = jsonparser.GetString(app.Ext, "prebid", "version")
	}

	if len(version) == 0 {
		version = "unknown"
	}

	v.Set("hbVersion", version)
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

	return &StrAdSeverParams{
		Pkey:            params.Get("placement_key"),
		BidID:           params.Get("bidId"),
		Iframe:          params.Get("stayInIframe") == "true",
		Height:          height,
		Width:           width,
		ConsentRequired: params.Get("consent_required") == "true",
		ConsentString:   params.Get("consent_string"),
	}, nil
}
