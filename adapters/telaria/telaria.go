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

const Endpoint = "https://ads.vhfp.net/ad/rtb/prebid"

type TelariaAdapter struct {
	URI string
}

type TagIDExt struct {
	OriginalTagID string `json:"originalTagid"`
}

// used for cookies and such
func (a *TelariaAdapter) Name() string {
	return "telaria"
}

func (a *TelariaAdapter) SkipNoCookies() bool {
	return false
}

func (a *TelariaAdapter) FetchEndpoint() string {
	return a.URI
}

func GetHeaders(request *openrtb.BidRequest) *http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")
	headers.Add("Accept-Encoding", "gzip")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("x-Forwarded-For", request.Device.IP)
		}

		if len(request.Device.Language) > 0 {
			headers.Add("Accept-Language", request.Device.Language)
		}

		if request.Device.DNT != nil {
			headers.Add("DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	return &headers
}

func (a *TelariaAdapter) CheckHasImps(request *openrtb.BidRequest) error {
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No imp object in the bid request",
		}
		return err
	}
	return nil
}

func (a *TelariaAdapter) FetchBidderExt(request *openrtb.BidRequest) (*adapters.ExtImpBidder, error) {
	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(request.Imp[0].Ext, &bidderExt)

	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}

		return nil, err
	}

	return &bidderExt, nil
}

func (a *TelariaAdapter) FetchTelariaParams(request *openrtb.BidRequest) (*openrtb_ext.ExtImpTelaria, error) {
	bidderExt, err := a.FetchBidderExt(request)
	if err != nil {
		return nil, err
	}

	var telariaExt openrtb_ext.ExtImpTelaria
	err = json.Unmarshal(bidderExt.Bidder, &telariaExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder.adCode not provided",
		}

		return nil, err
	}

	if telariaExt.AdCode == "" {
		err = &errortypes.BadInput{
			Message: "adCode is empty",
		}

		return nil, err
	}

	return &telariaExt, nil
}

func (a *TelariaAdapter) CheckHasVideoObject(request *openrtb.BidRequest) error {
	hasVideoObject := false

	for _, imp := range request.Imp {
		hasVideoObject = hasVideoObject || imp.Video != nil
	}

	if !hasVideoObject {
		return &errortypes.BadInput{
			Message: "Telaria only supports Video",
		}
	}

	return nil
}

func (a *TelariaAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	if noImps := a.CheckHasImps(request); noImps != nil {
		return nil, []error{noImps}
	}

	if noVideoObject := a.CheckHasVideoObject(request); noVideoObject != nil {
		return nil, []error{noVideoObject}
	}

	telariaExt, err := a.FetchTelariaParams(request)
	if err != nil {
		return nil, []error{err}
	}

	for i, _ := range request.Imp {
		impExt := &TagIDExt{request.Imp[i].TagID}
		request.Imp[i].TagID = telariaExt.AdCode

		if impExt.OriginalTagID != "" {
			request.Imp[i].Ext, _ = json.Marshal(impExt)
		}
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
func GetResponseBody(response *adapters.ResponseData) (*[]byte, error) {
	responseBody := response.Body

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
		responseBody = resB.Bytes()
	}

	return &responseBody, nil
}

func (a *TelariaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ] . ", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code:[ %d ]. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	responseBody, err := GetResponseBody(response)

	if err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(*responseBody, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: [ %d ]. ", err),
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
	return &TelariaAdapter{
		URI: Endpoint,
	}
}
