package triplelift 

import (
	//"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/mxmCherry/openrtb"
	//"github.com/prebid/prebid-server/errortypes"
	//"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/adapters"
)

type TripleliftAdapter struct {
    endpoint string
}

func (a *TripleliftAdapter)  MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
    errs := make([]error, 0, len(request.Imp))
    reqs := make([]*adapters.RequestData, 0, 1) 
    reqJSON, err := json.Marshal(request)
    if err != nil {
        errs = append(errs,err)
        return nil, errs
    }
    headers := http.Header{}
    headers.Add("Content-Type","application/json;charset=utf-8")
    headers.Add("Accept", "application/json")
    ad := a.endpoint
    reqs = append(reqs, &adapters.RequestData{
        Method: "POST",
        Uri: ad,
        Body: reqJSON,
        Headers: headers})
    return reqs, errs
}

func getBidCount(seatBid SeatBid) (int) {
    c := 0
    for _, sb := range seatBid {
        c = c + len(sb.Bid)
    }
    return c;
}

func (a *TripleliftAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}
    errs := make([]error,0,2)
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
    count := getBidCount(bidResp.SeatBid)
    bidResponse := adapters.NewBidderResponseWithBidsCapacity(count)

    for _, sb := range bidResp.SeatBig {
        for i := 0; i < len(sb.Bid); i++ {
            bid := sb.Bid[i]
            impVideo = &openrtb_ext.ExtBidPrebidVideo {
                Duration: 2
            }
            bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
                Bid: &bid,
                BidType: openrtb_ext.BidTypeBanner,
                BidVideo: impVideo
            }
        }
    }
    return bidResponse, errs
}

func NewTripleliftBidder(client *http.Client, endpoint string) *TripleliftAdapter {
    return &TripleliftAdapter{
        endpoint: endpoint}
}


