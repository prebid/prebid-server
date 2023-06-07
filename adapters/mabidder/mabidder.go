package mabidder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	validImps, errs := getValidImpressions(request, requestInfo)
	if len(validImps) == 0 {
		return nil, errs
	}
	request.Imp = validImps

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

	var response maServerResponse
	//var response openrtb2.BidResponse

	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	//bidResponse.Currency = response.Currency
	for _, maBidResp := range response.Responses {
		b := &adapters.TypedBid{
			Bid: &openrtb2.Bid{
				ID:     maBidResp.RequestID,
				ImpID:  maBidResp.RequestID,
				Price:  float64(maBidResp.CPM),
				AdM:    maBidResp.AdTag,
				W:      int64(maBidResp.Width),
				H:      int64(maBidResp.Height),
				CrID:   maBidResp.PlacementId,
				DealID: maBidResp.Deal,
			},
			//BidType: getMediaTypeForBid(bid),
			BidType: openrtb_ext.BidTypeBanner,
		}
		bidResponse.Bids = append(bidResponse.Bids, b)
	}
	return bidResponse, nil
}

// validate imps and check for bid floor currency. Convert to USD if necessary
func getValidImpressions(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]openrtb2.Imp, []error) {
	var errs []error
	var validImps []openrtb2.Imp

	for _, imp := range request.Imp {
		if err := convertBidFloorCurrency(&imp, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := processExtensions(&imp); err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
	}
	return validImps, errs
}

// convert to USD
func convertBidFloorCurrency(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor > 0 && strings.ToUpper(imp.BidFloorCur) != "USD" && imp.BidFloorCur != "" {
		if convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD"); err != nil {
			return err
		} else {
			imp.BidFloor = convertedValue
		}
	}
	imp.BidFloorCur = "USD"
	return nil
}

func processExtensions(imp *openrtb2.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var mabidderExt openrtb_ext.ImpExtMabidder
	if err := json.Unmarshal(bidderExt.Bidder, &mabidderExt); err != nil {
		return &errortypes.BadInput{
			Message: "Wrong mabidder bidder ext: " + err.Error(),
		}
	}
	return nil
}
