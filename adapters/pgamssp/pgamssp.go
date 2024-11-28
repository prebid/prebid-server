package pgamssp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	PGAMSspBidderExt reqBodyExtBidder `json:"bidder"`
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

		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
			// Convert to US dollars
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err == nil {
				imp.BidFloorCur = "USD"
				imp.BidFloor = convertedValue
			}
		}

		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		var pgamExt openrtb_ext.ImpExtPgamSsp

		if err = jsonutil.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}
		if err = jsonutil.Unmarshal(bidderExt.Bidder, &pgamExt); err != nil {
			return nil, []error{err}
		}

		temp := reqBodyExt{PGAMSspBidderExt: reqBodyExtBidder{}}

		if pgamExt.PlacementID != "" {
			temp.PGAMSspBidderExt.PlacementID = pgamExt.PlacementID
			temp.PGAMSspBidderExt.Type = "publisher"
		} else if pgamExt.EndpointID != "" {
			temp.PGAMSspBidderExt.EndpointID = pgamExt.EndpointID
			temp.PGAMSspBidderExt.Type = "network"
		}

		finalyImpExt, err := json.Marshal(temp)
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
	}, err
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
		return "", fmt.Errorf("Unable to fetch mediaType in multi-format: %s", bid.ImpID)
	}
}
