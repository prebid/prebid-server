package triplelift

import (
	//"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type TripleliftAdapter struct {
	endpoint string
}

type TripleliftRespExtTriplelift struct {
	format int `json:"format"`
}

type TripleliftRespExt struct {
	Triplelift TripleliftRespExtTriplelift `json:"triplelift_pb"`
}

func getBidType(ext TripleliftRespExt) (openrtb_ext.BidType, error) {
	t := ext.Triplelift.format
	if t == 2 || t == 8 || t == 11 {
		return openrtb_ext.BidTypeVideo, nil
	}
	if t == 10 {
		return openrtb_ext.BidTypeBanner, nil
	}
	return openrtb_ext.BidTypeNative, nil
}

func processImp(imp openrtb.Imp) (error) {
    // get the triplelift extension
    var ext adapters.ExtImpBidder
    var tlext ExtImpTriplelift
    if err = json.Unmarshal(imp.Ext, &ext); err != nil {
        return err
    }
    if err = json.Unmarshal(ext.Bidder, &tlext); err != nil {
        return err
    }
    imp.TagId = tlext.InvCode
    imp.BidFloor = tlext.Floor
}

func (a *TripleliftAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	reqs := make([]*adapters.RequestData, 0, 1)
    // copy the request, because we are going to mutate it
    tlRequest := *request
    // this will contain all the valid impressions
    var validImps []openrtb.Imp
    // pre-process the imps
    for _, imp := range tlRequest.Imp {
        if err := processImp(&imp); err == nil {
            append(validImps, imp)
        }
    }
    tlRequest.Imp = validImps
    reqJSON, err := json.Marshal(tlRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	ad := a.endpoint
	reqs = append(reqs, &adapters.RequestData{
		Method:  "POST",
		Uri:     ad,
		Body:    reqJSON,
		Headers: headers})
	return reqs, errs
}

func getBidCount(bidResponse openrtb.BidResponse) int {
	c := 0
	for _, sb := range bidResponse.SeatBid {
		c = c + len(sb.Bid)
	}
	return c
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
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	var errs []error
	count := getBidCount(bidResp)
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(count)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			var bidExt TripleliftRespExt
			if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, err)
			} else {
				if bidType, err := getBidType(bidExt); err != nil {
					errs = append(errs, err)
				} else {
					bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
						Bid:     &bid,
						BidType: bidType,
					})
				}
			}
		}
	}
	return bidResponse, errs
}

func NewTripleliftBidder(client *http.Client, endpoint string) *TripleliftAdapter {
	return &TripleliftAdapter{
		endpoint: endpoint}
}
