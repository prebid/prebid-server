package adelerate

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

const defaultCurrency = "USD"

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	validImps := make([]openrtb2.Imp, 0, len(request.Imp))

	for i := range request.Imp {
		currImp := request.Imp[i]

		var extImpBidder adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(currImp.Ext, &extImpBidder); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext for impression %s: %s", currImp.ID, err.Error()),
			})
			continue
		}

		var bidderExt openrtb_ext.ImpExtAdelerate
		if err := jsonutil.Unmarshal(extImpBidder.Bidder, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext for impression %s: %s", currImp.ID, err.Error()),
			})
			continue
		}

		if bidderExt.Floor > 0 && currImp.BidFloor == 0 {
			floorCurrency := bidderExt.FloorCurrency
			if floorCurrency == "" {
				floorCurrency = defaultCurrency
			}

			floor := bidderExt.Floor
			if floorCurrency != defaultCurrency {
				convertedFloor, err := reqInfo.ConvertCurrency(floor, floorCurrency, defaultCurrency)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				floor = convertedFloor
			}

			currImp.BidFloor = floor
			currImp.BidFloorCur = defaultCurrency
		}

		currImp.Secure = openrtb2.Int8Ptr(1)

		validImps = append(validImps, currImp)
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	reqCopy := *request
	reqCopy.Imp = validImps

	adapterReq, err := a.buildRequest(&reqCopy)
	if err != nil {
		return nil, append(errs, err)
	}

	return []*adapters.RequestData{adapterReq}, errs
}

func (a *adapter) buildRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	reqJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
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
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
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

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s missing required mtype", bid.ID),
		}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}
}
