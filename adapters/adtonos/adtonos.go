package adtonos

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
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}
	var impExt openrtb_ext.ImpExtAdTonos
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}

	endpoint, err := a.buildEndpointURL(&impExt)
	if err != nil {
		return nil, []error{err}
	}

	requestJson, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    requestJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ImpExtAdTonos) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: params.SupplierID}
	return macros.ResolveMacros(a.endpointTemplate, endpointParams)
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
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i], request.Imp)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errors
}

func getMediaTypeForBid(bid openrtb2.Bid, requestImps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	if bid.MType != 0 {
		// If present, use explicit markup type annotation from the bidder:
		switch bid.MType {
		case openrtb2.MarkupAudio:
			return openrtb_ext.BidTypeAudio, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupNative:
			return openrtb_ext.BidTypeNative, nil
		}
	}
	// As a fallback, guess markup type based on requested type - AdTonos is an audio company so we prioritize that.
	for _, requestImp := range requestImps {
		if requestImp.ID == bid.ImpID {
			if requestImp.Audio != nil {
				return openrtb_ext.BidTypeAudio, nil
			} else if requestImp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else {
				return "", &errortypes.BadInput{
					Message: fmt.Sprintf("Unsupported bidtype for bid: \"%s\"", bid.ImpID),
				}
			}
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression: \"%s\"", bid.ImpID),
	}
}
