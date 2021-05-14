package adot

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

type adotBidExt struct {
	Adot bidExt `json:"adot"`
}

type bidExt struct {
	MediaType string `json:"media_type"`
}

// Builder builds a new instance of the Adot adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var reqJSON []byte
	var err error

	if reqJSON, err = json.Marshal(request); err != nil {
		return nil, []error{fmt.Errorf("unable to marshal openrtb request (%v)", err)}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, nil
}

// MakeBids unpacks the server's response into Bids.
// The bidder return a status code 204 when it cannot delivery an ad.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	bidsCapacity := 1
	if len(bidResp.SeatBid) > 0 {
		bidsCapacity = len(bidResp.SeatBid[0].Bid)
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(bidsCapacity)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			if bidType, err := getMediaTypeForBid(&sb.Bid[i]); err == nil {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid == nil {
		return "", fmt.Errorf("the bid request object is nil")
	}

	var impExt adotBidExt
	if err := json.Unmarshal(bid.Ext, &impExt); err == nil {
		switch impExt.Adot.MediaType {
		case string(openrtb_ext.BidTypeBanner):
			return openrtb_ext.BidTypeBanner, nil
		case string(openrtb_ext.BidTypeVideo):
			return openrtb_ext.BidTypeVideo, nil
		case string(openrtb_ext.BidTypeNative):
			return openrtb_ext.BidTypeNative, nil
		}
	}

	return "", fmt.Errorf("unrecognized bid type in response from adot")
}
