package mgid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type MgidAdapter struct {
	endpoint string
}

type ReqExt struct {
	PlacementId string `json:"placementId"`
	AccountId   string `json:"accountId"`
}

type RespBidExt struct {
	CreativeType openrtb_ext.BidType `json:"crtype"`
}

func (a *MgidAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (adapterRequests []*adapters.RequestData, errs []error) {

	adapterReq, errs := a.makeRequest(request)
	if adapterReq != nil && len(errs) == 0 {
		adapterRequests = append(adapterRequests, adapterReq)
	}

	return
}

func (a *MgidAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	path, err := preprocess(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	// Last Step
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint + path,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// Mutate the request to get it ready to send to yieldmo.
func preprocess(request *openrtb2.BidRequest) (path string, err error) {
	if request.TMax == 0 {
		request.TMax = 200
	}
	for i := 0; i < len(request.Imp); i++ {
		var imp = request.Imp[i]
		var bidderExt adapters.ExtImpBidder

		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return "", &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var mgidExt openrtb_ext.ExtImpMgid

		if err := json.Unmarshal(bidderExt.Bidder, &mgidExt); err != nil {
			return "", &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		if path == "" {
			path = mgidExt.AccountId
		}
		if mgidExt.PlacementId == "" {
			request.Imp[i].TagID = imp.ID
		} else {
			request.Imp[i].TagID = mgidExt.PlacementId + "/" + imp.ID
		}

		cur := ""
		if mgidExt.Currency != "" && mgidExt.Currency != "USD" {
			cur = mgidExt.Currency
		}
		if cur == "" && mgidExt.Cur != "" && mgidExt.Cur != "USD" {
			cur = mgidExt.Cur
		}
		bidfloor := mgidExt.BidFloor
		if bidfloor <= 0 {
			bidfloor = mgidExt.BidFloor2
		}
		if bidfloor > 0 {
			request.Imp[i].BidFloor = bidfloor
		}
		if cur != "" {
			request.Imp[i].BidFloorCur = cur
		}
	}
	if path == "" {
		return "", &errortypes.BadInput{
			Message: "accountId is not set",
		}
	}

	return
}

func (a *MgidAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	bidResponse.Currency = bidResp.Cur

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType := openrtb_ext.BidTypeBanner
			if len(sb.Bid[i].Ext) > 0 && bytes.Contains(sb.Bid[i].Ext, []byte("crtype")) {
				ext := RespBidExt{}
				if err := json.Unmarshal(sb.Bid[i].Ext, &ext); err == nil && len(ext.CreativeType) > 0 {
					bidType = ext.CreativeType
				}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, nil
}

// Builder builds a new instance of the Mgid adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &MgidAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
