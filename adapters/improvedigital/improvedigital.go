package improvedigital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type ImprovedigitalAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *ImprovedigitalAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)
	errors := make([]error, 0)
	adapterRequests := make([]*adapters.RequestData, 0, numRequests)

	// Split multi-imp request into multiple ad server requests. SRA is currently not recommended.
	for i := 0; i < numRequests; i++ {
		if adapterReq, err := a.makeRequest(*request, request.Imp[i]); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errors = append(errors, err)
		}
	}

	return adapterRequests, errors
}

func (a *ImprovedigitalAdapter) makeRequest(request openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	request.Imp = []openrtb2.Imp{imp}

	request = a.applyAdditionalConsentProvidersArray(request)
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, nil
}

// MakeBids unpacks the server's response into Bids.
func (a *ImprovedigitalAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	if len(bidResp.SeatBid) > 1 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected SeatBid! Must be only one but have: %d", len(bidResp.SeatBid)),
		}}
	}

	seatBid := bidResp.SeatBid[0]

	if len(seatBid.Bid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(seatBid.Bid))

	for i := range seatBid.Bid {
		bid := seatBid.Bid[i]

		bidType, err := getMediaTypeForImp(bid.ImpID, internalRequest.Imp)
		if err != nil {
			return nil, []error{err}
		}

		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}
	return bidResponse, nil

}

// Builder builds a new instance of the Improvedigital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &ImprovedigitalAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}

			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}

			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}

			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unknown impression type for ID: \"%s\"", impID),
			}
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression for ID: \"%s\"", impID),
	}
}

// This method responsible to clone request and convert additional consent providers string to array when additional consent provider found
func (a *ImprovedigitalAdapter) applyAdditionalConsentProvidersArray(request openrtb2.BidRequest) openrtb2.BidRequest {
	// If user/user.ext not defined, no need to parse additional consent
	if request.User == nil || request.User.Ext == nil {
		return request
	}

	var (
		err     error
		reqJSON json.RawMessage
		cpStr   string
	)

	// Start validating additional consent
	// Check key exist user.ext.ConsentedProvidersSettings.consented_providers
	var userExtMap = make(map[string]json.RawMessage)
	if err = json.Unmarshal(request.User.Ext, &userExtMap); err != nil {
		return request
	}

	cpsMapValue, cpsJSONFound := userExtMap["ConsentedProvidersSettings"]
	if !cpsJSONFound {
		return request
	}

	// Check key exist user.ext.ConsentedProvidersSettings.consented_providers
	var cpMap = make(map[string]json.RawMessage)
	if err = json.Unmarshal(cpsMapValue, &cpMap); err != nil {
		return request
	}

	cpMapValue, cpJSONFound := cpMap["consented_providers"]
	if !cpJSONFound {
		return request
	}
	// End validating additional consent

	// Check if string contain ~, then substring after ~ to end of string
	consentStr := string(cpMapValue)
	var tildaPosition int
	if tildaPosition = strings.Index(consentStr, "~"); tildaPosition == -1 {
		return request
	}
	cpStr = consentStr[tildaPosition+1 : len(consentStr)-1]

	// Prepare consent providers string for append
	cpStr = fmt.Sprintf("[%s]", strings.Replace(cpStr, ".", ",", -1))
	cpMap["consented_providers"] = json.RawMessage(cpStr)

	cpJSON, _ := json.Marshal(cpMap)
	userExtMap["consented_providers_settings"] = cpJSON

	extJson, extErr := json.Marshal(userExtMap)
	if extErr != nil {
		return request
	}

	// Start request closing
	reqJSON, err = json.Marshal(request)
	if err != nil {
		return request
	}

	var reqClone openrtb2.BidRequest
	if err = json.Unmarshal(reqJSON, &reqClone); err != nil {
		return request
	}
	// End request closing

	reqClone.User.Ext = extJson

	return reqClone
}
