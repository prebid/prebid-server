package clickio

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No Imps in Bid Request"}}
	}

	requestCopy := *request
	requestCopy.Imp = append([]openrtb2.Imp(nil), request.Imp...)
	for i := range requestCopy.Imp {
		if err := updateImpExtWithParams(&requestCopy.Imp[i]); err != nil {
			return nil, []error{&errortypes.BadInput{Message: err.Error()}}
		}
	}

	body, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}}, nil
}

func updateImpExtWithParams(imp *openrtb2.Imp) error {
	if len(imp.Ext) == 0 {
		return nil
	}

	var impExt map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
		return fmt.Errorf("failed to parse imp.ext for imp %s: %w", imp.ID, err)
	}

	bidderParams, exists := impExt["bidder"]
	if !exists || len(bidderParams) == 0 {
		return nil
	}

	impExt["params"] = bidderParams

	rawExt, err := jsonutil.Marshal(impExt)
	if err != nil {
		return fmt.Errorf("failed to marshal imp.ext for imp %s: %w", imp.ID, err)
	}

	imp.Ext = rawExt
	return nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID != impID {
			continue
		}
		if imp.Banner != nil {
			return openrtb_ext.BidTypeBanner, nil
		}
		return "", &errortypes.BadInput{Message: fmt.Sprintf("Failed to resolve media type for impression \"%s\"", impID)}
	}
	return "", &errortypes.BadInput{Message: fmt.Sprintf("Failed to find impression \"%s\"", impID)}
}
