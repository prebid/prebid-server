package aso

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint template: %v", err)
	}

	bidder := &adapter{
		endpoint: endpointTemplate,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var requests []*adapters.RequestData
	var errors []error

	requestCopy := *request

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext, %s", err.Error()),
			})
			continue
		}

		var impExt openrtb_ext.ExtImpAso
		if err := json.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid bidderExt.Bidder, %s", err.Error()),
			})
			continue
		}

		requestCopy.Imp = []openrtb2.Imp{imp}
		endpoint, err := a.buildEndpointURL(&impExt)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		reqJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		requestData := &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    reqJSON,
			Headers: headers,
		}
		requests = append(requests, requestData)
	}
	return requests, errors
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			resolveMacros(&seatBid.Bid[i])

			bidType, err := getMediaType(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errors
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpAso) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ZoneID: strconv.Itoa(params.Zone)}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func getMediaType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to get type of bid \"%s\"", bid.ImpID),
	}
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid == nil {
		return
	}
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
	bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
}
