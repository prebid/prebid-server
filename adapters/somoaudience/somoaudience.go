package somoaudience

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type SomoaudienceAdapter struct {
	endpoint string
}

func (a *SomoaudienceAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {

	totalImps := len(request.Imp)
	errors := make([]error, 0, totalImps)
	imp2placement := make(map[string][]int)

	for i := 0; i < totalImps; i++ {

		placementHash, err := validateImpression(&request.Imp[i])

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if _, ok := imp2placement[placementHash]; !ok {
			imp2placement[placementHash] = make([]int, 0, totalImps-i)
		}

		imp2placement[placementHash] = append(imp2placement[placementHash], i)

	}

	totalReqs := len(imp2placement)
	if 0 == totalReqs {
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	reqs := make([]*adapters.RequestData, 0, totalReqs)

	imps := request.Imp
	request.Imp = make([]openrtb.Imp, 0, len(imps))

	for placementHash, impIds := range imp2placement {
		request.Imp = request.Imp[:0]

		for i := 0; i < len(impIds); i++ {
			request.Imp = append(request.Imp, imps[impIds[i]])
		}

		body, err := json.Marshal(request)
		if err != nil {
			errors = append(errors, fmt.Errorf("error while encoding bidRequest, err: %s", err))
			return nil, errors
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint + fmt.Sprintf("?s=%s", placementHash),
			Body:    body,
			Headers: headers,
		})
	}

	if 0 == len(reqs) {
		return nil, errors
	}

	return reqs, errors

}

func (a *SomoaudienceAdapter) MakeBids(bidReq *openrtb.BidRequest, unused *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, bidReq.Imp),
			})
		}
	}

	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			if imp.Banner != nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeBanner
			}
			return mediaType
		}
	}
	return mediaType
}

func validateImpression(imp *openrtb.Imp) (string, error) {

	if imp.Audio != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, Somoaudience doesn't support Audio", imp.ID),
		}
	}

	if 0 == len(imp.Ext) {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, extImpBidder is empty", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpSomoaudience{}
	err := json.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	return impExt.PlacementHash, nil
}

func NewSomoaudienceBidder(endpoint string) *SomoaudienceAdapter {
	return &SomoaudienceAdapter{
		endpoint: endpoint,
	}
}
