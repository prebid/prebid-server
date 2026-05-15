package ferio

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

const bidderExtKey = "bidder"

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	outgoingRequest := copyRequest(request)
	outgoingRequest.Imp = make([]openrtb2.Imp, 0, len(request.Imp))

	var publisherID string
	var errs []error

	for _, imp := range request.Imp {
		params, err := parseImpExt(imp.Ext, imp.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if publisherID == "" {
			publisherID = params.PublisherID
		} else if publisherID != params.PublisherID {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("imp %s has publisherId %q, expected %q", imp.ID, params.PublisherID, publisherID),
			}}
		}

		outgoingImp := imp
		outgoingImp.TagID = params.AdUnitID

		outgoingImpExt, err := makeOutgoingImpExt(imp.Ext, params)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		outgoingImp.Ext = outgoingImpExt
		outgoingRequest.Imp = append(outgoingRequest.Imp, outgoingImp)
	}

	if len(outgoingRequest.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "found no valid impressions"})
		return nil, errs
	}

	setPublisherID(&outgoingRequest, publisherID)

	body, err := jsonutil.Marshal(outgoingRequest)
	if err != nil {
		return nil, append(errs, err)
	}

	return []*adapters.RequestData{{
		Method: http.MethodPost,
		Uri:    a.endpoint,
		Body:   body,
		Headers: http.Header{
			"Accept":       []string{"application/json"},
			"Content-Type": []string{"application/json;charset=utf-8"},
		},
		ImpIDs: openrtb_ext.GetImpIDs(outgoingRequest.Imp),
	}}, errs
}

func copyRequest(request *openrtb2.BidRequest) openrtb2.BidRequest {
	requestCopy := *request

	if request.Site != nil {
		siteCopy := *request.Site
		requestCopy.Site = &siteCopy
		if request.Site.Publisher != nil {
			publisherCopy := *request.Site.Publisher
			requestCopy.Site.Publisher = &publisherCopy
		}
	}

	if request.App != nil {
		appCopy := *request.App
		requestCopy.App = &appCopy
		if request.App.Publisher != nil {
			publisherCopy := *request.App.Publisher
			requestCopy.App.Publisher = &publisherCopy
		}
	}

	return requestCopy
}

func parseImpExt(impExt json.RawMessage, impID string) (*openrtb_ext.ExtImpFerio, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(impExt, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext: %v", impID, err),
		}
	}

	var params openrtb_ext.ExtImpFerio
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext.bidder: %v", impID, err),
		}
	}

	return &params, nil
}

func makeOutgoingImpExt(impExt json.RawMessage, params *openrtb_ext.ExtImpFerio) (json.RawMessage, error) {
	var extMap map[string]json.RawMessage
	if err := jsonutil.Unmarshal(impExt, &extMap); err != nil {
		return nil, err
	}
	if extMap == nil {
		extMap = make(map[string]json.RawMessage)
	}

	for key := range extMap {
		if openrtb_ext.IsPotentialBidder(key) {
			delete(extMap, key)
		}
	}

	if prebidJSON, ok := extMap[openrtb_ext.PrebidExtKey]; ok {
		var prebidMap map[string]json.RawMessage
		if err := jsonutil.Unmarshal(prebidJSON, &prebidMap); err != nil {
			return nil, err
		}
		delete(prebidMap, openrtb_ext.PrebidExtBidderKey)
		if len(prebidMap) == 0 {
			delete(extMap, openrtb_ext.PrebidExtKey)
		} else {
			var err error
			extMap[openrtb_ext.PrebidExtKey], err = jsonutil.Marshal(prebidMap)
			if err != nil {
				return nil, err
			}
		}
	}

	bidderJSON, err := jsonutil.Marshal(params)
	if err != nil {
		return nil, err
	}
	extMap[bidderExtKey] = bidderJSON

	return jsonutil.Marshal(extMap)
}

func setPublisherID(request *openrtb2.BidRequest, publisherID string) {
	if request.Site != nil {
		if request.Site.Publisher == nil {
			request.Site.Publisher = &openrtb2.Publisher{}
		}
		request.Site.Publisher.ID = publisherID
	}

	if request.App != nil {
		if request.App.Publisher == nil {
			request.App.Publisher = &openrtb2.Publisher{}
		}
		request.App.Publisher.ID = publisherID
	}
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	response := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if bidResponse.Cur != "" {
		response.Currency = bidResponse.Cur
	}

	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getBidType(&seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			response.Bids = append(response.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return response, errs
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		return getBidTypeFromExt(bid)
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for imp %s", bid.MType, bid.ImpID),
		}
	}
}

func getBidTypeFromExt(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	var bidExt openrtb_ext.ExtBid
	if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("failed to parse bid.ext for imp %s: %v", bid.ImpID, err),
		}
	}

	if bidExt.Prebid == nil || bidExt.Prebid.Type == "" {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("missing bid.ext.prebid.type for imp %s", bid.ImpID),
		}
	}

	bidType, err := openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
	if err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("invalid bid.ext.prebid.type for imp %s: %v", bid.ImpID, err),
		}
	}

	switch bidType {
	case openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative:
		return bidType, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported bid.ext.prebid.type %s for imp %s", bidType, bid.ImpID),
		}
	}
}
