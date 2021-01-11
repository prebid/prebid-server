package foreshop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Adapter hosts the Foreshop methods for making requests and bids, which are sent to its "Endpoint" address
type Adapter struct {
	Endpoint string
}

// Builder builds instantiates a Foreshop adapter using the given parameters.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &Adapter{
		Endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests processes request data and returns a prebid-server adapter RequestData struct for making requests
func (a *Adapter) MakeRequests(request *openrtb.BidRequest, extraInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	// Headers that will be used in the final RequestData struct
	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	// Holds the request Imps
	var imps = make(map[openrtb_ext.ExtImpForeshop][]openrtb.Imp)

	// Loops through the request's array of Imps to retrieve bidder data, which is used for the request body
	for _, imp := range request.Imp {
		d, err := getBidderData(&imp)
		if err != nil {
			errs = append(errs, err)
		}

		imps[*d] = append(imps[*d], imp)
	}

	// Holds the requests that this function returns
	requests := []*adapters.RequestData{}

	// Loops through the Imps to create a request for each
	for ext, imp := range imps {
		// Imp is assigned here so we can make a json out of the request param
		request.Imp = imp

		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
		}

		// Adds ExtImpForeshop properties to the URL
		urlParams := macros.EndpointTemplateParams{Host: ext.Host, SourceId: strconv.Itoa(ext.SourceID)}
		url, err := macros.ResolveMacros(template.Template{}, urlParams)

		// Creates the request struct
		r := adapters.RequestData{
			Method:  "POST",
			Uri:     url,
			Body:    body,
			Headers: headers,
		}

		requests = append(requests, &r)
	}

	return requests, errs
}

// MakeBids is responsible for handling the return response from a bid request
func (a *Adapter) MakeBids(bidReq *openrtb.BidRequest, unused *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// Handles status codes
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed with response status code %v", response.StatusCode),
		}}
	}

	// Extracts the JSON into a struct
	var bidBody openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidBody); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error in unmarshalling response body for MakeBids: %s", err.Error()),
		}}
	}

	// Instantiates a bid response
	bidResponse := adapters.NewBidderResponse()
	bidResponse.Currency = bidBody.Cur

	// Loops through the seatBids, appending the Bid info to the bidResponse
	// for _, seatBid := range bidBody.SeatBid {
	for _, seatBid := range bidBody.SeatBid {
		for i := 0; i < len(seatBid.Bid); i++ {
			bid := seatBid.Bid[i]

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaType(bid.ImpID, bidReq.Imp),
			})
		}
	}

	return bidResponse, nil
}

// getMediaType matches an impID in the array of imps provided, returning the imp array index's bid type.
// If no type is found, then the imp is defaulted to the banner type
func getMediaType(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
	var mediaType openrtb_ext.BidType

	for _, imp := range imps {
		if imp.ID == impID {
			switch {
			case imp.Video != nil:
				mediaType = openrtb_ext.BidTypeVideo
			case imp.Native != nil:
				mediaType = openrtb_ext.BidTypeNative
			case imp.Audio != nil:
				mediaType = openrtb_ext.BidTypeAudio
			default:
				mediaType = openrtb_ext.BidTypeBanner
			}
		}
	}

	return mediaType
}

// getBidderData is used to extract bidder info from the received requests' Imps
func getBidderData(imp *openrtb.Imp) (*openrtb_ext.ExtImpForeshop, error) {
	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Skipping imp with ID %s, failed to unmarshal bidder ext: %s", imp.ID, err.Error()),
		}
	}

	fsImpExt := openrtb_ext.ExtImpForeshop{}
	if err := json.Unmarshal(bidderExt.Bidder, &fsImpExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Skipping imp with ID %s, failed to unmarshal bidder ext to foreshop imp ext: %s", imp.ID, err.Error()),
		}
	}

	if fsImpExt.SourceID < 1 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Skipping imp with ID %s, invalid source ID: %v", imp.ID, fsImpExt.SourceID),
		}
	}

	if fsImpExt.PlacementID < 1 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Skipping imp with ID %s, invalid placement ID: %v", imp.ID, fsImpExt.PlacementID),
		}
	}

	if len(fsImpExt.Host) == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Skipping imp with ID %s, invalid host: %s", imp.ID, fsImpExt.Host),
		}
	}

	return &fsImpExt, nil
}
