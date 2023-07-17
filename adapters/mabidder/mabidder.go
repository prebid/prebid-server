package mabidder

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type maServerResponse struct {
	Responses       []maBidResponse
	PrivateIdStatus string `json:"-"`
}

type maBidResponse struct {
	RequestID         string  `json:"requestId"`
	Currency          string  `json:"currency"`
	Width             int32   `json:"width"`
	Height            int32   `json:"height"`
	PlacementId       string  `json:"creativeId"`
	Deal              string  `json:"dealId,omitempty"`
	NetRevenue        bool    `json:"netRevenue"`
	TimeToLiveSeconds int32   `json:"ttl"`
	AdTag             string  `json:"ad"`
	MediaType         string  `json:"mediaType"`
	Meta              maMeta  `json:"meta"`
	CPM               float32 `json:"cpm"`
}

type maMeta struct {
	AdDomain []string `json:"advertiserDomains"`
}

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Mabidder adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	requestJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response maServerResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Responses[0].Currency
	for _, maBidResp := range response.Responses {
		b := &adapters.TypedBid{
			Bid: &openrtb2.Bid{
				ID:      maBidResp.RequestID,
				ImpID:   maBidResp.RequestID,
				Price:   float64(maBidResp.CPM),
				AdM:     maBidResp.AdTag,
				W:       int64(maBidResp.Width),
				H:       int64(maBidResp.Height),
				CrID:    maBidResp.PlacementId,
				DealID:  maBidResp.Deal,
				ADomain: maBidResp.Meta.AdDomain,
			},
			BidType: openrtb_ext.BidType(maBidResp.MediaType),
		}
		bidResponse.Bids = append(bidResponse.Bids, b)
	}
	return bidResponse, nil
}
