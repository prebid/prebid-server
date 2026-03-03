package adhese

import (
	"fmt"
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

type extTargetsAdhese map[string][]string
type extImpAdheseWrapper map[string]extTargetsAdhese
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

	imp := &request.Imp[0]

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Error unmarshalling imp.ext: %v", err)}}
	}

	var params openrtb_ext.ExtImpAdhese
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Error unmarshalling bidder ext: %v", err)}}
	}

	// define a map of targets[] and pre-fill it with the slot
	targets := extTargetsAdhese{
		"SL": []string{makeSlot(params)},
	}
	// add any additional targets to the map from the params
	for k, v := range params.Targets {
		targets[k] = v
	}

	// marshal the ext.adhese.bidder object into the ext field
	modifiedExt, err := jsonutil.Marshal(extImpAdheseWrapper{
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
	requestJSON, err := jsonutil.Marshal(modifiedRequest)

	if err != nil {
		return nil, []error{err}
	}
	var firstImIpd = openrtb_ext.GetImpIDs(request.Imp)[0]
	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    endpoint,
		Body:   requestJSON,
		ImpIDs: []string{firstImIpd},
	}

	return []*adapters.RequestData{requestData}, nil
}

func inferBidTypeFromImp(i openrtb2.Imp) (openrtb_ext.BidType, []error) {
	var mediaTypes []openrtb_ext.BidType

	if i.Banner != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeBanner)
	}
	if i.Video != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeVideo)
	}
	if i.Native != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeNative)
	}
	if i.Audio != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeAudio)
	}

	if len(mediaTypes) == 1 {
		// If there's only one media type, return it.
		return mediaTypes[0], nil
	} else if len(mediaTypes) > 1 {
		// Multi-format case: Log or return an error indicating multiple media types.
		return "", []error{&errortypes.BadServerResponse{Message: "Multiple media types detected, cannot infer"}}
	}

	// If no media type was found
	return "", []error{&errortypes.BadServerResponse{Message: "Could not infer bid type from imp"}}
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	err := adapters.CheckResponseStatusCodeForErrors(responseData)
	if err != nil {
		return nil, []error{err}
	}

	noContent := adapters.IsResponseStatusCodeNoContent(responseData)
	if noContent {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "Empty body"}}
	}

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

	// create a new bidResponse with a capacity of 1 because we only expect 1 bid
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(response.SeatBid[0].Bid))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	bid := bids[0]

	var wrappedBidExt wrappedBidExt
	if err := jsonutil.Unmarshal(bid.Ext, &wrappedBidExt); err != nil {
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

	marshalledBidExt, err := jsonutil.Marshal(wrappedBidExt["adhese"])
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
