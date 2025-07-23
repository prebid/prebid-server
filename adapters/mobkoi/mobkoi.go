package mobkoi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	// The endpoint of that the bid requests are sent to. Obtained from the server config that provided at adapter initialisation.
	bidderEndpoint string
}

type BidderExt struct {
	Bidder openrtb_ext.ImpExtMobkoi `json:"bidder"`
}

type UserExt struct {
	Consent string `json:"consent"`
}

// Builder builds a new instance of the {bidder} adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		bidderEndpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	ext := BidderExt{}
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &ext); err != nil {
		return nil, []error{err}
	}

	if request.Imp[0].TagID == "" {
		if ext.Bidder.PlacementID != "" {
			request.Imp[0].TagID = ext.Bidder.PlacementID
		} else {
			return nil, []error{
				errors.New("invalid because it comes with neither request.imp[0].tagId nor req.imp[0].ext.Bidder.placementId"),
			}
		}
	}

	if err := updateRequestExt(request); err != nil {
		return nil, []error{err}
	}

	if request.User != nil && request.User.Consent != "" {
		user := *request.User
		userExt, err := jsonutil.Marshal(UserExt{
			Consent: user.Consent,
		})
		if err != nil {
			return nil, []error{err}
		}
		user.Ext = userExt
		request.User = &user
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")

	bidderEndpoint, err := a.getBidderEndpoint(ext.Bidder)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     bidderEndpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
				Seat:    "mobkoi",
			})

		}
	}
	return bidResponse, nil
}

// This function returns the appropriate bidder endpoint, using the provided
// integration endpoint if valid, otherwise falling back to the bidder endpoint from config.
// Returns an error if no valid endpoint is available.
func (a *adapter) getBidderEndpoint(bidderExt openrtb_ext.ImpExtMobkoi) (string, error) {
	providedEndpoint := bidderExt.IntegrationEndpoint
	if providedEndpoint != "" && isValidURL(providedEndpoint) {
		return providedEndpoint, nil
	}

	if a.bidderEndpoint != "" && isValidURL(a.bidderEndpoint) {
		return a.bidderEndpoint, nil
	}

	return "", fmt.Errorf("no valid endpoint configured: both integration endpoint (%s) and bidder endpoint (%s) are invalid", providedEndpoint, a.bidderEndpoint)
}


// This function checks if the endpoint is a valid URL.
// Example valid and invalid URLs:
// - https://adapter.config.bidder.endpoint.com/bid (valid)
// - https://adapter.config.bidder.endpoint.com:8080/bid (valid)
// - adapter.config.bidder.endpoint.com/bid (invalid)
// - https://adapter.config.bidder.endpoint.com (invalid)
func isValidURL(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	return err == nil && parsed.Scheme != "" && parsed.Host != "" && parsed.Path != "" && parsed.Path != "/"
}

// updateRequestExt sets the mobkoi extension fields in the request extension using standard JSON manipulation.
func updateRequestExt(request *openrtb2.BidRequest) error {
	// Parse existing request.Ext as map[string]json.RawMessage
	extMap := make(map[string]json.RawMessage)
	if request.Ext != nil {
		if err := jsonutil.Unmarshal(request.Ext, &extMap); err != nil {
			return err
		}
	}

	// Create mobkoi extension with integration_type
	mobkoiExt := map[string]interface{}{
		"integration_type": "pbs",
	}

	// Marshal mobkoi extension and add to extMap
	mobkoiBytes, err := jsonutil.Marshal(mobkoiExt)
	if err != nil {
		return err
	}
	extMap["mobkoi"] = mobkoiBytes

	// Re-marshal and update request.Ext
	newExt, err := jsonutil.Marshal(extMap)
	if err != nil {
		return err
	}

	request.Ext = newExt
	return nil
}
