package adhese

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpointTemplate *template.Template
}

type ExtTargetsAdhese map[string][]string
type ExtImpAdheseWrapper map[string]ExtTargetsAdhese
type wrappedBidExt map[string]map[string]string

func makeSlot(params openrtb_ext.ExtImpAdhese) string {
	return fmt.Sprintf("%s-%s", params.Location, params.Format)
}

func (a *adapter) MakeRequests(
	request *openrtb2.BidRequest,
	requestInfo *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}
	imp := &request.Imp[0]

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Error unmarshalling imp.ext: %v", err)}}
	}

	var params openrtb_ext.ExtImpAdhese
	if err := json.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Error unmarshalling bidder ext: %v", err)}}
	}

	if params.Account == "" || params.Location == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "Missing required params: location, account",
		}}
	}

	// define a map of targets[] and pre-fill it with the slot
	targets := ExtTargetsAdhese{
		"SL": []string{makeSlot(params)},
	}
	// add any additional targets to the map from the params
	for k, v := range params.Targets {
		targets[k] = v
	}

	// marshal the ext.adhese.bidder object into the ext field
	modifiedExt, err := json.Marshal(ExtImpAdheseWrapper{
		"adhese": targets,
	})
	if err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Error marshalling modified ext: %v", err)}}
	}

	// copy the request and dereference any pointers* and override the ext of the
	modifiedRequest := *request
	modifiedRequest.Imp[0].Ext = modifiedExt

	// create a map of macros to resolve
	endpointParams := macros.EndpointTemplateParams{AccountID: params.Account}

	// resolve the macros in the endpoint template
	endpoint, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("Error resolving macros: %v", err)}}
	}

	// marshal the request body
	requestJSON, err := json.Marshal(modifiedRequest)

	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func inferBidTypeFromImp(i openrtb2.Imp) (openrtb_ext.BidType, []error) {
	if i.Banner != nil {
		return openrtb_ext.BidTypeBanner, nil
	}
	if i.Video != nil {
		return openrtb_ext.BidTypeVideo, nil
	}
	if i.Native != nil {
		return openrtb_ext.BidTypeNative, nil
	}
	if i.Audio != nil {
		return openrtb_ext.BidTypeAudio, nil
	}

	return "", []error{&errortypes.BadServerResponse{Message: "Could not infer bid type from imp"}}
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

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "No impression in the bid request",
		}}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}
	// create a new bidResponse with a capacity of 1 because we only expect 1 bid
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = response.Cur

	if (len(response.SeatBid)) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid",
		}}
	}

	bids := response.SeatBid[0].Bid
	if len(bids) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid.Bid",
		}}
	}
	bid := bids[0]

	var wrappedBidExt wrappedBidExt
	if err := json.Unmarshal(bid.Ext, &wrappedBidExt); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("BidExt parsing error. %s", err.Error()),
		}}
	}

	bidType, bidTypeErr := inferBidTypeFromImp(request.Imp[0])
	if bidTypeErr != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("BidType error: %s", bidTypeErr),
		}}
	}

	marshalledBidExt, err := json.Marshal(wrappedBidExt["adhese"])
	if err != nil {
		return nil, []error{err}
	}
	bid.Ext = marshalledBidExt

	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     &bid,
		BidType: bidType,
	})

	return bidResponse, nil
}

func Builder(name openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	templ, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpointTemplate: templ,
	}

	return bidder, nil
}
