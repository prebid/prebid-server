package loyal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type reqBodyExt struct {
	LoyalBidderExt reqBodyExtBidder `json:"bidder"`
}

type reqBodyExtBidder struct {
	Type        string `json:"type"`
	PlacementID string `json:"placementId,omitempty"`
	EndpointID  string `json:"endpointId,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		var loyalExt openrtb_ext.ImpExtLoyal

		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &loyalExt); err != nil {
			errs = append(errs, err)
			continue
		}

		impExt := reqBodyExt{LoyalBidderExt: reqBodyExtBidder{}}

		if loyalExt.PlacementID != "" {
			impExt.LoyalBidderExt.PlacementID = loyalExt.PlacementID
			impExt.LoyalBidderExt.Type = "publisher"
		} else if loyalExt.EndpointID != "" {
			impExt.LoyalBidderExt.EndpointID = loyalExt.EndpointID
			impExt.LoyalBidderExt.Type = "network"
		}

		finalyImpExt, err := json.Marshal(impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqCopy.Imp[0].Ext = finalyImpExt

		adapterReq, err := a.makeRequest(&reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
	}

	if len(adapterRequests) == 0 {
		errs = append(errs, errors.New("found no valid impressions"))
		return nil, errs
	}

	return adapterRequests, nil
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) != 0 {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getBidMediaType(&seatBid.Bid[i])
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

func getBidMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	var extBid openrtb_ext.ExtBid
	err := jsonutil.Unmarshal(bid.Ext, &extBid)
	if err != nil {
		return "", fmt.Errorf("unable to deserialize imp %v bid.ext, error: %v", bid.ImpID, err)
	}

	if extBid.Prebid == nil {
		return "", fmt.Errorf("imp %v with unknown media type", bid.ImpID)
	}

	switch extBid.Prebid.Type {
	case openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative:
		return extBid.Prebid.Type, nil
	default:
		return "", fmt.Errorf("invalid BidType: %s", extBid.Prebid.Type)
	}
}
