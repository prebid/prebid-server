package goldbach

import (
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

type requestExtAdapter struct {
	Goldbach requestExtGoldbach `json:"goldbach"`

	*openrtb_ext.ExtRequest `json:"inline,omitempty"`
}

type requestExtGoldbach struct {
	PublisherID  string `json:"publisherId"`
	MockResponse *bool  `json:"mockResponse,omitempty"`
}

type impExtAdapter struct {
	Goldbach impExtGoldbachOutgoing `json:"goldbach"`
}

type impExtGoldbachOutgoing struct {
	Targetings map[string][]string `json:"targetings,omitempty"`
	SlotID     string              `json:"slotId"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var reqs []*adapters.RequestData
	var errs []error

	var requestExt requestExtAdapter
	if request.Ext != nil {
		if err := jsonutil.Unmarshal(request.Ext, &requestExt); err != nil {
			errs = append(errs, &errortypes.FailedToUnmarshal{Message: fmt.Errorf("unable to unmarshal request.ext: %w", err).Error()})
		}
	}

	// group impressions by publisher ID
	publisherImps := make(map[string][]openrtb2.Imp)
	for _, imp := range request.Imp {
		publisherID, impCopy, err := buildImp(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		publisherImps[publisherID] = append(publisherImps[publisherID], impCopy)
	}

	if len(publisherImps) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "no valid impression found"})
	}

	// create a separate request for each publisher
	for publisherID, imps := range publisherImps {
		requestPublisher, err := buildRequest(*request, publisherID, imps, &requestExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		resJSON, err := jsonutil.Marshal(&requestPublisher)
		if err != nil {
			errs = append(errs, &errortypes.FailedToMarshal{Message: fmt.Errorf("unable to marshal request: %w", err).Error()})
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		req := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    resJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestPublisher.Imp),
		}

		reqs = append(reqs, req)
	}

	return reqs, errs
}

func (a *adapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpRes.StatusCode != http.StatusCreated {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", httpRes.StatusCode),
		}}
	}

	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(httpRes.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Errorf("unable to unmarshal response: %w", err).Error(),
		}}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = resp.Cur

	var errs []error
	for _, sb := range resp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getBidMediaType(&sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}

	if len(bidderResponse.Bids) == 0 {
		errs = append(errs, &errortypes.BadServerResponse{
			Message: "no valid bids found in response",
		})
		return nil, errs
	}

	return bidderResponse, errs
}

func buildImp(imp openrtb2.Imp) (string, openrtb2.Imp, error) {
	impExt, err := extractImpExt(&imp)
	if err != nil {
		return "", openrtb2.Imp{}, err
	}

	targetings := make(map[string][]string)
	for key, value := range impExt.CustomTargeting {
		targetings[key] = value
	}

	imp.Ext, err = jsonutil.Marshal(&impExtAdapter{
		Goldbach: impExtGoldbachOutgoing{
			Targetings: targetings,
			SlotID:     impExt.SlotID,
		},
	})

	if err != nil {
		return "", openrtb2.Imp{}, &errortypes.FailedToMarshal{Message: fmt.Errorf("unable to marshal imp.ext: %w", err).Error()}
	}

	return impExt.PublisherID, imp, nil
}

func extractImpExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtGoldbach, error) {
	var extImpBidder adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &extImpBidder); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Errorf("unable to unmarshal imp.ext: %w", err).Error(),
		}
	}

	var goldbachExt openrtb_ext.ImpExtGoldbach
	if err := jsonutil.Unmarshal(extImpBidder.Bidder, &goldbachExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Errorf("unable to unmarshal imp.ext.bidder: %w", err).Error(),
		}
	}

	if len(goldbachExt.PublisherID) == 0 || len(goldbachExt.SlotID) == 0 {
		return nil, &errortypes.BadInput{
			Message: "publisherId and slotId are required",
		}
	}
	return &goldbachExt, nil
}

func buildRequest(request openrtb2.BidRequest, publisherID string, imps []openrtb2.Imp, requestExt *requestExtAdapter) (*openrtb2.BidRequest, error) {
	request.Imp = imps
	request.ID = fmt.Sprintf("%s_%s", request.ID, publisherID)

	// Set the publisher ID in the request.ext
	requestPublisherExt, err := jsonutil.Marshal(&requestExtAdapter{
		Goldbach: requestExtGoldbach{
			PublisherID:  publisherID,
			MockResponse: requestExt.Goldbach.MockResponse,
		},
		ExtRequest: requestExt.ExtRequest,
	})
	if err != nil {
		return nil, &errortypes.FailedToMarshal{Message: fmt.Errorf("unable to marshal request.ext: %w", err).Error()}
	}

	request.Ext = requestPublisherExt

	return &request, nil
}

func getBidMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	var extBid openrtb_ext.ExtBid
	if err := jsonutil.Unmarshal(bid.Ext, &extBid); err != nil {
		return "", &errortypes.FailedToUnmarshal{Message: fmt.Errorf("unable to unmarshal ext for bid: %w", err).Error()}
	}

	if extBid.Prebid == nil || len(extBid.Prebid.Type) == 0 {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("no media type for bid %v", bid.ID)}
	}

	return extBid.Prebid.Type, nil
}
