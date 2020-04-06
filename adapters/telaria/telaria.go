package telaria

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
)

const Endpoint = "https://ads.tremorhub.com/ad/rtb/prebid"

type TelariaAdapter struct {
	URI string
}

// This will be part of Imp[i].Ext when this adapter calls out the Telaria Ad Server
type ImpressionExtOut struct {
	OriginalTagID       string `json:"originalTagid"`
	OriginalPublisherID string `json:"originalPublisherId"`
}

/*
// Full request extension including Telaria extension object
type ReqExtIn struct {
	openrtb_ext.ExtRequest
	Telaria *ReqExtTelariaIn `json:"telaria,omitempty"`
}*/

// used for cookies and such
func (a *TelariaAdapter) Name() string {
	return "telaria"
}

func (a *TelariaAdapter) SkipNoCookies() bool {
	return false
}

// Endpoint for Telaria Ad server
func (a *TelariaAdapter) FetchEndpoint() string {
	return a.URI
}

// Checker method to ensure len(request.Imp) > 0
func (a *TelariaAdapter) CheckHasImps(request *openrtb.BidRequest) error {
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No imp object in the bid request",
		}
		return err
	}
	return nil
}

// Checking if Imp[i].Video exists and Imp[i].Banner doesn't exist
func (a *TelariaAdapter) CheckHasVideoObject(request *openrtb.BidRequest) error {
	hasVideoObject := false

	for _, imp := range request.Imp {
		if imp.Banner != nil {
			return &errortypes.BadInput{
				Message: "Telaria doesn't support banner",
			}
		}

		hasVideoObject = hasVideoObject || imp.Video != nil
	}

	if !hasVideoObject {
		return &errortypes.BadInput{
			Message: "No Video object present in Imp object",
		}
	}

	return nil
}

// Fetches the populated header object
func GetHeaders(request *openrtb.BidRequest) *http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")
	headers.Add("Accept-Encoding", "gzip")

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
func (a *TelariaAdapter) FetchTelariaExtImpParams(imp *openrtb.Imp) (*openrtb_ext.ExtImpTelaria, error) {
	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(imp.Ext, &bidderExt)

	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}

		return nil, err
	}

	var telariaExt openrtb_ext.ExtImpTelaria
	err = json.Unmarshal(bidderExt.Bidder, &telariaExt)

	if err != nil {
		return nil, err
	}

	if telariaExt.SeatCode == "" {
		return nil, &errortypes.BadInput{Message: "Seat Code required"}
	}

	return &telariaExt, nil
}

// This method changes <site/app>.publisher.id to request.ext.telaria.seatCode
// And moves the publisher.id to request.ext.originalPublisherId
func (a *TelariaAdapter) PopulatePublisherId(request *openrtb.BidRequest, impExt *openrtb_ext.ExtImpTelaria) (string, error) {

	publisherObject := &openrtb.Publisher{ID: impExt.SeatCode}
	originalPubId := ""

	if request.Site != nil {
		if request.Site.Publisher != nil {
			if request.Site.Publisher.ID != "" {
				originalPubId = request.Site.Publisher.ID
				request.Site.Publisher.ID = impExt.SeatCode
			}
		} else {
			request.Site.Publisher = publisherObject
		}
	}

	if request.App != nil {
		if request.App.Publisher != nil {
			if request.App.Publisher.ID != "" {
				originalPubId = request.App.Publisher.ID
				request.App.Publisher.ID = impExt.SeatCode
			}
		} else {
			request.App.Publisher = publisherObject
		}
	}

	return originalPubId, nil
}

func (a *TelariaAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	if noImps := a.CheckHasImps(request); noImps != nil {
		return nil, []error{noImps}
	}

	if noVideoObjectError := a.CheckHasVideoObject(request); noVideoObjectError != nil {
		return nil, []error{noVideoObjectError}
	}

	var errors []error
	for i, imp := range request.Imp {
		telariaExt, err := a.FetchTelariaExtImpParams(&imp)

		if err != nil {
			errors = append(errors, err)
			break
		}

		var originalPublisherID string

		originalPublisherID, err = a.PopulatePublisherId(request, telariaExt)
		if err != nil {
			errors = append(errors, err)
			break
		}

		request.Imp[i].TagID = telariaExt.AdCode
		request.Imp[i].Ext, err = json.Marshal(&ImpressionExtOut{request.Imp[i].TagID, originalPublisherID})
		if err != nil {
			errors = append(errors, err)
			break
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	reqJSON, err := json.Marshal(request)

	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.FetchEndpoint(),
		Body:    reqJSON,
		Headers: *GetHeaders(request),
	}}, nil
}

// response isn't automatically decompressed. This method unzips the response if Content-Encoding is gzip
func GetResponseBody(response *adapters.ResponseData) ([]byte, error) {

	if "gzip" == response.Headers.Get("Content-Encoding") {
		body := bytes.NewBuffer(response.Body)
		r, readerErr := gzip.NewReader(body)
		if readerErr != nil {
			return nil, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Error while trying to unzip data [ %d ]", response.StatusCode),
			}
		}
		var resB bytes.Buffer
		var err error
		_, err = resB.ReadFrom(r)
		if err != nil {
			return nil, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Error while trying to unzip data [ %d ]", response.StatusCode),
			}
		}

		response.Headers.Del("Content-Encoding")

		return resB.Bytes(), nil
	} else {
		return response.Body, nil
	}
}

func (a *TelariaAdapter) CheckResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusNoContent {
		return &errortypes.BadInput{Message: "Invalid Bid Request received by the server"}
	}

	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	return nil
}

func (a *TelariaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	httpStatusError := a.CheckResponseStatusCodes(response)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	responseBody, err := GetResponseBody(response)

	if err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "bad server response",
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

func NewTelariaBidder(endpoint string) *TelariaAdapter {
	if endpoint == "" {
		endpoint = Endpoint
	}

	return &TelariaAdapter{
		URI: endpoint,
	}
}
