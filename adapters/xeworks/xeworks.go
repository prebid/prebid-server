package xeworks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)

	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := new(adapter)
	bidder.endpoint = template

	return bidder, nil
}

func (a *adapter) buildEndpointFromRequest(bidRequest *openrtb2.BidRequest) (string, error) {
	impExtRaw := bidRequest.Imp[0].Ext
	var impExt adapters.ExtImpBidder

	if err := json.Unmarshal(impExtRaw, &impExt); err != nil {
		return "", &errortypes.BadInput{
			Message: "Bidder impression extension can't be deserialized",
		}
	}

	var xeworksExt openrtb_ext.ExtXeworks
	if err := json.Unmarshal(impExt.Bidder, &xeworksExt); err != nil {
		return "", &errortypes.BadInput{
			Message: "Xeworks extenson can't be deserialized",
		}
	}

	endpointParams := macros.EndpointTemplateParams{
		Host:     xeworksExt.Env,
		SourceId: xeworksExt.Pid,
	}

	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	requestInfo *adapters.ExtraRequestInfo,
) ([]*adapters.RequestData, []error) {
	endpoint, err := a.buildEndpointFromRequest(openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	body, err := json.Marshal(openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    body,
		Uri:     endpoint,
		Headers: headers,
	}}, nil
}

func (a *adapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if bidderRawResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if bidderRawResponse.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad Request. %s", string(bidderRawResponse.Body)),
		}}
	}

	if bidderRawResponse.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadInput{
			Message: "Bidder unavailable. Please contact the bidder support.",
		}}
	}

	if bidderRawResponse.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Status Code: [ %d ] %s", bidderRawResponse.StatusCode, string(bidderRawResponse.Body)),
		}}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Array SeatBid cannot be empty",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	bids := bidResp.SeatBid[0].Bid

	if len(bids) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Array SeatBid[0].Bid cannot be empty",
		}}
	}

	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     &bids[0],
		BidType: getMediaTypeForImp(bids[0].ImpID, openRTBRequest.Imp),
	})
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
