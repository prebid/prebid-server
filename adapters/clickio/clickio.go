package clickio

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

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No Imps in Bid Request"}}
	}

	requestCopy := *request
	requestCopy.Imp = append([]openrtb2.Imp(nil), request.Imp...)
	for i := range requestCopy.Imp {
		updateImpExtWithParams(&requestCopy.Imp[i])
	}

	body, err := json.Marshal(&requestCopy)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}}, nil
}

func updateImpExtWithParams(imp *openrtb2.Imp) {
	if len(imp.Ext) == 0 {
		return
	}

	var impExt map[string]json.RawMessage
	if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
		return
	}

	params := map[string]json.RawMessage{}
	if rawParams, exists := impExt["params"]; exists && len(rawParams) > 0 {
		_ = json.Unmarshal(rawParams, &params)
	}

	if said, ok := extractParamFromPrebid(impExt, "said"); ok {
		params["said"] = said
	}
	if psid, ok := extractParamFromPrebid(impExt, "psid"); ok {
		params["psid"] = psid
	}
	if template, ok := extractParamFromPrebid(impExt, "template"); ok {
		params["template"] = template
	}
	if len(params) == 0 {
		return
	}

	rawParams, err := json.Marshal(params)
	if err != nil {
		return
	}
	impExt["params"] = rawParams

	rawExt, err := json.Marshal(impExt)
	if err != nil {
		return
	}
	imp.Ext = rawExt
}

func extractParamFromPrebid(impExt map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	// Preferred PBS adapter shape: imp.ext.bidder contains this adapter params.
	if rawBidder, ok := impExt["bidder"]; ok {
		if value, ok := extractParamFromBidderExt(rawBidder, key); ok {
			return value, true
		}
	}

	rawPrebid, ok := impExt["prebid"]
	if !ok || len(rawPrebid) == 0 {
		return nil, false
	}

	var prebid map[string]json.RawMessage
	if err := json.Unmarshal(rawPrebid, &prebid); err != nil {
		return nil, false
	}

	rawBidder, ok := prebid["bidder"]
	if !ok || len(rawBidder) == 0 {
		return nil, false
	}

	var bidderMap map[string]json.RawMessage
	if err := json.Unmarshal(rawBidder, &bidderMap); err != nil {
		return nil, false
	}

	if value, ok := extractParamFromBidderExt(bidderMap["clickio"], key); ok {
		return value, true
	}

	return nil, false
}

func extractParamFromBidderExt(raw json.RawMessage, key string) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var bidderExt map[string]json.RawMessage
	if err := json.Unmarshal(raw, &bidderExt); err != nil {
		return nil, false
	}
	value, ok := bidderExt[key]
	if !ok || len(value) == 0 {
		return nil, false
	}
	return value, true
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID != impID {
			continue
		}
		switch {
		case imp.Banner != nil:
			return openrtb_ext.BidTypeBanner, nil
		case imp.Video != nil:
			return openrtb_ext.BidTypeVideo, nil
		case imp.Native != nil:
			return openrtb_ext.BidTypeNative, nil
		case imp.Audio != nil:
			return openrtb_ext.BidTypeAudio, nil
		default:
			return "", &errortypes.BadInput{Message: fmt.Sprintf("Failed to resolve media type for impression \"%s\"", impID)}
		}
	}
	return "", &errortypes.BadInput{Message: fmt.Sprintf("Failed to find impression \"%s\"", impID)}
}
