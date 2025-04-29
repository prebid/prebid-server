package trustx2

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the TRUSTX 2.0 adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	ext, err := parseImpExt(&request.Imp[0])
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	body, err := jsonutil.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	publisherId := ext.PublisherId
	if request.Test == 1 {
		publisherId = "test"
	}

	reqs := make([]*adapters.RequestData, 0, 1)
	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.getUri(publisherId, ext.PlacementId),
		Body:    body,
		Headers: getHeaders(request),
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}
	reqs = append(reqs, requestData)

	return reqs, errs
}

func (a *adapter) getUri(publisherId string, placementId string) string {
	values := url.Values{}
	values.Add("publisher_id", publisherId)
	values.Add("placement_id", placementId)
	return a.endpoint + "?" + values.Encode()
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	var bidErrors []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for i := range resp.SeatBid {
		seatBid := &resp.SeatBid[i]
		for j := range seatBid.Bid {
			bid := &seatBid.Bid[j]
			typedBid, err := getTypedBid(bid)
			if err != nil {
				bidErrors = append(bidErrors, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, bidErrors
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpTrustX2, error) {
	var bidderExt adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while decoding imp.Ext, err: %v", err),
		}
	}

	ext := openrtb_ext.ExtImpTrustX2{}
	err := jsonutil.Unmarshal(bidderExt.Bidder, &ext)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while decoding bidderExt.Bidder, err: %v", err),
		}
	}

	return &ext, nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")
	headers.Set("X-Openrtb-Version", "2.6")

	if request.Site != nil {
		if request.Site.Ref != "" {
			headers.Set("Referer", request.Site.Ref)
		}
		if request.Site.Domain != "" {
			headers.Set("Origin", request.Site.Domain)
		}
	}

	if request.Device != nil {
		if len(request.Device.IP) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IP)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.UA) > 0 {
			headers.Set("User-Agent", request.Device.UA)
		}
	}

	return headers
}

func getTypedBid(bid *openrtb2.Bid) (*adapters.TypedBid, error) {
	var bidType openrtb_ext.BidType
	switch bid.MType {
	case openrtb2.MarkupBanner:
		bidType = openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		bidType = openrtb_ext.BidTypeVideo
	default:
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType: %v", bid.MType),
		}
	}

	var extBidPrebidVideo *openrtb_ext.ExtBidPrebidVideo
	if bidType == openrtb_ext.BidTypeVideo {
		extBidPrebidVideo = &openrtb_ext.ExtBidPrebidVideo{}
		if len(bid.Cat) > 0 {
			extBidPrebidVideo.PrimaryCategory = bid.Cat[0]
		}
		if bid.Dur > 0 {
			extBidPrebidVideo.Duration = int(bid.Dur)
		}
	}
	return &adapters.TypedBid{
		Bid:      bid,
		BidType:  bidType,
		BidVideo: extBidPrebidVideo,
	}, nil
}
