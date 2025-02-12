package trafficgate

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

type adapter struct {
	EndpointTemplate *template.Template
}

type BidResponseExt struct {
	Prebid struct {
		Type string
	}
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	// Pull the host and source ID info from the bidder params.
	reqImps, err := splitImpressions(request.Imp)
	if err != nil {
		return nil, []error{err}
	}

	requests := []*adapters.RequestData{}

	var errs []error
	for reqExt, reqImp := range reqImps {
		request.Imp = reqImp
		reqJson, err := json.Marshal(request)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		urlParams := macros.EndpointTemplateParams{Host: reqExt.Host}
		url, err := macros.ResolveMacros(a.EndpointTemplate, urlParams)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		request := adapters.RequestData{
			Method:  "POST",
			Uri:     url,
			Body:    reqJson,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp)}

		requests = append(requests, &request)
	}

	return requests, errs
}

/*
internal original request in OpenRTB, external = result of us having converted it (what comes out of MakeRequests)
*/
func (a *adapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error response with status %d", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	bidResponse.Currency = bidResp.Cur

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			var bidExt BidResponseExt
			if err := jsonutil.Unmarshal(seatBid.Bid[i].Ext, &bidExt); err != nil {
				return nil, []error{&errortypes.BadServerResponse{
					Message: "Missing response ext",
				}}
			}
			if len(bidExt.Prebid.Type) < 1 {
				return nil, []error{&errortypes.BadServerResponse{
					Message: "Unable to read bid.ext.prebid.type",
				}}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForImp(bidExt.Prebid.Type),
			})
		}
	}

	return bidResponse, nil
}

func splitImpressions(imps []openrtb2.Imp) (map[openrtb_ext.ExtImpTrafficGate][]openrtb2.Imp, error) {

	var multipleImps = make(map[openrtb_ext.ExtImpTrafficGate][]openrtb2.Imp)

	for _, imp := range imps {
		bidderParams, err := getBidderParams(&imp)
		if err != nil {
			return nil, err
		}

		multipleImps[*bidderParams] = append(multipleImps[*bidderParams], imp)
	}

	return multipleImps, nil
}

func getBidderParams(imp *openrtb2.Imp) (*openrtb_ext.ExtImpTrafficGate, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Missing bidder ext",
		}
	}
	var TrafficGateExt openrtb_ext.ExtImpTrafficGate
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &TrafficGateExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder parameters required",
		}
	}

	return &TrafficGateExt, nil
}

func getMediaTypeForImp(bidType string) openrtb_ext.BidType {
	switch bidType {
	case "video":
		return openrtb_ext.BidTypeVideo
	case "native":
		return openrtb_ext.BidTypeNative
	case "audio":
		return openrtb_ext.BidTypeAudio
	}
	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of the TrafficGate adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		EndpointTemplate: template,
	}
	return bidder, nil
}
