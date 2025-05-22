package resetdigital

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: cfg.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	var requests []*adapters.RequestData

	for _, imp := range request.Imp {
		if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "failed to find matching imp for bid " + imp.ID,
			})
			continue
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing bidderExt from imp.ext: %v", err),
			})
			continue
		}

		var resetDigitalExt openrtb_ext.ImpExtResetDigital
		if err := json.Unmarshal(bidderExt.Bidder, &resetDigitalExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing resetDigitalExt from bidderExt.bidder: %v", err),
			})
			continue
		}

		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}

		if imp.TagID == "" {
			reqCopy.Imp[0].TagID = resetDigitalExt.PlacementID
		}

		reqBody, err := json.Marshal(&reqCopy)
		if err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error marshalling OpenRTB request: %v", err),
			})
			continue
		}

		uri := a.endpoint
		if resetDigitalExt.PlacementID != "" {
			uri = fmt.Sprintf("%s?pid=%s", a.endpoint, resetDigitalExt.PlacementID)
		}

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqBody,
			Headers: getHeaders(),
			ImpIDs:  []string{imp.ID},
		})
	}

	return requests, errors
}

func getHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	headers.Add("X-OpenRTB-Version", "2.6")
	return headers
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to parse response body: %v", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	} else {
		bidResponse.Currency = "USD"
	}

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			if seatBid.Bid[i].Price <= 0 {
				continue
			}

			bidType, err := getBidType(seatBid.Bid[i], request)
			if err != nil {
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, nil
}

func getBidType(bid openrtb2.Bid, request *openrtb2.BidRequest) (openrtb_ext.BidType, error) {
	if bid.MType > 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		case openrtb2.MarkupAudio:
			return openrtb_ext.BidTypeAudio, nil
		case openrtb2.MarkupNative:
			return openrtb_ext.BidTypeNative, nil
		}
	}

	var impOrtb openrtb2.Imp
	var found bool
	for _, imp := range request.Imp {
		if bid.ImpID == imp.ID {
			impOrtb = imp
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("no matching impression found for ImpID: %s", bid.ImpID)
	}

	return getMediaType(impOrtb), nil
}

func getMediaType(imp openrtb2.Imp) openrtb_ext.BidType {
	switch {
	case imp.Video != nil:
		return openrtb_ext.BidTypeVideo
	case imp.Audio != nil:
		return openrtb_ext.BidTypeAudio
	case imp.Native != nil:
		return openrtb_ext.BidTypeNative
	default:
		return openrtb_ext.BidTypeBanner
	}
}
