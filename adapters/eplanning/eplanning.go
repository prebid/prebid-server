package eplanning

import (
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"strconv"
)

const DEFAULT_EXCHANGE_ID = "5a1ad71d2d53a0f5"

type EPlanningAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

func (adapter *EPlanningAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errors := make([]error, 0, len(request.Imp))
	totalImps := len(request.Imp)
	sourceMapper := make(map[string][]int)

	for i := 0; i < totalImps; i++ {
		source, err := verifyImp(&request.Imp[i])
		if err != nil {
			errors = append(errors, err)
			continue
		}

		// Save valid imp
		if _, ok := sourceMapper[source]; !ok {
			sourceMapper[source] = make([]int, 0, totalImps-i)
		}

		sourceMapper[source] = append(sourceMapper[source], i)
	}

	totalRequests := len(sourceMapper)

	if totalRequests == 0 {
		return nil, errors
	}

	requests := make([]*adapters.RequestData, 0, totalRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(request.Device.DNT)))
	}

	imps := make([]openrtb.Imp, len(request.Imp))
	copy(imps, request.Imp)

	for source, impIds := range sourceMapper {
		request.Imp = request.Imp[:0]

		for i := 0; i < len(impIds); i++ {
			request.Imp = append(request.Imp, imps[impIds[i]])
		}

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
		}

		requestData := adapters.RequestData{
			Method:  "POST",
			Uri:     adapter.URI + fmt.Sprintf("/%s", source),
			Body:    reqJSON,
			Headers: headers,
		}

		requests = append(requests, &requestData)
	}

	return requests, errors
}

func verifyImp(imp *openrtb.Imp) (string, error) {
	// We currently only support banner impressions
	if imp.Banner == nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("EPlanning only supports banner Imps. Ignoring Imp ID=%s", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpEPlanning{}
	err := json.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	if impExt.ExchangeID == "" {
		impExt.ExchangeID = DEFAULT_EXCHANGE_ID
	}

	return impExt.ExchangeID, nil
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func (adapter *EPlanningAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponse()

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidResponse, nil
}

func NewEPlanningBidder(client *http.Client, endpoint string) *EPlanningAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}

	return &EPlanningAdapter{
		http: adapter,
		URI:  endpoint,
	}
}
