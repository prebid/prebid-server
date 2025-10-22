package beop

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

func Builder(
	bidderName openrtb_ext.BidderName,
	config config.Adapter,
	server config.Server) (
	adapters.Bidder, error,
) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) getRequestExtImpBeop(imp *openrtb2.Imp) (*openrtb_ext.ExtImpBeop, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var beopExt openrtb_ext.ExtImpBeop
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &beopExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	if beopExt.BeopPublisherID == "" && beopExt.BeopNetworkID == "" {
		return nil, &errortypes.BadInput{
			Message: "Missing pid or nid parameters",
		}
	}
	return &beopExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpBeop) (string, error) {
	url, err := url.Parse(a.endpoint)
	if err != nil {
		return "", &errortypes.Warning{
			Message: "Failed to parse endpoint",
		}
	}
	query := url.Query()
	if pid := params.BeopPublisherID; len(pid) != 0 {
		query.Set("pid", pid)
	}
	if nid := params.BeopNetworkID; len(nid) != 0 {
		query.Set("nid", nid)
	}
	url.RawQuery = query.Encode()
	return url.String(), nil
}

func (a *adapter) MakeRequests(
	request *openrtb2.BidRequest,
	requestInfo *adapters.ExtraRequestInfo) (
	[]*adapters.RequestData, []error,
) {
	var beopExt *openrtb_ext.ExtImpBeop
	var err error

	beopExt, err = a.getRequestExtImpBeop(&request.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	url, err := a.buildEndpointURL(beopExt)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	fmt.Println(requestData)

	return []*adapters.RequestData{requestData}, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}

func (a *adapter) MakeBids(
	request *openrtb2.BidRequest,
	requestData *adapters.RequestData,
	responseData *adapters.ResponseData) (
	*adapters.BidderResponse, []error,
) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Service Unavailable. Status Code: [ %d ] ", responseData.StatusCode),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var responseBody openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &responseBody); err != nil {
		return nil, []error{err}
	}

	if len(responseBody.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponseFinal := adapters.NewBidderResponseWithBidsCapacity(len(responseBody.SeatBid[0].Bid))
	seatBid := responseBody.SeatBid[0]
	fmt.Println(seatBid)
	var errors []error
	for _, bid := range seatBid.Bid {
		bidType, err := getMediaTypeForBid(bid)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		bidResponseFinal.Bids = append(bidResponseFinal.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}
	return bidResponseFinal, errors
}
