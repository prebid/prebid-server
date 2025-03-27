package emtv

import (
	"encoding/json"
	"errors"
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

type reqBodyExt struct {
	EmtvBidderExt reqBodyExtBidder `json:"bidder"`
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
	var err error
	var adapterRequests []*adapters.RequestData

	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		var emtvExt openrtb_ext.ImpExtEmtv

		if err = jsonutil.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}
		if err = jsonutil.Unmarshal(bidderExt.Bidder, &emtvExt); err != nil {
			return nil, []error{err}
		}

		impExt := reqBodyExt{EmtvBidderExt: reqBodyExtBidder{}}

		if emtvExt.PlacementID != "" {
			impExt.EmtvBidderExt.PlacementID = emtvExt.PlacementID
			impExt.EmtvBidderExt.Type = "publisher"
		} else if emtvExt.EndpointID != "" {
			impExt.EmtvBidderExt.EndpointID = emtvExt.EndpointID
			impExt.EmtvBidderExt.Type = "network"
		} else {
			continue
		}

		finalyImpExt, err := json.Marshal(impExt)
		if err != nil {
			return nil, []error{err}
		}

		reqCopy.Imp[0].Ext = finalyImpExt

		adapterReq, err := a.makeRequest(&reqCopy)
		if err != nil {
			return nil, []error{err}
		}

		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
	}

	if len(adapterRequests) == 0 {
		return nil, []error{errors.New("found no valid impressions")}
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
	bidResponse.Currency = response.Cur

	impsMappedByID := make(map[string]openrtb2.Imp, len(request.Imp))
	for i, imp := range request.Imp {
		impsMappedByID[request.Imp[i].ID] = imp
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i].ImpID, request.Imp, impsMappedByID)
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

func getMediaTypeForImp(impID string, imps []openrtb2.Imp, impMap map[string]openrtb2.Imp) (openrtb_ext.BidType, error) {
	if index, found := impMap[impID]; found {
		if index.Banner != nil {
			return openrtb_ext.BidTypeBanner, nil
		}
		if index.Video != nil {
			return openrtb_ext.BidTypeVideo, nil
		}
		if index.Native != nil {
			return openrtb_ext.BidTypeNative, nil
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", impID),
	}
}
