package triplelift_native

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type TripleliftNativeAdapter struct {
	endpoint string
}

type TripleliftInnerExt struct {
	Format int `json:"format"`
}

type TripleliftRespExt struct {
	Triplelift TripleliftInnerExt `json:"triplelift_pb"`
}

type TripleliftNativeExtInfo struct {
	// Array is used for deserialization.
	PublisherWhitelist []string `mapstructure:"publisher_whitelist,flow"`

	// Map is used for optimized memory access and should be constructed after deserialization.
	PublisherWhitelistMap map[string]bool
}

func getBidType(ext TripleliftRespExt) openrtb_ext.BidType {
	return openrtb_ext.BidTypeNative
}

func processImp(imp *openrtb.Imp) error {
	// get the triplelift extension
	var ext adapters.ExtImpBidder
	var tlext openrtb_ext.ExtImpTriplelift
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal(ext.Bidder, &tlext); err != nil {
		return err
	}
	if imp.Native == nil {
		return fmt.Errorf("no native object specified")
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

func (a *TripleliftNativeAdapter) MakeRequests(request *openrtb.BidRequest, extra *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp)+1)
	reqs := make([]*adapters.RequestData, 0, 1)
	// copy the request, because we are going to mutate it
	tlRequest := *request
	// this will contain all the valid impressions
	var validImps []openrtb.Imp
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

func (a *TripleliftNativeAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
			bidType := getBidType(bidExt)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}

func NewTripleliftNativeBidder(client *http.Client, endpoint string, extraInfo string) *TripleliftNativeAdapter {
	var extInfo TripleliftNativeExtInfo

	if err := json.Unmarshal(extraAdapterInfo, &extInfo); err != nil {
		panic("Invalid TripleLife Native extra adapter info: " + err.Message)
	}

	// Populate map for faster memory access (recommended).
	extInfo.PublisherWhitelistMap = make(map[string]bool)
	for _, v := range extInfo.PublisherWhitelist {
		extInfo.PublisherWhitelistMap[v] = true
	}

	return &TripleliftNativeAdapter{
		endpoint: endpoint}
}
