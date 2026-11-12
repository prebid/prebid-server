package jjtech

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

// Builder builds a new instance of the JJTech adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

// MakeRequests batches every valid impression into a single call to JJTech's bidding server,
// swapping the Prebid Server imp.ext.bidder params for JJTech's own imp.ext.jjtech shape.
// Every other key Prebid Server core supplies on imp.ext (gpid, tid, data, skadn, ae and the
// sanitized prebid object) is preserved and forwarded untouched.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	imps := make([]openrtb2.Imp, 0, len(request.Imp))

	for _, imp := range request.Imp {
		params, impExt, err := parseImpParams(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		paramsJSON, err := jsonutil.Marshal(params)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// impExt is freshly unmarshaled from imp.Ext, so rewriting it cannot reach
		// the caller's impression.
		delete(impExt, "bidder")
		impExt["jjtech"] = paramsJSON

		ext, err := jsonutil.Marshal(impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// imp is a copy, so this leaves the caller's request untouched.
		imp.Ext = ext
		imps = append(imps, imp)
	}

	if len(imps) == 0 {
		return nil, errs
	}

	outgoing := *request
	outgoing.Imp = imps

	body, err := jsonutil.Marshal(outgoing)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps),
	}}, errs
}

// parseImpParams validates the JJTech publisher params on an impression and returns them
// alongside the impression's full ext, decoded key by key so callers can rewrite only the
// bidder entry and keep everything else Prebid Server core supplied.
func parseImpParams(imp openrtb2.Imp) (openrtb_ext.ExtImpJJTech, map[string]json.RawMessage, error) {
	var impExt map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
		return openrtb_ext.ExtImpJJTech{}, nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse imp.ext: %s", imp.ID, err.Error()),
		}
	}

	var params openrtb_ext.ExtImpJJTech
	if err := jsonutil.Unmarshal(impExt["bidder"], &params); err != nil {
		return openrtb_ext.ExtImpJJTech{}, nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse jjtech params: %s", imp.ID, err.Error()),
		}
	}

	if params.PlacementID == "" {
		return openrtb_ext.ExtImpJJTech{}, nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: placementId is required", imp.ID),
		}
	}

	return params, impExt, nil
}

// MakeBids converts JJTech's OpenRTB response into Prebid Server bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("failed to parse bid response: %s", err.Error()),
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidderResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
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

// getMediaTypeForBid resolves the bid's media type from ORTB 2.6 mtype. JJTech is
// banner-only, so any other markup type is a bad response.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.MType == openrtb2.MarkupBanner {
		return openrtb_ext.BidTypeBanner, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ID),
	}
}
