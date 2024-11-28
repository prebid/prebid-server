package adverxo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type adapter struct {
	endpointTemplate *template.Template
}

// Builder builds a new instance of the Adverxo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	urlTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)

	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpointTemplate: urlTemplate,
	}

	return bidder, nil
}

func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var (
		result []*adapters.RequestData
		errors []error
	)

	for i := range request.Imp {
		imp := request.Imp[i]

		adUnitParams, err := getAdUnitsParams(imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		endpointUrl, err := adapter.buildEndpointURL(adUnitParams)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = modifyImp(&imp, requestInfo)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		thisRequest := makeRequestCopyWithImp(request, imp)
		thisRequestBody, err := json.Marshal(thisRequest)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		result = append(result, &adapters.RequestData{
			Method: "POST",
			Uri:    endpointUrl,
			Body:   thisRequestBody,
			ImpIDs: []string{imp.ID},
		})
	}

	return result, errors
}

func (adapter *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid
			bidType, err := getMediaTypeForBid(&bid)
			if err != nil {
				return nil, []error{err}
			}

			// for native bid responses fix Adm field
			if bidType == openrtb_ext.BidTypeNative {
				bid.AdM, err = getNativeAdm(bid.AdM)
				if err != nil {
					return nil, []error{err}
				}
			}

			resolveMacros(&bid)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, nil
}

func getAdUnitsParams(imp openrtb2.Imp) (*openrtb_ext.ImpExtAdverxo, error) {
	var ext adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext", imp.ID),
		}
	}

	var adverxoExt openrtb_ext.ImpExtAdverxo
	if err := json.Unmarshal(ext.Bidder, &adverxoExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext.bidder: %v", imp.ID, err),
		}
	}

	return &adverxoExt, nil
}

func modifyImp(imp *openrtb2.Imp, requestInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
		// Convert to US dollars
		convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return err
		}

		// Update after conversion. All imp elements inside request.Imp are shallow copies
		// therefore, their non-pointer values are not shared memory and are safe to modify.
		imp.BidFloorCur = "USD"
		imp.BidFloor = convertedValue
	}

	return nil
}

func makeRequestCopyWithImp(request *openrtb2.BidRequest, imp openrtb2.Imp) openrtb2.BidRequest {
	requestCopy := *request
	requestCopy.Imp = []openrtb2.Imp{imp}

	return requestCopy
}

func (adapter *adapter) buildEndpointURL(params *openrtb_ext.ImpExtAdverxo) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AdUnit:  strconv.Itoa(params.AdUnitId),
		TokenID: params.Auth,
	}

	return macros.ResolveMacros(adapter.endpointTemplate, endpointParams)
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}

func getNativeAdm(adm string) (string, error) {
	nativeAdm := make(map[string]interface{})
	err := json.Unmarshal([]byte(adm), &nativeAdm)
	if err != nil {
		return adm, errors.New("unable to unmarshal native adm")
	}

	// move bid.adm.native to bid.adm
	if _, ok := nativeAdm["native"]; ok {
		//using jsonparser to avoid marshaling, encode escape, etc.
		value, dataType, _, err := jsonparser.Get([]byte(adm), string(openrtb_ext.BidTypeNative))
		if err != nil || dataType != jsonparser.Object {
			return adm, errors.New("unable to get native adm")
		}
		adm = string(value)
	}

	return adm, nil
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid != nil {
		price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
		bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
	}
}
