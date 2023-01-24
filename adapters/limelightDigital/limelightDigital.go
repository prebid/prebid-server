package limelightDigital

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/macros"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpointTemplate *template.Template
}

// Builder builds a new instance of the Limelight Digital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	if config.Endpoint == "" {
		return nil, errors.New("Endpoint  adapter parameter is not provided")
	}
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpointTemplate: template,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error

	requestCopy := *request
	for _, imp := range request.Imp {
		limelightDigitalExt, err := a.getImpressionExt(&imp)
		if err != nil {
			return nil, append(errors, err)
		}

		url, err := a.buildEndpointURL(limelightDigitalExt)
		if err != nil {
			return nil, []error{err}
		}

		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {

			// Convert to US dollars
			convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				return nil, []error{err}
			}

			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		requestCopy.ID = request.ID + "-" + imp.ID
		requestCopy.Imp = []openrtb2.Imp{imp}

		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    url,
			Body:   requestJSON,
		}
		requests = append(requests, requestData)
	}
	return requests, errors
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

	if len(responseData.Body) == 0 {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForBid(bid.ImpID, request.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func (a *adapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtLimelightDigital, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder is not provided",
		}
	}
	var limelightDigitalExt openrtb_ext.ImpExtLimelightDigital
	if err := json.Unmarshal(bidderExt.Bidder, &limelightDigitalExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder is not provided",
		}
	}
	imp.Ext = nil
	return &limelightDigitalExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ImpExtLimelightDigital) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		Host:        params.Host,
		PublisherID: strconv.Itoa(params.PublisherID),
	}
	return macros.ResolveMacros(a.endpointTemplate, endpointParams)
}

func getMediaTypeForBid(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}
