package dianomi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint         string
	endpointTemplate *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint:         config.Endpoint,
		endpointTemplate: template,
	}
	return bidder, nil
}

type dianomiExtImpBidder struct {
	Bidder json.RawMessage `json:"bidder"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, numRequests)
	errs := make([]error, 0, len(request.Imp))
	var err error

	requestImpCopy := request.Imp
	for _, imp := range requestImpCopy {

		var bidderExt dianomiExtImpBidder
		if err = json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var dianomiExt openrtb_ext.ImpExtDianomi
		if err = json.Unmarshal(bidderExt.Bidder, &dianomiExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		m := macros.EndpointTemplateParams{
			AdUnit: strconv.Itoa(dianomiExt.SmartadID),
		}
		endpoint, err := macros.ResolveMacros(a.endpointTemplate, m)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		request := &adapters.RequestData{
			Method: "GET",
			Uri:    endpoint,
		}

		requestData = append(requestData, request)
	}

	return requestData, errs
}

type dianomiResponse struct {
	BidAmount  string `json:"bid_amount"`
	BidCurency string `json:"bid_currency"`
	WinURL     string `json:"win_url"`
	Content    string `json:"content"`
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var response dianomiResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.BidCurency

	amount, err := strconv.ParseFloat(response.BidAmount, 64)
	if err != nil {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Can't parse bid amount: %s", response.BidAmount),
			},
		}
	}
	for _, imp := range request.Imp {
		b := &adapters.TypedBid{
			Bid: &openrtb2.Bid{
				ID:    "1234", // bid id
				CrID:  "1234", // creative id
				ImpID: imp.ID,
				Price: amount,
				AdM:   response.Content,
			},
			BidType: openrtb_ext.BidTypeBanner,
		}
		bidResponse.Bids = append(bidResponse.Bids, b)
	}

	return bidResponse, nil
}
