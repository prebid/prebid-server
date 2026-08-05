package viant

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	reqCopy := *request
	cleanImps := make([]openrtb2.Imp, 0, len(request.Imp))

	for i := range request.Imp {
		var impExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &impExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext for impression index %d. %s", i, err.Error()),
			})
			continue
		}

		var bidderExt openrtb_ext.ImpExtViant
		if err := jsonutil.Unmarshal(impExt.Bidder, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext.bidder for impression index %d. %s", i, err.Error()),
			})
			continue
		}

		if bidderExt.PublisherID == "" {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp.ext.bidder.publisherId is required for impression index %d", i),
			})
			continue
		}

		imp := request.Imp[i]
		imp.Ext = stripBidderExt(imp.Ext)
		cleanImps = append(cleanImps, imp)
	}

	if len(cleanImps) == 0 {
		return nil, append(errs, &errortypes.BadInput{
			Message: "no valid impressions in the bid request",
		})
	}

	reqCopy.Imp = cleanImps

	requestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
	}}, errs
}

// stripBidderExt removes the "bidder" and "prebid" keys from imp.ext,
// returning nil if nothing else remains. If imp.ext is not a JSON object
// (and therefore can't be unmarshalled into a map), it is returned unchanged
// so the original value is preserved rather than silently dropped.
func stripBidderExt(ext json.RawMessage) json.RawMessage {
	if ext == nil {
		return nil
	}

	var extMap map[string]json.RawMessage
	if err := jsonutil.Unmarshal(ext, &extMap); err != nil {
		return ext
	}

	delete(extMap, openrtb_ext.PrebidExtBidderKey)
	delete(extMap, openrtb_ext.PrebidExtKey)

	if len(extMap) == 0 {
		return nil
	}

	cleaned, err := json.Marshal(extMap)
	if err != nil {
		return nil
	}
	return cleaned
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("service unavailable: HTTP status %d", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}}
	}

	if len(response.Body) == 0 {
		return nil, nil
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("JSON parsing error: %s", err.Error()),
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidderResponse.Currency = bidResponse.Cur

	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d for bid %s", bid.MType, bid.ImpID),
		}
	}
}
