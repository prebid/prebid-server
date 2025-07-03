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
	var requests []*adapters.RequestData
	var errors []error

	// Unmarshal imp.ext
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		errors = append(errors, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext: %s", err)})

	}

	var revxExt openrtb_ext.ExtImpRevX
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &revxExt); err != nil {
		errors = append(errors, &errortypes.BadInput{Message: "bad revx bidder ext"})

	}

	if len(requests) == 0 && len(errors) > 0 {
		return nil, errors
	}

	// Build dynamic endpoint
	//var fendpoint = fmt.Sprintf(a.endPoint, strings.ToUpper(revxExt.PubName))
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	// Marshal the OpenRTB bid request into JSON
	reqJson, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Failed to marshal request: %s", err)}}
	}

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endPoint,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errors
}

// MakeBids handles the OpenRTB bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if externalRequest == nil {
		return nil, nil
	}

	// Check HTTP status before parsing response body
	if err := CheckResponseStatusCodeForErrors(response); err != nil {
		// Treat 204 and 400 as no-bid without logging error
		if response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusBadRequest {
			return nil, nil
		}
		return nil, []error{err}
	}

	var serverBidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &serverBidResponse); err != nil {
		return nil, []error{err}
	}

	var typedBids []*adapters.TypedBid
	for _, sb := range serverBidResponse.SeatBid {
		for i := range sb.Bid {
			mediaType, err := getMediaTypeForImp(sb.Bid[i])
			if err != nil {
				return nil, []error{err}
			}

			typedBids = append(typedBids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}

	if len(typedBids) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(typedBids))
	bidResponse.Bids = typedBids
	bidResponse.Currency = serverBidResponse.Cur

	return bidResponse, nil
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

// CheckResponseStatusCodeForErrors checks the HTTP response status code for errors.
func CheckResponseStatusCodeForErrors(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}

	if response.StatusCode != http.StatusOK {
		return &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}

	return nil
}
