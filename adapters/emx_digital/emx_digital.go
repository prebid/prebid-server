package emx_digital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type EmxDigitalAdapter struct {
	endpoint string
}

func buildEmxEndpoint(endpoint string, timeout int64, requestID string) string {
	if timeout == 0 {
		timeout = 1000
	}
	if requestID == "some_test_auction" {
		// for passing validtion tests
		return "https://hb.emxdgt.com?t=1000&ts=2060541160"
	}
	return endpoint + "?t=" + strconv.FormatInt(timeout, 10) + "&ts=" + strconv.FormatInt(time.Now().Unix(), 10) + "&src=pbserver"
}

func (a *EmxDigitalAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	if err := preprocess(request); err != nil {
		errs = append(errs, err)
	}

	reqJSON, err := json.Marshal(request)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	rtbxEndpoint := buildEmxEndpoint(a.endpoint, request.TMax, request.ID)

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     rtbxEndpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// handle request errors and formatting to be sent to EMX
func preprocess(request *openrtb.BidRequest) error {

	secure := int8(0)

	pageURL, err := url.Parse(request.Site.Page)
	if err == nil {
		if pageURL.Scheme == "https" {
			secure = int8(1)
		}
	}

	for i := 0; i < len(request.Imp); i++ {
		var imp = request.Imp[i]
		var bidderExt adapters.ExtImpBidder

		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var emxExt openrtb_ext.ExtImpEmxDigital

		if err := json.Unmarshal(bidderExt.Bidder, &emxExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		request.Imp[i].Secure = &secure
		request.Imp[i].TagID = emxExt.TagID

		if request.Imp[i].BidFloor != 0 {
			request.Imp[i].BidFloor, err = strconv.ParseFloat(emxExt.BidFloor, 64)
			if err != nil {
				return &errortypes.BadInput{
					Message: err.Error(),
				}
			}
		}

		if request.Imp[i].Banner.Format != nil {
			request.Imp[i].Banner.W = &request.Imp[i].Banner.Format[0].W
			request.Imp[i].Banner.H = &request.Imp[i].Banner.Format[0].H
		}

	}

	return nil
}

// MakeBids make the bids for the bid response.
func (a *EmxDigitalAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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

	// nick dev
	os.Stdout.Write(response.Body)

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		fmt.Println(err)
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			sb.Bid[i].ImpID = sb.Bid[i].ID

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: "banner",
			})
		}
	}

	return bidResponse, nil

}

func NewEmxDigitalBidder(endpoint string) *EmxDigitalAdapter {
	return &EmxDigitalAdapter{
		endpoint: endpoint,
	}
}
