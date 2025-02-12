package onetag

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
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
	pubID := ""
	for idx, imp := range request.Imp {
		onetagExt, err := getImpressionExt(imp)
		if err != nil {
			return nil, []error{err}
		}
		if onetagExt.PubId != "" {
			if pubID == "" {
				pubID = onetagExt.PubId
			} else if pubID != onetagExt.PubId {
				return nil, []error{&errortypes.BadInput{
					Message: "There must be only one publisher ID",
				}}
			}
		} else {
			return nil, []error{&errortypes.BadInput{
				Message: "The publisher ID must not be empty",
			}}
		}
		request.Imp[idx].Ext = onetagExt.Ext
	}

	url, err := a.buildEndpointURL(pubID)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    url,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func getImpressionExt(imp openrtb2.Imp) (*openrtb_ext.ExtImpOnetag, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided or can't be unmarshalled",
		}
	}

	var onetagExt openrtb_ext.ExtImpOnetag
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &onetagExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}

	return &onetagExt, nil
}

func (a *adapter) buildEndpointURL(pubID string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: pubID}
	return macros.ResolveMacros(a.endpointTemplate, endpointParams)
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
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
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			bidMediaType, err := getMediaTypeForBid(request.Imp, bid)
			if err != nil {
				return nil, []error{err}
			}
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidMediaType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(impressions []openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	for _, impression := range impressions {
		if impression.ID == bid.ImpID {
			if impression.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if impression.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			if impression.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("The impression with ID %s is not present into the request", bid.ImpID),
	}
}
