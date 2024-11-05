package bluesea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type blueseaBidExt struct {
	MediaType string `json:"mediatype"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {

	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
			Uri:     fmt.Sprintf("%s?%s", a.endpoint, queryString),
			Body:    reqJson,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		}
		requestDatas = append(requestDatas, requestData)
	}
	// to safe double check in case the requestDatas is empty and no error is raised.
	if len(requestDatas) == 0 && len(errs) == 0 {
		errs = append(errs, fmt.Errorf("Empty RequestData"))
	}
	return requestDatas, errs
}

func extraImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpBluesea, error) {
	var impExt adapters.ExtImpBidder
	var blueseaImpExt openrtb_ext.ExtImpBluesea

	err := jsonutil.Unmarshal(imp.Ext, &impExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in parsing imp.ext. err = %v, imp.ext = %v", err.Error(), string(imp.Ext)),
		}
	}

	err = jsonutil.Unmarshal(impExt.Bidder, &blueseaImpExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in parsing imp.ext.bidder. err = %v, bidder = %v", err.Error(), string(impExt.Bidder)),
		}
	}
	if len(blueseaImpExt.PubId) == 0 || len(blueseaImpExt.Token) == 0 {
		return nil, &errortypes.BadInput{
			Message: "Error in parsing imp.ext.bidder, empty pubid or token",
		}
	}
	return &blueseaImpExt, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var blueseaResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &blueseaResponse); err != nil {
		return nil, []error{fmt.Errorf("Error in parsing bidresponse body")}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	if blueseaResponse.Cur != "" {
		bidResponse.Currency = blueseaResponse.Cur
	}
	for _, seatBid := range blueseaResponse.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(&bid)

			if err != nil {
				errs = append(errs, err)
				continue
			}
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {

	var bidExt blueseaBidExt
	if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
		return "", fmt.Errorf("Error in parsing bid.ext")
	}

	switch bidExt.MediaType {
	case "banner":
		return openrtb_ext.BidTypeBanner, nil
	case "native":
		return openrtb_ext.BidTypeNative, nil
	case "video":
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("Unknown bid type, %v", bidExt.MediaType)
	}
}
