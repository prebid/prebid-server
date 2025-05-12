package sspBC

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	adapterVersion = "5.8"
)

type RequestData struct {
	Request     *openrtb2.BidRequest `json:"bidRequest"`
	RequestInfo *RequestInfo         `json:"requestInfo"`
}

type RequestInfo struct {
	PbsEntryPoint metrics.RequestType
}

type adapter struct {
	endpoint string
}

// ---------------ADAPTER INTERFACE------------------
// Builder builds a new instance of the sspBC adapter
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	sspBcRequest := &RequestData{
		Request: request,
		RequestInfo: &RequestInfo{
			PbsEntryPoint: requestInfo.PbsEntryPoint,
		},
	}

	requestJSON, err := json.Marshal(sspBcRequest)
	if err != nil {
		return nil, []error{err}
	}

	requestURL, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, []error{err}
	}

	// add query parameters to request
	queryParams := requestURL.Query()
	queryParams.Add("bdver", adapterVersion)
	requestURL.RawQuery = queryParams.Encode()

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    requestURL.String(),
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, externalResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if externalResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if externalResponse.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", externalResponse.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(externalResponse.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	bidResponse.Currency = response.Cur

	var errors []error
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errors
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType: %d.", bid.MType),
		}
	}
}
