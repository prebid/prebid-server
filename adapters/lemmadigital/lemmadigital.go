package lemmadigital

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Lemmadigital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{errors.New("Impression array should not be empty")}
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}

	var impExt openrtb_ext.ImpExtLemmaDigital
	if err := json.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}

	endpoint, err := a.buildEndpointURL(impExt)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidType := openrtb_ext.BidTypeBanner
	if nil != request.Imp[0].Video {
		bidType = openrtb_ext.BidTypeVideo
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) > 0 {
		bidResponse.Currency = response.Cur
	}
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
		break
	}

	return bidResponse, nil
}

func (a *adapter) buildEndpointURL(params openrtb_ext.ImpExtLemmaDigital) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: strconv.Itoa(params.PublisherId),
		AdUnit: strconv.Itoa(params.AdId)}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}
