package outbrain

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/native1"
	nativeResponse "github.com/mxmCherry/openrtb/v15/native1/response"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Outbrain adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	reqCopy := *request

	var errs []error
	var outbrainExt openrtb_ext.ExtImpOutbrain
	for i := 0; i < len(reqCopy.Imp); i++ {
		imp := reqCopy.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := json.Unmarshal(bidderExt.Bidder, &outbrainExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if outbrainExt.TagId != "" {
			imp.TagID = outbrainExt.TagId
			reqCopy.Imp[i] = imp
		}
	}

	publisher := &openrtb2.Publisher{
		ID:     outbrainExt.Publisher.Id,
		Name:   outbrainExt.Publisher.Name,
		Domain: outbrainExt.Publisher.Domain,
	}
	if reqCopy.Site != nil {
		siteCopy := *reqCopy.Site
		siteCopy.Publisher = publisher
		reqCopy.Site = &siteCopy
	} else if reqCopy.App != nil {
		appCopy := *reqCopy.App
		appCopy.Publisher = publisher
		reqCopy.App = &appCopy
	}

	if outbrainExt.BCat != nil {
		reqCopy.BCat = outbrainExt.BCat
	}
	if outbrainExt.BAdv != nil {
		reqCopy.BAdv = outbrainExt.BAdv
	}

	requestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
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

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getMediaTypeForImp(bid.ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if bidType == openrtb_ext.BidTypeNative {
				var nativePayload nativeResponse.Response
				if err := json.Unmarshal(json.RawMessage(bid.AdM), &nativePayload); err != nil {
					errs = append(errs, err)
					continue
				}
				transformEventTrackers(&nativePayload)
				nativePayloadJson, err := json.Marshal(nativePayload)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				bid.AdM = string(nativePayloadJson)
			}

			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			} else if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find native/banner impression \"%s\" ", impID),
	}
}

func transformEventTrackers(nativePayload *nativeResponse.Response) {
	// the native-trk.js library used to trigger the trackers currently doesn't support the native 1.2 eventtrackers,
	// so transform them to the deprecated imptrackers and jstracker
	for _, eventTracker := range nativePayload.EventTrackers {
		if eventTracker.Event != native1.EventTypeImpression {
			continue
		}
		switch eventTracker.Method {
		case native1.EventTrackingMethodImage:
			nativePayload.ImpTrackers = append(nativePayload.ImpTrackers, eventTracker.URL)
		case native1.EventTrackingMethodJS:
			nativePayload.JSTracker = fmt.Sprintf("<script src=\"%s\"></script>", eventTracker.URL)
		}
	}
	nativePayload.EventTrackers = nil
}
