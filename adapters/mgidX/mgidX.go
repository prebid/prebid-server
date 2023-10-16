package mgidX

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

type reqBodyExt struct {
	MgidXBidderExt reqBodyExtBidder `json:"bidder"`
}

type reqBodyExtBidder struct {
	Type        string `json:"type"`
	PlacementID string `json:"placementId,omitempty"`
	EndpointID  string `json:"endpointId,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
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
		var mgidXExt openrtb_ext.ImpExtMgidX

		if err = json.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}
		if err = json.Unmarshal(bidderExt.Bidder, &mgidXExt); err != nil {
			return nil, []error{err}
		}

		impExt := reqBodyExt{MgidXBidderExt: reqBodyExtBidder{}}

		if mgidXExt.PlacementID != "" {
			impExt.MgidXBidderExt.PlacementID = mgidXExt.PlacementID
			impExt.MgidXBidderExt.Type = "publisher"
		} else if mgidXExt.EndpointID != "" {
			impExt.MgidXBidderExt.EndpointID = mgidXExt.EndpointID
			impExt.MgidXBidderExt.Type = "network"
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

func (a *adapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtMgidX, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, fmt.Errorf("Bidder extension not provided or can't be unmarshalled")
	}

	var mgidXExt openrtb_ext.ImpExtMgidX
	if err := json.Unmarshal(bidderExt.Bidder, &mgidXExt); err != nil {
		return nil, fmt.Errorf("Error while unmarshaling bidder extension")
	}

	return &mgidXExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ImpExtMgidX) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: params.Host}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	var mgidXExt *openrtb_ext.ImpExtMgidX
	mgidXExt, err := a.getImpressionExt(&(request.Imp[0]))
	if err != nil {
		return nil, err
	}

	url, err := a.buildEndpointURL(mgidXExt)
	if err != nil {
		return nil, err
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: headers,
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
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
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
	err := json.Unmarshal(bid.Ext, &extBid)
	if err != nil {
		return "", fmt.Errorf("unable to deserialize imp %v bid.ext", bid.ImpID)
	}

	if extBid.Prebid == nil {
		return "", fmt.Errorf("imp %v with unknown media type", bid.ImpID)
	}

	return extBid.Prebid.Type, nil
}
