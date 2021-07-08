package logicad

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

type LogicadAdapter struct {
	endpoint string
}

func (adapter *LogicadAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No impression in the bid request"}}
	}

	pub2impressions, imps, errs := getImpressionsInfo(request.Imp)
	if len(pub2impressions) == 0 || len(imps) == 0 {
		return nil, errs
	}

	result := make([]*adapters.RequestData, 0, len(pub2impressions))
	for k, imps := range pub2impressions {
		bidRequest, err := adapter.buildAdapterRequest(request, &k, imps)
		if err != nil {
			errs = append(errs, err)
		} else {
			result = append(result, bidRequest)
		}
	}
	return result, errs
}

func getImpressionsInfo(imps []openrtb2.Imp) (map[openrtb_ext.ExtImpLogicad][]openrtb2.Imp, []openrtb2.Imp, []error) {
	errors := make([]error, 0, len(imps))
	resImps := make([]openrtb2.Imp, 0, len(imps))
	res := make(map[openrtb_ext.ExtImpLogicad][]openrtb2.Imp)

	for _, imp := range imps {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if err := validateImpression(&impExt); err != nil {
			errors = append(errors, err)
			continue
		}

		if res[impExt] == nil {
			res[impExt] = make([]openrtb2.Imp, 0)
		}
		res[impExt] = append(res[impExt], imp)
		resImps = append(resImps, imp)
	}
	return res, resImps, errors
}

func validateImpression(impExt *openrtb_ext.ExtImpLogicad) error {
	if impExt.Tid == "" {
		return &errortypes.BadInput{Message: "No tid value provided"}
	}
	return nil
}

func getImpressionExt(imp *openrtb2.Imp) (openrtb_ext.ExtImpLogicad, error) {
	var bidderExt adapters.ExtImpBidder
	var logicadExt openrtb_ext.ExtImpLogicad

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return logicadExt, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	if err := json.Unmarshal(bidderExt.Bidder, &logicadExt); err != nil {
		return logicadExt, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	return logicadExt, nil
}

func (adapter *LogicadAdapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpLogicad, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	newBidRequest := createBidRequest(prebidBidRequest, params, imps)
	reqJSON, err := json.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    reqJSON,
		Headers: headers}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpLogicad, imps []openrtb2.Imp) *openrtb2.BidRequest {
	bidRequest := *prebidBidRequest
	bidRequest.Imp = imps
	for idx := range bidRequest.Imp {
		imp := &bidRequest.Imp[idx]
		imp.TagID = params.Tid
		imp.Ext = nil
	}
	return &bidRequest
}

//MakeBids translates Logicad bid response to prebid-server specific format
func (adapter *LogicadAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}

	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		msg := fmt.Sprintf("Bad server response: %d", err)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}
	if len(bidResp.SeatBid) != 1 {
		msg := fmt.Sprintf("Invalid SeatBids count: %d", len(bidResp.SeatBid))
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	seatBid := bidResp.SeatBid[0]
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(seatBid.Bid))

	for i := 0; i < len(seatBid.Bid); i++ {
		bid := seatBid.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: openrtb_ext.BidTypeBanner,
		})
	}
	return bidResponse, nil
}

// Builder builds a new instance of the Logicad adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &LogicadAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
