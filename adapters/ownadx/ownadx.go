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

func getRequestData(reqJSON []byte, url string) []*adapters.RequestData {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    reqJSON,
		Uri:     url,
		Headers: headers,
	}}
}

func (adapter *adapter) buildEndpointURL(params *openrtb_ext.ExtImpOwnAdx) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		Host:      params.Host,
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
	var ownAdxExt *openrtb_ext.ExtImpOwnAdx
	ownAdxExt, err := getImpressionExt(&(request.Imp[0]))
	if err != nil {
		return nil, []error{
			httpBadResponseError(fmt.Sprintf("Bidder extension not valid or can't be unmarshalled")),
		}
	}

	endPoint, err := adapter.buildEndpointURL(ownAdxExt)
	if err != nil {
		return nil, []error{err}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	return getRequestData(reqJSON, endPoint), nil
}
func httpBadResponseError(message string) error {
	return &errortypes.BadServerResponse{
		Message: message,
	}
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
			httpBadResponseError(fmt.Sprintf("Bad request: %d", response.StatusCode)),
		}
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{
			httpBadResponseError(fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)),
		}
	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{
			httpBadResponseError(fmt.Sprintf("Bad server response ")),
		}
	}
	if len(bidResp.SeatBid) == 0 {
		return nil, []error{
			httpBadResponseError(fmt.Sprintf("Array SeatBid cannot be empty ")),
		}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	if len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, []error{
			httpBadResponseError(fmt.Sprintf("Bid cannot be empty ")),
		}
	}

	bid := bidResp.SeatBid[0].Bid[0]
	var bidExt bidExt
	var bidType openrtb_ext.BidType

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
