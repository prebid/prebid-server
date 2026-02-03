package revx

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// RevXAdapter struct
type adapter struct {
	endPoint string
}

// Builder builds a new instance of the RevX adapter.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endPoint: config.Endpoint, // Default endpoint
	}, nil
}

// MakeRequests handles the OpenRTB bid request and returns адаптер.RequestData
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	// Build headers
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	// Marshal request
	reqJson, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Failed to marshal request: %s", err)}} // skip append
	}

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endPoint,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, errors
}

// MakeBids handles the OpenRTB bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if externalRequest == nil {
		return nil, nil
	}

	// Handle specific status codes first
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	// For all other status codes
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	// Proceed with normal parsing
	var serverBidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &serverBidResponse); err != nil {
		return nil, []error{err}
	}

	var typedBids []*adapters.TypedBid
	var errs []error
	for _, sb := range serverBidResponse.SeatBid {
		for i := range sb.Bid {
			mediaType, err := getMediaTypeForImp(sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			typedBids = append(typedBids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}

	if len(typedBids) == 0 {
		return nil, errs
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(typedBids))
	bidResponse.Bids = typedBids
	bidResponse.Currency = serverBidResponse.Cur

	return bidResponse, errs
}

func getMediaTypeForImp(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}
}
