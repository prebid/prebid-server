package generic

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint string
}

type genericBidExt struct {
	VideoCreativeInfo *genericBidExtVideo `json:"video,omitempty"`
}

type genericBidExtVideo struct {
	Duration *int `json:"duration,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			var mediaType = getBidType(sb.Bid[i].ImpID, internalRequest.Imp)
			bid := sb.Bid[i]
			typedBid := &adapters.TypedBid{
				Bid:      &bid,
				BidType:  mediaType,
				BidVideo: &openrtb_ext.ExtBidPrebidVideo{},
			}
			if bid.Ext != nil {
				var bidExt *genericBidExt
				err := json.Unmarshal(bid.Ext, &bidExt)
				if err != nil {
					return nil, []error{fmt.Errorf("bid.ext json unmarshal error")}
				} else if bidExt != nil {
					if bidExt.VideoCreativeInfo != nil && bidExt.VideoCreativeInfo.Duration != nil {
						typedBid.BidVideo.Duration = *bidExt.VideoCreativeInfo.Duration
					}
				}
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}
	return bidResponse, nil

}

func getBidType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	bidType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				break
			}
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
				break
			}
			if imp.Native != nil {
				bidType = openrtb_ext.BidTypeNative
				break
			}

		}
	}
	return bidType
}
