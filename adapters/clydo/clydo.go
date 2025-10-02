package clydo

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse endpoint url: %v", err)
	}
	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error

	for _, imp := range request.Imp {
		reqData, err := a.prepareRequest(request, imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		requests = append(requests, reqData)
	}
	return requests, errors
}

func (a *adapter) MakeBids(
	request *openrtb2.BidRequest,
	requestData *adapters.RequestData,
	responseData *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if errResp := checkResponseStatus(responseData); errResp != nil {
		return nil, errResp
	}
	response, err := prepareBidResponse(responseData.Body)
	if err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	bidTypeMap, err := buildBidTypeMap(request.Imp)
	if err != nil {
		return nil, []error{err}
	}
	bids, errors := prepareSeatBids(response.SeatBid, bidTypeMap)
	bidResponse.Bids = bids

	return bidResponse, errors
}

func (a *adapter) prepareRequest(request *openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	params, err := prepareExtParams(imp)
	if err != nil {
		return nil, err
	}
	endpoint, err := a.prepareEndpoint(params)
	if err != nil {
		return nil, err
	}
	body, err := prepareBody(request, imp)
	if err != nil {
		return nil, err
	}
	headers, err := prepareHeaders(request)
	if err != nil {
		return nil, err
	}

	impIds, err := prepareImpIds(request)
	if err != nil {
		return nil, err
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  impIds,
	}, nil
}

func prepareExtParams(imp openrtb2.Imp) (*openrtb_ext.ImpExtClydo, error) {
	var clydoImpExt openrtb_ext.ImpExtClydo
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "missing ext.bidder",
		}
	}
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &clydoImpExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "invalid ext.bidder",
		}
	}
	return &clydoImpExt, nil
}

func (a *adapter) prepareEndpoint(params *openrtb_ext.ImpExtClydo) (string, error) {
	partnerId := params.PartnerId
	if partnerId == "" {
		return "", &errortypes.BadInput{
			Message: "invalid partnerId",
		}
	}

	region := params.Region
	if region == "" {
		region = "us"
	}

	endpointParams := macros.EndpointTemplateParams{
		PartnerId: partnerId,
		Region:    region,
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func prepareBody(request *openrtb2.BidRequest, imp openrtb2.Imp) ([]byte, error) {
	reqCopy := *request
	reqCopy.Imp = []openrtb2.Imp{imp}

	reqCopy.Imp[0].Ext = imp.Ext

	body, err := jsonutil.Marshal(&reqCopy)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func prepareHeaders(request *openrtb2.BidRequest) (http.Header, error) {
	allHeaders := map[string]string{
		"X-OpenRTB-Version": "2.5",
		"Accept":            "application/json",
		"Content-Type":      "application/json; charset=utf-8",
	}

	allHeaders, err := appendDeviceHeaders(allHeaders, request)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	for k, v := range allHeaders {
		headers.Add(k, v)
	}

	return headers, nil
}

func appendDeviceHeaders(headers map[string]string, request *openrtb2.BidRequest) (map[string]string, error) {
	if request.Device == nil {
		return nil, &errortypes.BadInput{Message: "Failed to get device headers"}
	}

	if ipv6 := request.Device.IPv6; ipv6 != "" {
		headers["X-Forwarded-For"] = ipv6
	}
	if ip := request.Device.IP; ip != "" {
		headers["X-Forwarded-For"] = ip
	}
	if ua := request.Device.UA; ua != "" {
		headers["User-Agent"] = ua
	}

	return headers, nil
}

func prepareImpIds(request *openrtb2.BidRequest) ([]string, error) {
	impIds := openrtb_ext.GetImpIDs(request.Imp)
	if impIds == nil {
		return nil, &errortypes.BadInput{Message: "Failed to get imp ids"}
	}
	return impIds, nil
}

func checkResponseStatus(responseData *adapters.ResponseData) []error {
	switch responseData.StatusCode {
	case http.StatusNoContent:
		return []error{}
	case http.StatusBadRequest:
		return []error{&errortypes.BadInput{
			Message: "Bad request. Run with request.debug = 1 for more info.",
		}}
	case http.StatusOK:
		return nil
	default:
		return []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}}
	}
}

func prepareBidResponse(body []byte) (openrtb2.BidResponse, error) {
	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(body, &response); err != nil {
		return response, err
	}
	return response, nil
}

func prepareSeatBids(seatBids []openrtb2.SeatBid, bidTypeMap map[string]openrtb_ext.BidType) ([]*adapters.TypedBid, []error) {
	var typedBids []*adapters.TypedBid
	var errors []error

	if seatBids == nil {
		return typedBids, nil
	}

	for _, seatBid := range seatBids {
		if seatBid.Bid == nil {
			continue
		}
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType := getMediaTypeForBid(bid, bidTypeMap)
			typedBids = append(typedBids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return typedBids, errors
}

func buildBidTypeMap(imps []openrtb2.Imp) (map[string]openrtb_ext.BidType, error) {
	bidTypeMap := make(map[string]openrtb_ext.BidType, len(imps))
	for _, imp := range imps {
		switch {
		case imp.Video != nil:
			bidTypeMap[imp.ID] = openrtb_ext.BidTypeVideo
		case imp.Native != nil:
			bidTypeMap[imp.ID] = openrtb_ext.BidTypeNative
		case imp.Banner != nil:
			bidTypeMap[imp.ID] = openrtb_ext.BidTypeBanner
		default:
			return nil, &errortypes.BadInput{
				Message: "Failed to get media type",
			}
		}
	}
	return bidTypeMap, nil
}

func getMediaTypeForBid(bid *openrtb2.Bid, bidTypeMap map[string]openrtb_ext.BidType) openrtb_ext.BidType {
	if mediaType, ok := bidTypeMap[bid.ImpID]; ok {
		return mediaType
	}
	return openrtb_ext.BidTypeBanner
}
