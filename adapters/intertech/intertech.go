package intertech

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	pageIDMacro = "{{page_id}}"
	impIDMacro  = "{{imp_id}}"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var requests []*adapters.RequestData

	referer := getReferer(request)
	cur := getCur(request)

	for _, imp := range request.Imp {
		extImp, err := parseAndValidateImpExt(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		modifiedImp, err := modifyImp(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		modifiedUrl := a.modifyUrl(extImp, referer, cur)

		modRequest := *request
		modRequest.Imp = []openrtb2.Imp{modifiedImp}

		reqData, err := buildRequestData(modRequest, modifiedUrl)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, reqData)
	}

	return requests, errs
}

func parseAndValidateImpExt(imp openrtb2.Imp) (openrtb_ext.ExtImpIntertech, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpIntertech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: unable to parse bidder ext: %s", imp.ID, err),
		}
	}

	var extImp openrtb_ext.ExtImpIntertech
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &extImp); err != nil {
		return openrtb_ext.ExtImpIntertech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: unable to parse intertech ext: %s", imp.ID, err),
		}
	}

	return extImp, nil
}

func modifyImp(imp openrtb2.Imp) (openrtb2.Imp, error) {
	if imp.Banner != nil {
		banner, err := updateBanner(imp.Banner)
		if err != nil {
			return openrtb2.Imp{}, &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%s: %s", imp.ID, err.Error()),
			}
		}
		imp.Banner = banner
	}
	return imp, nil
}

func updateBanner(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	bannerCopy := *banner
	if bannerCopy.W == nil || bannerCopy.H == nil || *bannerCopy.W == 0 || *bannerCopy.H == 0 {
		if len(bannerCopy.Format) > 0 {
			w := bannerCopy.Format[0].W
			h := bannerCopy.Format[0].H
			bannerCopy.W = &w
			bannerCopy.H = &h
		} else {
			return nil, fmt.Errorf("Invalid sizes provided for Banner")
		}
	}
	return &bannerCopy, nil
}

func (a *adapter) modifyUrl(extImp openrtb_ext.ExtImpIntertech, referer, cur string) string {
	pageStr := strconv.Itoa(extImp.PageID)
	impStr := strconv.Itoa(extImp.ImpID)

	resolvedUrl := strings.ReplaceAll(a.endpoint, pageIDMacro, url.QueryEscape(pageStr))
	resolvedUrl = strings.ReplaceAll(resolvedUrl, impIDMacro, url.QueryEscape(impStr))

	if referer != "" {
		resolvedUrl += "&target-ref=" + url.QueryEscape(referer)
	}

	if cur != "" {
		resolvedUrl += "&ssp-cur=" + cur
	}

	return resolvedUrl
}

func buildRequestData(bidRequest openrtb2.BidRequest, uri string) (*adapters.RequestData, error) {
	body, err := jsonutil.Marshal(bidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if bidRequest.Device != nil {
		if bidRequest.Device.UA != "" {
			headers.Add("User-Agent", bidRequest.Device.UA)
		}
		if bidRequest.Device.IP != "" {
			headers.Add("X-Forwarded-For", bidRequest.Device.IP)
			headers.Add("X-Real-Ip", bidRequest.Device.IP)
		}
		if bidRequest.Device.Language != "" {
			headers.Add("Accept-Language", bidRequest.Device.Language)
		}
	}
	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(bidRequest.Imp),
	}, nil
}

func getReferer(request *openrtb2.BidRequest) string {
	if request.Site != nil {
		return request.Site.Page
	}
	return ""
}

func getCur(request *openrtb2.BidRequest) string {
	if len(request.Cur) > 0 {
		return request.Cur[0]
	}
	return ""
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) != 0 {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", fmt.Errorf("could not define media type for impression: %s", bid.ImpID)
}
