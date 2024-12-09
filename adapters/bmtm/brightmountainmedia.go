package bmtm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the BrightMountainMedia adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var extRequests []*adapters.RequestData
	var errs []error

	for _, imp := range request.Imp {
		extRequest, err := a.makeRequest(*request, imp)
		if err != nil {
			errs = append(errs, err)
		} else {
			extRequests = append(extRequests, extRequest)
		}
	}
	return extRequests, errs
}

func (a *adapter) makeRequest(ortbRequest openrtb2.BidRequest, ortbImp openrtb2.Imp) (*adapters.RequestData, error) {
	if ortbImp.Banner == nil && ortbImp.Video == nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("For Imp ID %s Banner or Video is undefined", ortbImp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(ortbImp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
		}
	}

	var bmtmExt openrtb_ext.ImpExtBmtm
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &bmtmExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBmtm: %s", err.Error()),
		}
	}

	ortbImp.TagID = strconv.Itoa(bmtmExt.PlacementID)
	ortbImp.Ext = nil
	ortbRequest.Imp = []openrtb2.Imp{ortbImp}

	requestJSON, err := json.Marshal(ortbRequest)
	if err != nil {
		return nil, err
	}

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: setHeaders(ortbRequest),
		ImpIDs:  openrtb_ext.GetImpIDs(ortbRequest.Imp),
	}
	return requestData, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unknown status code: %d.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown status code: %d.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForBid(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

func setHeaders(request openrtb2.BidRequest) http.Header {
	headers := http.Header{}

	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Add("User-Agent", request.Device.UA)
		}

		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}

	if request.Site != nil && request.Site.Page != "" {
		headers.Add("Referer", request.Site.Page)
	}
	return headers
}

func getMediaTypeForBid(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}
