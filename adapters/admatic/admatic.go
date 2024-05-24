package admatic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint template: %v", err)
	}

	bidder := &adapter{
		endpoint: endpointTemplate,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	host := ""

	for _, imp := range request.Imp {
		admaticExt, err := getImpressionExt(imp)
		if err != nil {
			return nil, []error{err}
		}

		if host == "" {
			host = admaticExt.Host
		} else if host != admaticExt.Host {
			return nil, []error{&errortypes.BadInput{
				Message: "There must be only one Host",
			}}
		}
	}

	resolvedUrl, err := a.resolveUrl(host)

	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    resolvedUrl,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func getImpressionExt(imp openrtb2.Imp) (*openrtb_ext.ImpExtAdmatic, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided or can't be unmarshalled",
		}
	}

	var admaticExt openrtb_ext.ImpExtAdmatic
	if err := json.Unmarshal(bidderExt.Bidder, &admaticExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}

	return &admaticExt, nil
}

// "Un-templates" the endpoint by replacing macroses and adding the required query parameters
func (a *adapter) resolveUrl(host string) (string, error) {
	params := macros.EndpointTemplateParams{Host: host}

	endpointStr, err := macros.ResolveMacros(a.endpoint, params)
	if err != nil {
		return "", err
	}

	parsedUrl, err := url.Parse(endpointStr)
	if err != nil {
		return "", err
	}

	return parsedUrl.String(), nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) != 0 {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {

			bidMediaType, err := getMediaTypeForBid(seatBid.Bid[i].ImpID, request.Imp)
			if err != nil {
				return nil, []error{err}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidMediaType,
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("The impression with ID %s is not present into the request", impID),
	}
}
