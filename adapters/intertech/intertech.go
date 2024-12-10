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

type ExtImpIntertech struct {
	PageID int `json:"page_id"`
	ImpID  int `json:"imp_id"`
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

		modifiedUrl, err := a.modifyUrl(extImp, referer, cur)
		if err != nil {
			errs = append(errs, err)
			continue
		}

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

func parseAndValidateImpExt(imp openrtb2.Imp) (ExtImpIntertech, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return ExtImpIntertech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: unable to parse bidder ext: %s", imp.ID, err),
		}
	}

	var extImp ExtImpIntertech
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &extImp); err != nil {
		return ExtImpIntertech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: unable to parse intertech ext: %s", imp.ID, err),
		}
	}

	if extImp.PageID == 0 {
		return ExtImpIntertech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: missing param page_id", imp.ID),
		}
	}
	if extImp.ImpID == 0 {
		return ExtImpIntertech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: missing param imp_id", imp.ID),
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
		return imp, nil
	}
	if imp.Native != nil {
		return imp, nil
	}
	return openrtb2.Imp{}, &errortypes.BadInput{
		Message: fmt.Sprintf("Intertech only supports banner and native types. Ignoring imp id=%s", imp.ID),
	}
}

func updateBanner(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner == nil {
		return nil, fmt.Errorf("banner is null")
	}
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

func (a *adapter) modifyUrl(extImp ExtImpIntertech, referer, cur string) (string, error) {
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

	return resolvedUrl, nil
}

func buildRequestData(bidRequest openrtb2.BidRequest, uri string) (*adapters.RequestData, error) {
	body, err := jsonutil.Marshal(bidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

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

func (a *adapter) MakeBids(req *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", responseData.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to decode bid response: %s", err.Error()),
		}}
	}

	seatBids := bidResp.SeatBid
	if seatBids == nil {
		return &adapters.BidderResponse{
			Currency: bidResp.Cur,
			Bids:     make([]*adapters.TypedBid, 0),
		}, nil
	}

	if len(seatBids) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "SeatBids is empty",
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(seatBids))
	bidderResponse.Currency = bidResp.Cur

	var errs []error
	for _, seatBid := range seatBids {
		for _, bid := range seatBid.Bid {
			bidType, err := getBidTypeFromImps(bid.ImpID, req.Imp)
			if err != nil {
				errs = append(errs, &errortypes.BadServerResponse{Message: err.Error()})
				continue
			}
			typedBid := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, errs
}

func getBidTypeFromImps(bidImpID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == bidImpID {
			return resolveImpType(imp)
		}
	}
	return "", fmt.Errorf("Invalid bid imp ID %s does not match any imp IDs from the original bid request", bidImpID)
}

func resolveImpType(imp openrtb2.Imp) (openrtb_ext.BidType, error) {
	if imp.Native != nil {
		return openrtb_ext.BidTypeNative, nil
	}
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner, nil
	}
	return "", fmt.Errorf("Processing an invalid impression; cannot resolve impression type")
}
