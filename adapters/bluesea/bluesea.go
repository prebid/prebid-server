package bluesea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type blueseaAdapter struct {
	Endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {

	bidder := &blueseaAdapter{
		Endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *blueseaAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	impCount := len(request.Imp)

	if impCount == 0 {
		err := &errortypes.BadInput{
			Message: "Empty Imp objects",
		}
		return nil, []error{err}
	}

	requestDatas := make([]*adapters.RequestData, 0, impCount)
	errs := make([]error, 0, impCount)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	for _, imp := range request.Imp {
		blueseaImpExt, err := extraImpExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		reqJson, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		queryParams := url.Values{}
		queryParams.Add("pubid", blueseaImpExt.PubId)
		queryParams.Add("token", blueseaImpExt.Token)
		queryString := queryParams.Encode()
		requestData := &adapters.RequestData{
			Method:  "POST",
			Uri:     fmt.Sprintf("%s?%s", a.Endpoint, queryString),
			Body:    reqJson,
			Headers: headers,
		}
		requestDatas = append(requestDatas, requestData)
	}
	return requestDatas, errs
}

func extraImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpBluesea, error) {
	var impExt adapters.ExtImpBidder
	var blueseaImpExt openrtb_ext.ExtImpBluesea

	err := json.Unmarshal(imp.Ext, &impExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in parsing imp.ext. err = %v, imp.ext = %v", err.Error(), string(imp.Ext)),
		}
	}

	err = json.Unmarshal(impExt.Bidder, &blueseaImpExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in parsing imp.ext.bidder. err = %v, bidder = %v", err.Error(), string(impExt.Bidder)),
		}
	}
	if len(blueseaImpExt.PubId) == 0 || len(blueseaImpExt.Token) == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in parsing imp.ext.bidder, empty pubId or token"),
		}
	}
	return &blueseaImpExt, nil
}

func (a *blueseaAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForBid(bid, internalRequest.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(bid openrtb2.Bid, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}
