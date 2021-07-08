package telaria

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const Endpoint = "https://ads.tremorhub.com/ad/rtb/prebid"

type TelariaAdapter struct {
	URI string
}

// This will be part of Imp[i].Ext when this adapter calls out the Telaria Ad Server
type ImpressionExtOut struct {
	OriginalTagID       string `json:"originalTagid"`
	OriginalPublisherID string `json:"originalPublisherid"`
}

type telariaBidExt struct {
	Extra json.RawMessage `json:"extra,omitempty"`
}

// Endpoint for Telaria Ad server
func (a *TelariaAdapter) FetchEndpoint() string {
	return a.URI
}

// Checker method to ensure len(request.Imp) > 0
func (a *TelariaAdapter) CheckHasImps(request *openrtb2.BidRequest) error {
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "Telaria: Missing Imp Object",
		}
		return err
	}
	return nil
}

// Checking if Imp[i].Video exists and Imp[i].Banner doesn't exist
func (a *TelariaAdapter) CheckHasVideoObject(request *openrtb2.BidRequest) error {
	hasVideoObject := false

	for _, imp := range request.Imp {
		if imp.Banner != nil {
			return &errortypes.BadInput{
				Message: "Telaria: Banner not supported",
			}
		}

		hasVideoObject = hasVideoObject || imp.Video != nil
	}

	if !hasVideoObject {
		return &errortypes.BadInput{
			Message: "Telaria: Only Supports Video",
		}
	}

	return nil
}

// Fetches the populated header object
func GetHeaders(request *openrtb2.BidRequest) *http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}

		if len(request.Device.Language) > 0 {
			headers.Add("Accept-Language", request.Device.Language)
		}

		if request.Device.DNT != nil {
			headers.Add("Dnt", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	return &headers
}

// Checks the imp[i].ext object and returns a imp.ext object as per ExtImpTelaria format
func (a *TelariaAdapter) FetchTelariaExtImpParams(imp *openrtb2.Imp) (*openrtb_ext.ExtImpTelaria, error) {
	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(imp.Ext, &bidderExt)

	if err != nil {
		err = &errortypes.BadInput{
			Message: "Telaria: ext.bidder not provided",
		}

		return nil, err
	}

	var telariaExt openrtb_ext.ExtImpTelaria
	err = json.Unmarshal(bidderExt.Bidder, &telariaExt)

	if err != nil {
		return nil, err
	}

	if telariaExt.SeatCode == "" {
		return nil, &errortypes.BadInput{Message: "Telaria: Seat Code required"}
	}

	return &telariaExt, nil
}

// Method to fetch the original publisher ID. Note that this method must be called
// before we replace publisher.ID with seatCode
func (a *TelariaAdapter) FetchOriginalPublisherID(request *openrtb2.BidRequest) string {

	if request.Site != nil && request.Site.Publisher != nil {
		return request.Site.Publisher.ID
	} else if request.App != nil && request.App.Publisher != nil {
		return request.App.Publisher.ID
	}

	return ""
}

// Method to do a deep copy of the publisher object. It also adds the seatCode as publisher.ID
func (a *TelariaAdapter) MakePublisherObject(seatCode string, publisher *openrtb2.Publisher) *openrtb2.Publisher {
	var pub = &openrtb2.Publisher{ID: seatCode}

	if publisher != nil {
		pub.Domain = publisher.Domain
		pub.Name = publisher.Name
		pub.Cat = publisher.Cat
		pub.Ext = publisher.Ext
	}

	return pub
}

// This method changes <site/app>.publisher.id to the seatCode
func (a *TelariaAdapter) PopulatePublisherId(request *openrtb2.BidRequest, seatCode string) (*openrtb2.Site, *openrtb2.App) {
	if request.Site != nil {
		siteCopy := *request.Site
		siteCopy.Publisher = a.MakePublisherObject(seatCode, request.Site.Publisher)
		return &siteCopy, nil
	} else if request.App != nil {
		appCopy := *request.App
		appCopy.Publisher = a.MakePublisherObject(seatCode, request.App.Publisher)
		return nil, &appCopy
	}
	return nil, nil
}

func (a *TelariaAdapter) MakeRequests(requestIn *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	// make a copy of the incoming request
	request := *requestIn

	// ensure that the request has Impressions
	if noImps := a.CheckHasImps(&request); noImps != nil {
		return nil, []error{noImps}
	}

	// ensure that the request has a Video object
	if noVideoObjectError := a.CheckHasVideoObject(&request); noVideoObjectError != nil {
		return nil, []error{noVideoObjectError}
	}

	var seatCode string
	originalPublisherID := a.FetchOriginalPublisherID(&request)

	var errors []error
	var telariaImpExt *openrtb_ext.ExtImpTelaria
	var err error
	for i, imp := range request.Imp {
		// fetch adCode & seatCode from Imp[i].Ext
		telariaImpExt, err = a.FetchTelariaExtImpParams(&imp)
		if err != nil {
			errors = append(errors, err)
			break
		}

		seatCode = telariaImpExt.SeatCode

		// move the original tagId and the original publisher.id into the Imp[i].Ext object
		request.Imp[i].Ext, err = json.Marshal(&ImpressionExtOut{request.Imp[i].TagID, originalPublisherID})
		if err != nil {
			errors = append(errors, err)
			break
		}

		// Swap the tagID with adCode
		request.Imp[i].TagID = telariaImpExt.AdCode
	}

	// Add the Extra from Imp to the top level Ext
	if telariaImpExt != nil && telariaImpExt.Extra != nil {
		request.Ext, err = json.Marshal(&telariaBidExt{Extra: telariaImpExt.Extra})
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	// Add seatCode to <Site/App>.Publisher.ID
	siteObject, appObject := a.PopulatePublisherId(&request, seatCode)

	request.Site = siteObject
	request.App = appObject

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.FetchEndpoint(),
		Body:    reqJSON,
		Headers: *GetHeaders(&request),
	}}, nil
}

func (a *TelariaAdapter) CheckResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusNoContent {
		return &errortypes.BadInput{Message: "Telaria: Invalid Bid Request received by the server"}
	}

	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Telaria: Unexpected status code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Telaria: Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Telaria: Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	return nil
}

func (a *TelariaAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	httpStatusError := a.CheckResponseStatusCodes(response)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	responseBody := response.Body

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Telaria: Bad Server Response",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	sb := bidResp.SeatBid[0]

	for _, bid := range sb.Bid {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: openrtb_ext.BidTypeVideo,
		})
	}
	return bidResponse, nil
}

// Builder builds a new instance of the Telaria adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = Endpoint // Hardcoded default
	}

	bidder := &TelariaAdapter{
		URI: endpoint,
	}
	return bidder, nil
}
