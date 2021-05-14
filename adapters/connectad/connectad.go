package connectad

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type ConnectAdAdapter struct {
	endpoint string
}

type connectadImpExt struct {
	ConnectAd openrtb_ext.ExtImpConnectAd `json:"connectad"`
}

// Builder builds a new instance of the ConnectAd adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &ConnectAdAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *ConnectAdAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error

	if errs := preprocess(request); len(errs) > 0 {
		return nil, append(errs, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in preprocess of Imp"),
		})
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error in packaging request to JSON"),
		}}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.IP != "" {
			addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IPv6)
		}
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		} else {
			addHeaderIfNonEmpty(headers, "DNT", "0")
		}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    data,
		Headers: headers,
	}}, errs
}

func (a *ConnectAdAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpRes.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Invalid Status Returned: %d. Run with request.debug = 1 for more info", httpRes.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(httpRes.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unable to unpackage bid response. Error: %s", err.Error()),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: "banner",
			})
		}
	}

	return bidResponse, nil
}

func preprocess(request *openrtb2.BidRequest) []error {
	impsCount := len(request.Imp)
	errors := make([]error, 0, impsCount)
	resImps := make([]openrtb2.Imp, 0, impsCount)
	secure := int8(0)

	if request.Site != nil && request.Site.Page != "" {
		pageURL, err := url.Parse(request.Site.Page)
		if err == nil && pageURL.Scheme == "https" {
			secure = int8(1)
		}
	}

	for _, imp := range request.Imp {
		cadExt, err := unpackImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		addImpInfo(&imp, &secure, cadExt)

		if err := buildImpBanner(&imp); err != nil {
			errors = append(errors, err)
			continue
		}
		resImps = append(resImps, imp)
	}

	request.Imp = resImps

	return errors
}

func addImpInfo(imp *openrtb2.Imp, secure *int8, cadExt *openrtb_ext.ExtImpConnectAd) {
	imp.TagID = strconv.Itoa(cadExt.SiteID)
	imp.Secure = secure

	if cadExt.Bidfloor != 0 {
		imp.BidFloor = cadExt.Bidfloor
		imp.BidFloorCur = "USD"
	}

	return
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func unpackImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpConnectAd, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Impression id=%s has an Error: %s", imp.ID, err.Error()),
		}
	}

	var cadExt openrtb_ext.ExtImpConnectAd
	if err := json.Unmarshal(bidderExt.Bidder, &cadExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Impression id=%s, has invalid Ext", imp.ID),
		}
	}

	if cadExt.SiteID == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Impression id=%s, has no siteId present", imp.ID),
		}
	}

	return &cadExt, nil
}

func buildImpBanner(imp *openrtb2.Imp) error {
	imp.Ext = nil

	if imp.Banner == nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("We need a Banner Object in the request"),
		}
	}

	if imp.Banner.W == nil && imp.Banner.H == nil {
		bannerCopy := *imp.Banner
		banner := &bannerCopy

		if len(banner.Format) == 0 {
			return &errortypes.BadInput{
				Message: fmt.Sprintf("At least one size is required"),
			}
		}
		format := banner.Format[0]
		banner.Format = banner.Format[1:]
		banner.W = &format.W
		banner.H = &format.H
		imp.Banner = banner
	}

	return nil
}
