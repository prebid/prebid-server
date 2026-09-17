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
	var publisherID string

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

		if publisherID == "" {
			publisherID = bidderExt.PublisherID
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
	stampPublisherID(&reqCopy, publisherID)

	requestJSON, err := jsonutil.Marshal(reqCopy)
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

// stripBidderExt re-marshals imp.ext as a JSON object, returning nil if it is
// empty. If imp.ext is not a JSON object or can't be re-marshalled, the
// original value is returned unchanged so it is preserved rather than silently
// dropped. bidder/prebid keys are left in place so they reach the endpoint.
func stripBidderExt(ext json.RawMessage) json.RawMessage {
	if ext == nil {
		return nil
	}

	var extMap map[string]json.RawMessage
	if err := jsonutil.Unmarshal(ext, &extMap); err != nil {
		return ext
	}

	if len(extMap) == 0 {
		return nil
	}

	cleaned, err := jsonutil.Marshal(extMap)
	if err != nil {
		return ext
	}
	return cleaned
}

// stampPublisherID writes the Viant-assigned publisher ID onto site.publisher.id
// or app.publisher.id so it reaches the endpoint. request.Site/App and their
// nested Publisher are pointers shared with other bidders' requests, so they are
// shallow-copied before mutation to avoid leaking the value into other bidders.
func stampPublisherID(request *openrtb2.BidRequest, publisherID string) {
	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{}
		}
		siteCopy.Publisher.ID = publisherID
		request.Site = &siteCopy
	}

	if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{}
		}
		appCopy.Publisher.ID = publisherID
		request.App = &appCopy
	}
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
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
