package telaria

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
)

const Endpoint = "fubarTodoChange"

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

func (a *TelariaAdapter) GetHeaders(request *openrtb.BidRequest) http.Header {
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

	return headers
}

func (a *TelariaAdapter) MakeRequests(requestIn *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	request := *requestIn
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No imp object in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}

	errors := make([]error, 0, 1)

	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(request.Imp[0].Ext, &bidderExt)

	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}

	var telariaExt openrtb_ext.ExtImpTelaria
	err = json.Unmarshal(bidderExt.Bidder, &telariaExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder.adCode not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}

	if telariaExt.AdCode == "" {
		err = &errortypes.BadInput{
			Message: "adCode is empty",
		}
		errors = append(errors, err)
		return nil, errors
	}
	validImpExists := false

	for i, imp := range request.Imp {
		validImpExists = validImpExists || imp.Video != nil
		var impExt = &TagIDExt{request.Imp[i].TagID}
		request.Imp[i].TagID = telariaExt.AdCode
		if impExt.OriginalTagID != "" {
			request.Imp[i].Ext, _ = json.Marshal(impExt)
		}

	}

	if !validImpExists {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("No valid impression in the bid request"),
		}
		errors = append(errors, err)
		return nil, errors
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.FetchEndpoint(),
		Body:    reqJSON,
		Headers: a.GetHeaders(&request),
	}}, errors
}

func (a *TelariaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. ", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
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
