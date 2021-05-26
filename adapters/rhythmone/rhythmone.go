package rhythmone

import (
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type RhythmoneAdapter struct {
	endPoint string
}

func (a *RhythmoneAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var uri string
	request, uri, errs = a.preProcess(request, errs)
	if request != nil {
		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		if uri != "" {
			headers := http.Header{}
			headers.Add("Content-Type", "application/json;charset=utf-8")
			headers.Add("Accept", "application/json")
			return []*adapters.RequestData{{
				Method:  "POST",
				Uri:     uri,
				Body:    reqJSON,
				Headers: headers,
			}}, errs
		}
	}
	return nil, errs
}

func (a *RhythmoneAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

// Builder builds a new instance of the Rythomone adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &RhythmoneAdapter{
		endPoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *RhythmoneAdapter) preProcess(req *openrtb2.BidRequest, errors []error) (*openrtb2.BidRequest, string, []error) {
	numRequests := len(req.Imp)
	var uri string = ""
	for i := 0; i < numRequests; i++ {
		imp := req.Imp[i]
		var bidderExt adapters.ExtImpBidder
		err := json.Unmarshal(imp.Ext, &bidderExt)
		if err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("ext data not provided in imp id=%s. Abort all Request", imp.ID),
			}
			errors = append(errors, err)
			return nil, "", errors
		}
		var rhythmoneExt openrtb_ext.ExtImpRhythmone
		err = json.Unmarshal(bidderExt.Bidder, &rhythmoneExt)
		if err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("placementId | zone | path not provided in imp id=%s. Abort all Request", imp.ID),
			}
			errors = append(errors, err)
			return nil, "", errors
		}
		rhythmoneExt.S2S = true
		rhythmoneExtCopy, err := json.Marshal(&rhythmoneExt)
		if err != nil {
			errors = append(errors, err)
			return nil, "", errors
		}
		bidderExtCopy := struct {
			Bidder json.RawMessage `json:"bidder,omitempty"`
		}{rhythmoneExtCopy}
		impExtCopy, err := json.Marshal(&bidderExtCopy)
		if err != nil {
			errors = append(errors, err)
			return nil, "", errors
		}
		imp.Ext = impExtCopy
		req.Imp[i] = imp
		if uri == "" {
			uri = fmt.Sprintf("%s/%s/0/%s?z=%s&s2s=%s", a.endPoint, rhythmoneExt.PlacementId, rhythmoneExt.Path, rhythmoneExt.Zone, "true")
		}
	}
	return req, uri, errors
}
