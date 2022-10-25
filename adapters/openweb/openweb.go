package openweb

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

type openwebImpExt struct {
	OpenWeb openrtb_ext.ExtImpOpenWeb `json:"openweb"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	totalImps := len(request.Imp)
	errors := make([]error, 0, totalImps)
	sourceIdToImpIds := make(map[int][]int)
	var sourceIds []int

	for i := 0; i < totalImps; i++ {

		sourceId, err := validateImpression(&request.Imp[i])

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if _, ok := sourceIdToImpIds[sourceId]; !ok {
			sourceIdToImpIds[sourceId] = make([]int, 0, totalImps-i)
			sourceIds = append(sourceIds, sourceId)
		}

		sourceIdToImpIds[sourceId] = append(sourceIdToImpIds[sourceId], i)

	}

	totalReqs := len(sourceIdToImpIds)
	if totalReqs == 0 {
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	reqs := make([]*adapters.RequestData, 0, totalReqs)

	imps := request.Imp
	reqCopy := *request
	reqCopy.Imp = make([]openrtb2.Imp, totalImps)
	for _, sourceId := range sourceIds {
		impIds := sourceIdToImpIds[sourceId]
		reqCopy.Imp = reqCopy.Imp[:0]

		for i := 0; i < len(impIds); i++ {
			reqCopy.Imp = append(reqCopy.Imp, imps[impIds[i]])
		}

		body, err := json.Marshal(reqCopy)
		if err != nil {
			errors = append(errors, fmt.Errorf("error while encoding bidRequest, err: %s", err))
			return nil, errors
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint + fmt.Sprintf("?aid=%d", sourceId),
			Body:    body,
			Headers: headers,
		})
	}

	return reqs, errors

}

func (a *adapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpRes.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Remote server error: %s", httpRes.Body),
		}}
	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(httpRes.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("error while decoding response, err: %s", err),
		}}
	}

	bidResponse := adapters.NewBidderResponse()
	var errors []error

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {

			bid := sb.Bid[i]

			mediaType, impOK := getBidType(bidReq.Imp, bid.ImpID)
			if !impOK {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("ignoring bid id=%s, request doesn't contain any impression with id=%s", bid.ID, bid.ImpID),
				})
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mediaType,
			})
		}
	}

	return bidResponse, errors
}

func getBidType(imps []openrtb2.Imp, impId string) (mediaType openrtb_ext.BidType, ok bool) {
	mediaType = openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			ok = true

			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}

			break
		}
	}

	return
}

func validateImpression(imp *openrtb2.Imp) (int, error) {
	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpOpenWeb{}
	err := json.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	var impExtBuffer []byte

	impExtBuffer, err = json.Marshal(&openwebImpExt{
		OpenWeb: impExt,
	})
	if err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while encoding impExt, err: %s", imp.ID, err),
		}
	}

	if impExt.BidFloor > 0 {
		imp.BidFloor = impExt.BidFloor
	}

	imp.Ext = impExtBuffer

	return impExt.SourceID, nil
}

// Builder builds a new instance of the OpenWeb adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
