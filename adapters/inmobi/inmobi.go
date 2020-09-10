package inmobi

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type InmobiAdapter struct {
	endPoint string
}

func NewInmobiAdapter(endpoint string) *InmobiAdapter {
	return &InmobiAdapter{
		endPoint: endpoint,
	}
}

func (a *InmobiAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	for _, imp := range request.Imp {
		if imp.Banner != nil {
			banner := imp.Banner
			if (banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0) && len(banner.Format) > 0 {
				format := banner.Format[0]
				banner.W = &format.W
				banner.H = &format.H
			}
		}
	}

	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endPoint,
		Body:    reqJson,
		Headers: headers,
	}}, errs
}

func (a *InmobiAdapter) MakeBids(interalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d", response.StatusCode),
		}}
	}

	var serverBidResponse openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &serverBidResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range serverBidResponse.SeatBid {
		for i := range sb.Bid {
			mediaType := getMediaTypeForImp(sb.Bid[i].ImpID, interalRequest.Imp)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}

	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			break
		}
	}
	return mediaType
}
