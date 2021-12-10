package dianomi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	numRequests := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, numRequests)
	errs := make([]error, 0, len(request.Imp))
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	rq := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Headers: headers,
		Body:    reqJSON,
	}

	requestData = append(requestData, rq)

	return requestData, errs
}

type dianomiResponse struct {
	BidAmount  string `json:"bid_amount"`
	BidCurency string `json:"bid_currency"`
	WinURL     string `json:"win_url"`
	Content    string `json:"content"`
	CrID       string `json:"crid"`
	BidID      string `json:"bid_id"`
	Width      int64  `json:"width"`
	Height     int64  `json:"height"`
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
				ID:    response.BidID, // bid id
				CrID:  response.CrID,  // creative id
				ImpID: imp.ID,
				Price: amount,
				AdM:   response.Content,
				NURL:  response.WinURL,
				W:     response.Width,
				H:     response.Height,
			},
			BidType: openrtb_ext.BidTypeBanner,
		}
		bidResponse.Bids = append(bidResponse.Bids, b)
	}

	return bidResponse, nil
}
