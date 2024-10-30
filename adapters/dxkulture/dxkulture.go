package dxkulture

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/openrtb/v20/openrtb2"
)

var markupTypeToBidType = map[openrtb2.MarkupType]openrtb_ext.BidType{
	openrtb2.MarkupBanner: openrtb_ext.BidTypeBanner,
	openrtb2.MarkupVideo:  openrtb_ext.BidTypeVideo,
}

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the DXKulture adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	impressions := request.Imp

	adapterRequests := make([]*adapters.RequestData, 0, len(impressions))
	var errs []error

	for _, impression := range impressions {
		impExt, err := parseExt(&impression)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		request.Imp = []openrtb2.Imp{impression}
		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if request.Test == 1 {
			impExt.PublisherId = "test"
		}

		params := url.Values{}
		params.Add("publisher_id", impExt.PublisherId)
		params.Add("placement_id", impExt.PlacementId)

		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint + "?" + params.Encode(),
			Body:    body,
			Headers: getHeaders(request),
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	request.Imp = impressions
	return adapterRequests, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var ortbResponse openrtb2.BidResponse
	err := jsonutil.Unmarshal(response.Body, &ortbResponse)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	var bidErrors []error

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, seatBid := range ortbResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getBidType(&bid)
			if err != nil {
				bidErrors = append(bidErrors, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, bidErrors
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bidType, ok := markupTypeToBidType[bid.MType]; ok {
		return bidType, nil
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
	}
}

func parseExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpDXKulture, error) {
	var bidderExt adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpDXKulture{}
	err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	return &impExt, nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Site != nil {
		if request.Site.Ref != "" {
			headers.Set("Referer", request.Site.Ref)
		}
		if request.Site.Domain != "" {
			headers.Add("Origin", request.Site.Domain)
		}
	}

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}
	return headers
}
