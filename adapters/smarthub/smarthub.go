package smarthub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	ADAPTER_VER = "1.0.0"
)

type adapter struct {
	endpoint *template.Template
}

type bidExt struct {
	MediaType string `json:"mediaType"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}

	return bidder, nil
}

func (a *adapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtSmartHub, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided or can't be unmarshalled",
		}
	}

	var smarthubExt openrtb_ext.ExtSmartHub
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &smarthubExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}

	return &smarthubExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtSmartHub) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AccountID: params.Seat,
		SourceId:  params.Token,
		Host:      params.PartnerName,
	}

	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	var smarthubExt *openrtb_ext.ExtSmartHub
	smarthubExt, err := a.getImpressionExt(&(openRTBRequest.Imp[0]))
	if err != nil {
		return nil, []error{err}
	}

	url, err := a.buildEndpointURL(smarthubExt)
	if err != nil {
		return nil, []error{err}
	}

	reqJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("Prebid-Adapter-Ver", ADAPTER_VER)

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    reqJSON,
		Uri:     url,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}}, nil
}

func (a *adapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	*adapters.BidderResponse,
	[]error,
) {
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
	if err := jsonutil.Unmarshal(responseBody, &bidResp); err != nil {
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

	bid := bids[0]

	var bidExt bidExt
	var bidType openrtb_ext.BidType

	if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Field BidExt is required",
		}}
	}

	bidType, err := getBidType(bidExt)

	if err != nil {
		return nil, []error{err}
	}

	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     &bid,
		BidType: bidType,
	})
	return bidResponse, nil
}

func getBidType(ext bidExt) (openrtb_ext.BidType, error) {
	return openrtb_ext.ParseBidType(ext.MediaType)
}
