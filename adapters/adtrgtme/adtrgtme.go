package adtrgtme

import (
	"encoding/json"
	"fmt"
	"net/http"

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

// Builder builds a new instance of the Adtrgtme adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (v *adapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	requestInfo *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	var requests []*adapters.RequestData
	var errors []error

	requestCopy := *openRTBRequest
	for _, imp := range openRTBRequest.Imp {
		siteID, err := getSiteIDFromImp(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		imp.Ext = nil
		requestCopy.Imp = []openrtb2.Imp{imp}

		requestBody, err := json.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     v.buildRequestURI(siteID),
			Body:    requestBody,
			Headers: makeRequestHeaders(openRTBRequest),
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		}

		requests = append(requests, requestData)
	}
	return requests, errors
}

func getSiteIDFromImp(imp *openrtb2.Imp) (uint64, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return 0, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var ext openrtb_ext.ExtImpAdtrgtme
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &ext); err != nil {
		return 0, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	return ext.SiteID, nil
}

func (v *adapter) buildRequestURI(siteID uint64) string {
	return fmt.Sprintf("%s?s=%d&prebid", v.endpoint, siteID)
}

func makeRequestHeaders(openRTBRequest *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if openRTBRequest.Device != nil {
		if len(openRTBRequest.Device.UA) > 0 {
			headers.Add("User-Agent", openRTBRequest.Device.UA)
		}

		if len(openRTBRequest.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", openRTBRequest.Device.IPv6)
		}

		if len(openRTBRequest.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", openRTBRequest.Device.IP)
		}
	}
	return headers
}

func (v *adapter) checkResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ]", response.StatusCode),
		}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode != http.StatusOK {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ]. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}

	return nil
}

func (v *adapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	*adapters.BidderResponse,
	[]error,
) {
	if bidderRawResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	httpStatusError := v.checkResponseStatusCodes(bidderRawResponse)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	if len(response.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(openRTBRequest.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(bid.ImpID, openRTBRequest.Imp)
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

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else {
				return "", &errortypes.BadInput{
					Message: fmt.Sprintf("Unsupported bidtype for bid: \"%s\"", impId),
				}
			}
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression: \"%s\"", impId),
	}
}
