package displayio

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"net/http"
	"text/template"
)

type adapter struct {
	endpoint *template.Template
}

type reqDioExt struct {
	UserSession string `json:"userSession,omitempty"`
	PlacementId string `json:"placementId"`
	InventoryId string `json:"inventoryId"`
}

func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	var requestExt map[string]interface{}
	var dioExt reqDioExt

	impressions := request.Imp
	result := make([]*adapters.RequestData, 0, len(impressions))
	errs := make([]error, 0, len(impressions))

	for _, impression := range impressions {
		if impression.BidFloor == 0 {
			errs = append(errs, &errortypes.BadInput{
				Message: "BidFloor should be defined",
			})
			continue
		}

		if impression.BidFloorCur == "" {
			impression.BidFloorCur = "USD"
		}

		if impression.BidFloorCur != "USD" {
			convertedValue, err := requestInfo.ConvertCurrency(impression.BidFloor, impression.BidFloorCur, "USD")

			if err != nil {
				errs = append(errs, err)
				continue
			}

			impression.BidFloorCur = "USD"
			impression.BidFloor = convertedValue
		}

		if len(impression.Ext) == 0 {
			errs = append(errs, errors.New("impression extensions required"))
			continue
		}

		var bidderExt adapters.ExtImpBidder
		err := json.Unmarshal(impression.Ext, &bidderExt)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		var impressionExt openrtb_ext.ExtImpDisplayio
		err = json.Unmarshal(bidderExt.Bidder, &impressionExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		dioExt = reqDioExt{PlacementId: impressionExt.PlacementId, InventoryId: impressionExt.InventoryId}

		err = json.Unmarshal(request.Ext, &requestExt)
		if err != nil {
			requestExt = make(map[string]interface{})
		}

		requestExt["displayio"] = dioExt

		request.Ext, err = json.Marshal(requestExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		request.Imp = []openrtb2.Imp{impression}
		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		url, err := adapter.buildEndpointURL(&impressionExt)
		if err != nil {
			return nil, []error{err}
		}

		result = append(result, &adapters.RequestData{
			Method:  "POST",
			Uri:     url,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	request.Imp = impressions

	if len(result) == 0 {
		return nil, errs
	}
	return result, errs
}

// MakeBids translates Displayio bid response to prebid-server specific format
func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {
		msg := fmt.Sprintf("Bad server response: %d", err)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	if len(bidResp.SeatBid) != 1 {
		msg := fmt.Sprintf("Invalid SeatBids count: %d", len(bidResp.SeatBid))
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponse()

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getBidMediaTypeFromMtype(&sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}

	return bidResponse, errs
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: endpoint,
	}
	return bidder, nil
}

func getBidMediaTypeFromMtype(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unexpected media type for bid: %s", bid.ImpID)
	}
}

func (adapter *adapter) buildEndpointURL(params *openrtb_ext.ExtImpDisplayio) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: params.PublisherId}
	return macros.ResolveMacros(adapter.endpoint, endpointParams)
}
