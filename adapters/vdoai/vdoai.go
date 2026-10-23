package vdoai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpointTemplate *template.Template
}

// Builder builds a new instance of the vdoai adapter for the given bidder with the given config.
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
	var errors []error

	// Group imps by endpoint URL
	impsByEndpoint := make(map[string][]openrtb2.Imp)

	for i := range request.Imp {
		imp := request.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
			})
			continue
		}

		var impExt openrtb_ext.ImpExtVdoai
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
			})
			continue
		}

		// Inject bidfloor from bidder ext into the imp if not already set
		if impExt.BidFloor > 0 && imp.BidFloor == 0 {
			imp.BidFloor = impExt.BidFloor
		}

		endpointParams := macros.EndpointTemplateParams{
			Host:        impExt.Host,
			PublisherID: impExt.PublisherId,
		}

		endpointURL, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		impsByEndpoint[endpointURL] = append(impsByEndpoint[endpointURL], imp)
	}

	if len(impsByEndpoint) == 0 {
		return nil, errors
	}

	var requests []*adapters.RequestData
	for endpoint, imps := range impsByEndpoint {
		requestCopy := *request
		requestCopy.Imp = imps

		requestJSON, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json")

		requestData := &adapters.RequestData{
			Method:  "POST",
			Uri:     endpoint,
			Body:    requestJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(imps),
		}

		requests = append(requests, requestData)
	}

	return requests, errors
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("error while decoding response, err: %s", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid, request.Imp)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, errors
}

// getMediaTypeForBid determines the media type for a bid.
// It first tries explicit bid response fields, then falls back to the matching
// impression only when the impression has a single media type.
func getMediaTypeForBid(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	// Try bid.ext.prebid.type first
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err == nil && bidExt.Prebid != nil {
			bidType, err := openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
			if err == nil {
				return bidType, nil
			}
		}
	}

	// Try bid.MType (OpenRTB 2.6)
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	}

	// Fallback: detect from the matching impression
	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			return getSingleMediaTypeForImp(imp)
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("failed to determine media type for bid with imp id \"%s\"", bid.ImpID),
	}
}

func getSingleMediaTypeForImp(imp openrtb2.Imp) (openrtb_ext.BidType, error) {
	var mediaTypes []openrtb_ext.BidType
	if imp.Banner != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeBanner)
	}
	if imp.Video != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeVideo)
	}
	if imp.Audio != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeAudio)
	}
	if imp.Native != nil {
		mediaTypes = append(mediaTypes, openrtb_ext.BidTypeNative)
	}

	if len(mediaTypes) == 1 {
		return mediaTypes[0], nil
	}

	if len(mediaTypes) > 1 {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid response must include mtype or ext.prebid.type for multi-format impression with id \"%s\"", imp.ID),
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("failed to determine media type for bid with imp id \"%s\"", imp.ID),
	}
}

// vdoaiImpExt is used to re-marshal the imp ext with the vdoai-specific fields
// preserved under the "bidder" key.
type vdoaiImpExt struct {
	Bidder json.RawMessage `json:"bidder"`
}
