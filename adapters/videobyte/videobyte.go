package videobyte

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

type adapter struct {
	endpoint string
}

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

		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint + "?" + getParams(impExt).Encode(),
			Body:    body,
			Headers: getHeaders(request),
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	request.Imp = impressions
	return adapterRequests, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad user input: HTTP status %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var ortbResponse openrtb2.BidResponse
	err := jsonutil.Unmarshal(response.Body, &ortbResponse)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	impIdToImp := make(map[string]*openrtb2.Imp)
	for i := range internalRequest.Imp {
		imp := internalRequest.Imp[i]
		impIdToImp[imp.ID] = &imp
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, seatBid := range ortbResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(impIdToImp[bid.ImpID]),
			})
		}
	}

	return bidderResponse, nil
}

func getMediaTypeForImp(imp *openrtb2.Imp) openrtb_ext.BidType {
	if imp != nil && imp.Banner != nil {
		return openrtb_ext.BidTypeBanner
	}
	return openrtb_ext.BidTypeVideo
}

func getParams(impExt *openrtb_ext.ExtImpVideoByte) url.Values {
	params := url.Values{}
	params.Add("source", "pbs")
	params.Add("pid", impExt.PublisherId)
	if impExt.PlacementId != "" {
		params.Add("placementId", impExt.PlacementId)
	}
	if impExt.NetworkId != "" {
		params.Add("nid", impExt.NetworkId)
	}
	return params
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Site != nil {
		if request.Site.Domain != "" {
			headers.Add("Origin", request.Site.Domain)
		}
		if request.Site.Ref != "" {
			headers.Set("Referer", request.Site.Ref)
		}
	}
	return headers
}

func parseExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpVideoByte, error) {
	var bidderExt adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpVideoByte{}
	err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	return &impExt, nil
}
