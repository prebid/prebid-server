package logan

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	URI string
}

// Builder builds a new instance of the Logan adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests create bid request for logan demand
func (a *adapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) (adapterRequests []*adapters.RequestData, errs []error) {
	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb.Imp{imp}
		reqJSON, _ := json.Marshal(reqCopy)
		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.URI,
			Body:    reqJSON,
			Headers: headers,
		})
	}

	return adapterRequests, errs
}

// MakeBids makes the bids
func (a *adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(bid.ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType, nil
		}
	}
	return mediaType, nil
}
