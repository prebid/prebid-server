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

const (
	contentTypeJSON = "application/json"
	openRTBVersion  = "2.6"
	currencyUSD     = "USD"
	bidderSeat      = "resetdigital"
)

var baseHeaders = http.Header{
	"Content-Type":      []string{contentTypeJSON},
	"Accept":            []string{contentTypeJSON},
	"X-OpenRTB-Version": []string{openRTBVersion},
}

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: cfg.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	_ = reqInfo

	if len(request.Imp) != 1 {
		return nil, []error{&errortypes.BadInput{
			Message: "ResetDigital adapter supports only one impression per request",
		}}
	}

	errs := make([]error, 0, 1)

	imp := request.Imp[0]
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error parsing bidderExt from imp.ext: %v", err),
		}}
	}

	var resetDigitalExt openrtb_ext.ImpExtResetDigital
	if err := json.Unmarshal(bidderExt.Bidder, &resetDigitalExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error parsing resetDigitalExt from bidderExt.bidder: %v", err),
		}}
	}

	if resetDigitalExt.PlacementID == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "Missing required parameter 'placement_id'",
		}}
	}

	reqCopy := openrtb2.BidRequest{
		ID:     request.ID,
		Source: request.Source,
		TMax:   request.TMax,
		Test:   request.Test,
		Imp:    []openrtb2.Imp{imp},
		Device: request.Device,
		Site:   request.Site,
		App:    request.App,
		User:   request.User,
		Regs:   request.Regs,
		Ext:    request.Ext,
		AT:     request.AT,
		BAdv:   request.BAdv,
		BCat:   request.BCat,
		BSeat:  request.BSeat,
		WLang:  request.WLang,
		WSeat:  request.WSeat,
	}

	if imp.TagID == "" {
		reqCopy.Imp[0].TagID = resetDigitalExt.PlacementID
	}

	if len(request.Cur) == 0 || (len(request.Cur) == 1 && request.Cur[0] == "") {
		reqCopy.Cur = []string{currencyUSD}
	} else {
		reqCopy.Cur = request.Cur
	}

	reqBody, err := json.Marshal(&reqCopy)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error marshalling OpenRTB request: %v", err),
		}}
	}

	uri := a.endpoint
	if resetDigitalExt.PlacementID != "" {
		uri = fmt.Sprintf("%s?pid=%s", a.endpoint, resetDigitalExt.PlacementID)
	}

	reqHeaders := baseHeaders.Clone()

	reqs := []*adapters.RequestData{
		{
			Method:  http.MethodPost,
			Uri:     uri,
			Body:    reqBody,
			Headers: reqHeaders,
			ImpIDs:  []string{imp.ID},
		},
	}

	return reqs, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode >= http.StatusBadRequest && responseData.StatusCode < http.StatusInternalServerError {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %s", err),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	return parseBidResponse(request, &bidResp)
}

func parseBidResponse(request *openrtb2.BidRequest, bidResp *openrtb2.BidResponse) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	var errs []error

	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	} else {
		bidResponse.Currency = currencyUSD
	}

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			if seatBid.Bid[i].Price <= 0 {
				errs = append(errs, &errortypes.Warning{
					Message: fmt.Sprintf("price %f <= 0 filtered out", seatBid.Bid[i].Price),
				})
				continue
			}

			bidType, err := getBidType(seatBid.Bid[i], request)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			seat := openrtb_ext.BidderName(bidderSeat)
			if seatBid.Seat != "" {
				seat = openrtb_ext.BidderName(seatBid.Seat)
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
				Seat:    seat,
			})
		}
	}

	return bidResponse, errs
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

	if len(request.Imp) == 1 {
		if request.Imp[0].ID != bid.ImpID {
			return "", fmt.Errorf("no matching impression found for ImpID: %s", bid.ImpID)
		}
		return getMediaType(request.Imp[0]), nil
	}

	for _, imp := range request.Imp {
		if bid.ImpID == imp.ID {
			return getMediaType(imp), nil
		}
	}

	return "", fmt.Errorf("no matching impression found for ImpID: %s", bid.ImpID)
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
