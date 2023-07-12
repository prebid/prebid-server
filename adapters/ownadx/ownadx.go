package ownadx

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"text/template"
)

type adapter struct {
	endpoint *template.Template
}
type bidExt struct {
	MediaType string `json:"mediaType"`
}

func (adapter *adapter) getRequestData(bidRequest *openrtb2.BidRequest, impExt *openrtb_ext.ExtImpOwnAdx, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	pbidRequest := createBidRequest(bidRequest, imps)
	reqJSON, err := json.Marshal(pbidRequest)
	if err != nil {
		return nil, err
	}
	adapter.buildEndpointURL(impExt)
	url, err := adapter.buildEndpointURL(impExt)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: headers}, nil

}
func createBidRequest(rtbBidRequest *openrtb2.BidRequest, imps []openrtb2.Imp) *openrtb2.BidRequest {
	bidRequest := *rtbBidRequest
	bidRequest.Imp = imps
	return &bidRequest
}
func getExtImps(imps []openrtb2.Imp, impsExt []openrtb_ext.ExtImpOwnAdx) map[openrtb_ext.ExtImpOwnAdx][]openrtb2.Imp {
	respExt := make(map[openrtb_ext.ExtImpOwnAdx][]openrtb2.Imp)
	for idx := range imps {
		imp := imps[idx]
		impExt := impsExt[idx]
		if respExt[impExt] == nil {
			respExt[impExt] = make([]openrtb2.Imp, 0)
		}
		respExt[impExt] = append(respExt[impExt], imp)
	}
	return respExt
}
func (adapter *adapter) buildEndpointURL(params *openrtb_ext.ExtImpOwnAdx) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		ZoneID:    params.SspId,
		AccountID: params.SeatId,
		SourceId:  params.TokenId,
	}
	return macros.ResolveMacros(adapter.endpoint, endpointParams)
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpOwnAdx, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not valid or can't be unmarshalled",
		}
	}

	var ownAdxExt openrtb_ext.ExtImpOwnAdx
	if err := json.Unmarshal(bidderExt.Bidder, &ownAdxExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}

	return &ownAdxExt, nil
}

func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{
			Message: "No impression in the bid request"},
		)
		return nil, errs
	}

	imps, impExts, err := getImpressionsAndImpExt(request.Imp)
	if len(imps) == 0 {
		return nil, err
	}
	errs = append(errs, err...)
	if len(imps) == 0 {
		return nil, err
	}

	extImps := getExtImps(imps, impExts)

	reqDetail := make([]*adapters.RequestData, 0, len(extImps))
	for k, imps := range extImps {
		bidRequest, err := adapter.getRequestData(request, &k, imps)
		if err != nil {
			errs = append(errs, err)
		} else {
			reqDetail = append(reqDetail, bidRequest)
		}
	}
	return reqDetail, errs
}

func getImpressionsAndImpExt(imps []openrtb2.Imp) ([]openrtb2.Imp, []openrtb_ext.ExtImpOwnAdx, []error) {
	impLen := len(imps)
	errors := make([]error, 0, impLen)
	rsImpExts := make([]openrtb_ext.ExtImpOwnAdx, 0, impLen)
	rsImps := make([]openrtb2.Imp, 0, impLen)

	for _, imp := range imps {
		ownAdxExt, err := getImpressionExt(&(imp))
		if err != nil {
			errors = append(errors, err)
			continue
		}

		rsImps = append(rsImps, imp)
		rsImpExts = append(rsImpExts, *ownAdxExt)
	}
	return rsImps, rsImpExts, errors
}

func getBidType(ext bidExt) (openrtb_ext.BidType, error) {
	return openrtb_ext.ParseBidType(ext.MediaType)
}

func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Bad request: %d", response.StatusCode),
			},
		}
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unexpected status code: %d. Run with request.test = 1 for more info.", response.StatusCode),
			},
		}
	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Bad server response "),
			},
		}
	}
	if len(bidResp.SeatBid) == 0 {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Array SeatBid cannot be empty "),
			},
		}
	}

	seatBid := bidResp.SeatBid[0]
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	if len(seatBid.Bid) == 0 {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Bid cannot be empty "),
			},
		}
	}
	for i := 0; i < len(seatBid.Bid); i++ {
		var bidExt bidExt
		var bidType openrtb_ext.BidType
		bid := seatBid.Bid[i]
		if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: "BidExt is required",
			}}
		}

		bidType, err := getBidType(bidExt)

		if err != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: "Bid type is invalid",
			}}
		}
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}

	return bidResponse, nil
}

// Builder builds a new instance of the OwnAdx adapter for the given bidder with the given config
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
