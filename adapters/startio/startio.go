package startio

import (
	"fmt"
	"net/http"
	"slices"

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

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error
	requestCopy := *request

	if err := validateRequest(requestCopy); err != nil {
		return nil, []error{err}
	}

	impressions := requestCopy.Imp

	for i := range impressions {
		requestCopy.Imp = []openrtb2.Imp{impressions[i]}

		requestBody, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, fmt.Errorf("imp[%d]: failed to marshal request: %w", i, err))
			continue
		}

		requestData := &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     adapter.endpoint,
			Body:    requestBody,
			Headers: buildRequestHeaders(),
			ImpIDs:  []string{impressions[i].ID},
		}

		requests = append(requests, requestData)
	}

	return requests, errors
}

func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{wrapReqError(fmt.Sprintf("failed to unmarshal response body: %v", err))}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.SeatBid[0].Bid))

	for i := range bidResponse.SeatBid {
		for j := range bidResponse.SeatBid[i].Bid {
			bid := &bidResponse.SeatBid[i].Bid[j]
			bidType, err := getMediaTypeForBid(*bid)

			if err != nil {
				errs = append(errs, err)
			} else {
				bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
					Bid:     bid,
					BidType: bidType,
				})
			}
		}
	}

	if len(bidResponse.SeatBid) == 0 || len(bidResponse.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	return bidderResponse, errs
}

func validateRequest(request openrtb2.BidRequest) error {
	if !isSupportedCurrency(request.Cur) {
		return wrapReqError("unsupported currency: only USD is accepted")
	}

	return nil
}

func buildRequestHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	return headers
}

func isSupportedCurrency(currencies []string) bool {
	return len(currencies) == 0 || slices.Contains(currencies, "USD")
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			switch bidExt.Prebid.Type {
			case "banner":
				return openrtb_ext.BidTypeBanner, nil
			case "video":
				return openrtb_ext.BidTypeVideo, nil
			case "native":
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse bid media type for impression %s.", bid.ImpID),
	}
}

func wrapReqError(errorStr string) *errortypes.BadInput {
	return &errortypes.BadInput{Message: errorStr}
}
