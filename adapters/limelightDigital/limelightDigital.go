package limelightDigital

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
		limelightDigitalExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		url, err := a.buildEndpointURL(limelightDigitalExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {

			// Convert to US dollars
			convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				errors = append(errors, err)
				continue
			}

			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		requestCopy.ID = request.ID + "-" + imp.ID
		requestCopy.Imp = []openrtb2.Imp{imp}
		requestCopy.Ext = nil

		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, err)
		} else {
			requestData := &adapters.RequestData{
				Method: "POST",
				Uri:    url,
				Body:   requestJSON,
				ImpIDs: openrtb_ext.GetImpIDs(requestCopy.Imp),
			}
			requests = append(requests, requestData)
		}
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid.ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &seatBid.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtLimelightDigital, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder is not provided",
		}
	}
	var limelightDigitalExt openrtb_ext.ImpExtLimelightDigital
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &limelightDigitalExt); err != nil {
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
		PublisherID: params.PublisherID.String(),
	}
	return macros.ResolveMacros(a.endpointTemplate, endpointParams)
}

func getMediaTypeForBid(impId string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
			return "", fmt.Errorf("unknown media type of imp: %s", impId)
		}
	}
	return "", fmt.Errorf("bid contains unknown imp id: %s", impId)
}
