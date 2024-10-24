package tradplus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the tradplus adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	return &adapter{
		endpoint: template,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	adapterRequest, errs := a.makeRequest(request)
	if errs != nil {
		return nil, errs
	}
	return []*adapters.RequestData{adapterRequest}, nil
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {

	tradplusExt, err := getImpressionExt(&request.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	request.Imp[0].Ext = nil

	url, err := a.buildEndpointURL(tradplusExt)
	if err != nil {
		return nil, []error{err}
	}

	err = transform(request)
	if err != nil {
		return nil, []error{err}
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqBody,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpTradPlus, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing tradplusExt - " + err.Error(),
		}
	}

	var tradplusExt openrtb_ext.ExtImpTradPlus
	if err := json.Unmarshal(bidderExt.Bidder, &tradplusExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing bidderExt - " + err.Error(),
		}
	}

	if tradplusExt.AccountID == "" {
		return nil, &errortypes.BadInput{
			Message: "imp.ext.accountId required",
		}
	}

	return &tradplusExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpTradPlus) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AccountID: params.AccountID,
		ZoneID:    params.ZoneID,
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func transform(request *openrtb2.BidRequest) error {
	for i, imp := range request.Imp {
		if imp.Native != nil {
			var nativeRequest map[string]interface{}
			nativeCopyRequest := make(map[string]interface{})
			if err := json.Unmarshal([]byte(request.Imp[i].Native.Request), &nativeRequest); err != nil {
				return err
			}
			_, exists := nativeRequest["native"]
			if exists {
				continue
			}
			nativeCopyRequest["native"] = nativeRequest
			nativeReqByte, err := json.Marshal(nativeCopyRequest)
			if err != nil {
				return err
			}
			nativeCopy := *request.Imp[i].Native
			nativeCopy.Request = string(nativeReqByte)
			request.Imp[i].Native = &nativeCopy
		}
	}
	return nil
}

// MakeBids make the bids for the bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			mediaType, err := getMediaTypeForBid(sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}
	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unrecognized bid type in response from tradplus for bid %s", bid.ImpID)
	}
}
