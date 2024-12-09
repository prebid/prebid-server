package seedingAlliance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, extraRequestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var accountId string
	var err error

	for i := range request.Imp {
		if accountId, err = getExtInfo(&request.Imp[i]); err != nil {
			return nil, []error{err}
		}
	}

	if !curExists(request.Cur, "EUR") {
		request.Cur = append(request.Cur, "EUR")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	url, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{AccountID: accountId})
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    url,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
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

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			resolvePriceMacro(&seatBid.Bid[i])

			bidType, err := getMediaTypeForBid(seatBid.Bid[i].Ext)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func resolvePriceMacro(bid *openrtb2.Bid) {
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
}

func getMediaTypeForBid(ext json.RawMessage) (openrtb_ext.BidType, error) {
	var bidExt openrtb_ext.ExtBid

	if err := jsonutil.Unmarshal(ext, &bidExt); err != nil {
		return "", fmt.Errorf("could not unmarshal openrtb_ext.ExtBid: %w", err)
	}

	if bidExt.Prebid == nil {
		return "", fmt.Errorf("bid.Ext.Prebid is empty")
	}

	return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
}

func curExists(allowedCurrencies []string, newCurrency string) bool {
	for i := range allowedCurrencies {
		if allowedCurrencies[i] == newCurrency {
			return true
		}
	}
	return false
}

func getExtInfo(imp *openrtb2.Imp) (string, error) {
	var ext adapters.ExtImpBidder
	var extSA openrtb_ext.ImpExtSeedingAlliance

	accountId := "pbs"

	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return "", fmt.Errorf("could not unmarshal adapters.ExtImpBidder: %w", err)
	}

	if err := jsonutil.Unmarshal(ext.Bidder, &extSA); err != nil {
		return "", fmt.Errorf("could not unmarshal openrtb_ext.ImpExtSeedingAlliance: %w", err)
	}

	imp.TagID = extSA.AdUnitID

	if extSA.SeatID != "" {
		accountId = extSA.SeatID
	}

	if extSA.AccountID != "" {
		accountId = extSA.AccountID
	}

	return accountId, nil
}
