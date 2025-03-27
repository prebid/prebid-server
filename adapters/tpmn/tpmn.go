package tpmn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
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
		validImps = append(validImps, imp)
	}
	return validImps, errs
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
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{fmt.Errorf("bid response unmarshal: %v", err)}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(bid)
			if err != nil {
				return nil, []error{err}
			}
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unsupported MType %d", bid.MType)
	}
}

// Builder builds a new instance of the TpmnBidder adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		uri: config.Endpoint,
	}
	return bidder, nil
}
