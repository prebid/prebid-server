package mabidder

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type serverResponse struct {
	Responses       []bidResponse
	PrivateIdStatus string `json:"-"`
}

type bidResponse struct {
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
	Meta              meta    `json:"meta"`
	CPM               float32 `json:"cpm"`
}

type meta struct {
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
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response serverResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
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
		if maBidResp.Currency != "" {
			bidResponse.Currency = maBidResp.Currency
		}
	}
	return bidResponse, nil
}
