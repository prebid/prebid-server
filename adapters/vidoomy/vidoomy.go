package vidoomy

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
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

type adapter struct {
	endpoint string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	reqs := make([]*adapters.RequestData, 0, len(request.Imp))

	header := getHeaders(request)
	for _, imp := range request.Imp {

		// Split up multi-impression requests into multiple requests so that
		// each split request is only associated to a single impression
		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}

		if err := changeRequestForBidService(&reqCopy); err != nil {
			errors = append(errors, err)
			continue
		}

		reqJSON, err := json.Marshal(&reqCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: header,
			ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
		})
	}

	return reqs, errors
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}

	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	if request.Device == nil {
		return headers
	}

	if request.Device.UA != "" {
		headers.Set("User-Agent", request.Device.UA)
	}

	if len(request.Device.IPv6) > 0 {
		headers.Add("X-Forwarded-For", request.Device.IPv6)
	}

	if len(request.Device.IP) > 0 {
		headers.Add("X-Forwarded-For", request.Device.IP)
	}

	return headers
}

func changeRequestForBidService(request *openrtb2.BidRequest) error {
	if request.Imp[0].Banner == nil {
		return nil
	}

	banner := *request.Imp[0].Banner
	request.Imp[0].Banner = &banner

	if banner.W != nil && banner.H != nil {
		if *banner.W == 0 || *banner.H == 0 {
			return fmt.Errorf("invalid sizes provided for Banner %d x %d", *banner.W, *banner.H)
		}
		return nil
	}

	if len(banner.Format) == 0 {
		return fmt.Errorf("no sizes provided for Banner %v", banner.Format)
	}

	banner.W = ptrutil.ToPtr(banner.Format[0].W)
	banner.H = ptrutil.ToPtr(banner.Format[0].H)

	return nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d.", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bid := bid
			exists, mediaType := getImpInfo(bid.ImpID, internalRequest.Imp)
			if !exists {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}}
			}

			if openrtb_ext.BidTypeBanner != mediaType &&
				openrtb_ext.BidTypeVideo != mediaType {
				//only banner and video are supported, anything else is ignored
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mediaType,
			})
		}
	}

	return bidResponse, nil
}

func getImpInfo(impId string, imps []openrtb2.Imp) (bool, openrtb_ext.BidType) {
	var mediaType openrtb_ext.BidType
	for _, imp := range imps {
		if imp.ID == impId {

			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			}

			return true, mediaType
		}
	}
	return false, mediaType
}

// Builder builds a new instance of the Vidoomy adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
