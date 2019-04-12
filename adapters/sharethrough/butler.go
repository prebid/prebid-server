package sharethrough

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/url"
	"strconv"
)

type hbUriParams struct {
	Pkey               string
	BidID              string
	ConsentRequired    bool
	ConsentString      string
	InstantPlayCapable bool
	Iframe             bool
	Height             uint64
	Width              uint64
}

func butlerToOpenRTBResponse(btlrReq *adapters.RequestData, strResp openrtb_ext.ExtImpSharethroughResponse) (*adapters.BidderResponse, []error) {
	var errs []error
	bidResponse := adapters.NewBidderResponse()

	bidResponse.Currency = "USD"
	typedBid := &adapters.TypedBid{BidType: openrtb_ext.BidTypeNative}
	creative := strResp.Creatives[0]

	btlrParams, _ := parseHBUri(btlrReq.Uri)

	bid := &openrtb.Bid{
		AdID:   strResp.AdServerRequestID,
		ID:     strResp.BidID,
		ImpID:  btlrParams.BidID,
		Price:  creative.CPM,
		CID:    creative.Metadata.CampaignKey,
		CrID:   creative.Metadata.CreativeKey,
		DealID: creative.Metadata.DealID,
		AdM:    getAdMarkup(strResp, btlrParams),
		H:      btlrParams.Height,
		W:      btlrParams.Width,
	}

	typedBid.Bid = bid
	bidResponse.Bids = append(bidResponse.Bids, typedBid)

	return bidResponse, errs
}

func generateHBUri(baseUrl string, params hbUriParams, app *openrtb.App) string {
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
		var err error
		version, err = jsonparser.GetString(app.Ext, "prebid", "version")
		if err == nil {
			// todo: handle error
			fmt.Printf("Error extracting version: %+v", err)
		}
	}

	v.Set("hbVersion", version)
	v.Set("hbSource", hbSource)
	v.Set("strVersion", strVersion)

	return baseUrl + "?" + v.Encode()
}

func parseHBUri(uri string) (*hbUriParams, error) {
	btlrUrl, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	params := btlrUrl.Query()
	height, _ := strconv.ParseUint(params.Get("height"), 10, 64)
	width, _ := strconv.ParseUint(params.Get("width"), 10, 64)

	return &hbUriParams{
		Pkey:            params.Get("placement_key"),
		BidID:           params.Get("bidId"),
		Iframe:          params.Get("stayInIframe") == "true",
		Height:          height,
		Width:           width,
		ConsentRequired: params.Get("consent_required") == "true",
		ConsentString:   params.Get("consent_string"),
	}, nil
}
