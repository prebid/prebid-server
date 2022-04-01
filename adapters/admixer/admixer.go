package admixer

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

type AdmixerAdapter struct {
	endpoint string
}

// Builder builds a new instance of the Admixer adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &AdmixerAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

type admixerImpExt struct {
	CustomParams map[string]interface{} `json:"customParams"`
}

func (a *AdmixerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (requests []*adapters.RequestData, errors []error) {
	rq, errs := a.makeRequest(request)

	if len(errs) > 0 {
		errors = append(errors, errs...)
		return
	}

	if rq != nil {
		requests = append(requests, rq)
	}

	return
}

func (a *AdmixerAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb2.Imp

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impressions in request",
		}}
	}

	for _, imp := range request.Imp {
		if err := preprocess(&imp); err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

func preprocess(imp *openrtb2.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var admixerExt openrtb_ext.ExtImpAdmixer
	if err := json.Unmarshal(bidderExt.Bidder, &admixerExt); err != nil {
		return &errortypes.BadInput{
			Message: "Wrong Admixer bidder ext",
		}
	}

	//don't use regexp due to possible performance reduce
	if len(admixerExt.ZoneId) < 32 || len(admixerExt.ZoneId) > 36 {
		return &errortypes.BadInput{
			Message: "ZoneId must be UUID/GUID",
		}
	}

	imp.TagID = admixerExt.ZoneId

	if imp.BidFloor == 0 && admixerExt.CustomBidFloor > 0 {
		imp.BidFloor = admixerExt.CustomBidFloor
	}

	imp.Ext = nil

	if admixerExt.CustomParams != nil {
		impExt := admixerImpExt{
			CustomParams: admixerExt.CustomParams,
		}
		var err error
		if imp.Ext, err = json.Marshal(impExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	return nil
}

func (a *AdmixerAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Dsp server internal error", response.StatusCode),
		}}
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Bad request to dsp", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	//additional no content check
	if len(bidResp.SeatBid) == 0 || len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}
