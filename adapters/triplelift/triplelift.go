package triplelift

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type TripleliftAdapter struct {
	endpoint string
}

type TripleliftInnerExt struct {
	Format int `json:"format"`
}

type TripleliftRespExt struct {
	Triplelift TripleliftInnerExt `json:"triplelift_pb"`
}

func getBidType(ext TripleliftRespExt) openrtb_ext.BidType {
	t := ext.Triplelift.Format
	if t == 11 || t == 12 || t == 17 {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}

func processImp(imp *openrtb2.Imp) error {
	// get the triplelift extension
	var ext adapters.ExtImpBidder
	var tlext openrtb_ext.ExtImpTriplelift
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return err
	}
	if err := jsonutil.Unmarshal(ext.Bidder, &tlext); err != nil {
		return err
	}
	if imp.Banner == nil && imp.Video == nil {
		return fmt.Errorf("neither Banner nor Video object specified")
	}
	imp.TagID = tlext.InvCode
	// floor is optional
	if tlext.Floor == nil {
		return nil
	} else {
		imp.BidFloor = *tlext.Floor
	}
	// no error
	return nil
}

func (a *TripleliftAdapter) MakeRequests(request *openrtb2.BidRequest, extra *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp)+1)
	reqs := make([]*adapters.RequestData, 0, 1)
	// copy the request, because we are going to mutate it
	tlRequest := *request
	// this will contain all the valid impressions
	var validImps []openrtb2.Imp
	// pre-process the imps
	for _, imp := range tlRequest.Imp {
		if err := processImp(&imp); err == nil {
			validImps = append(validImps, imp)
		} else {
			errs = append(errs, err)
		}
	}
	if len(validImps) == 0 {
		err := fmt.Errorf("No valid impressions for triplelift")
		errs = append(errs, err)
		return nil, errs
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
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(tlRequest.Imp)})
	return reqs, errs
}

func getBidCount(bidResponse openrtb2.BidResponse) int {
	c := 0
	for _, sb := range bidResponse.SeatBid {
		c = c + len(sb.Bid)
	}
	return c
}

func (a *TripleliftAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	var errs []error
	count := getBidCount(bidResp)
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(count)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			var bidExt TripleliftRespExt
			if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, err)
			} else {
				bidType := getBidType(bidExt)
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, errs
}

// Builder builds a new instance of the Triplelift adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &TripleliftAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
