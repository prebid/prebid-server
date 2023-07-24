package tpmn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// TpmnAdapter struct
type adapter struct {
	uri string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids from TpmnBidder.
func (rcv *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	validImps, errs := getValidImpressions(request, reqInfo)
	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     rcv.uri,
		Body:    requestBodyJSON,
		Headers: headers,
	}}, errs
}

// getValidImpressions validate imps and check for bid floor currency. Convert to EUR if necessary
func getValidImpressions(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]openrtb2.Imp, []error) {
	var errs []error
	var validImps []openrtb2.Imp

	for _, imp := range request.Imp {
		if err := preprocessBidFloorCurrency(&imp, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := preprocessExtensions(&imp); err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
	}
	return validImps, errs
}

func preprocessExtensions(imp *openrtb2.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	return nil
}

func preprocessBidFloorCurrency(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	// we expect every currency related data to be EUR
	if imp.BidFloor > 0 && strings.ToUpper(imp.BidFloorCur) != "USD" && imp.BidFloorCur != "" {
		if convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD"); err != nil {
			return err
		} else {
			imp.BidFloor = convertedValue
		}
	}
	imp.BidFloorCur = "USD"
	return nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		// no bid response
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Invalid Status Returned: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{fmt.Errorf("bid response unmarshal: %v", err)}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType := getMediaTypeForImp(seatBid.Bid[i].ImpID, request.Imp)
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			break
		}
	}
	return mediaType
}

// Builder builds a new instance of the TpmnBidder adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		uri: config.Endpoint,
	}
	return bidder, nil
}
