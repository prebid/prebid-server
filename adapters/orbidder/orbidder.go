package orbidder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type OrbidderAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids from orbidder.
func (rcv *OrbidderAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(request)

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     rcv.endpoint,
		Body:    body.Bytes(),
		Headers: headers,
	}
	adapterRequests = append(adapterRequests, reqData)
	return adapterRequests, errs
}

// MakeBids unpacks server response into Bids.
func (rcv OrbidderAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Append debug=1 as request parameter for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Append debug=1 as request parameter for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := &sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bidResponse, nil
}

func NewOrbidderBidder(endpoint string) *OrbidderAdapter {
	return &OrbidderAdapter{
		endpoint: endpoint,
	}
}

func UnmarshalOrbidderExtImp(ext json.RawMessage) (*openrtb_ext.ExtImpOrbidder, error) {
	extImpBidder := new(adapters.ExtImpBidder)
	if err := json.Unmarshal(ext, extImpBidder); err != nil {
		return nil, err
	}

	impExt := new(openrtb_ext.ExtImpOrbidder)
	if err := json.Unmarshal(extImpBidder.Bidder, impExt); err != nil {
		return nil, err
	}

	return impExt, nil
}
