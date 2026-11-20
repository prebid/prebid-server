package adplayx

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	parsedURL, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, []error{err}
	}

	for i := range request.Imp {
		imp := request.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("error unmarshalling imp.ext: %s", err)})
			continue
		}

		var impExt openrtb_ext.ExtImpAdplayx
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("error unmarshalling imp.ext.bidder: %s", err)})
			continue
		}

		if strings.TrimSpace(impExt.AppToken) == "" {
			errs = append(errs, &errortypes.BadInput{Message: "apptoken is required"})
			continue
		}

		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}

		body, err := jsonutil.Marshal(reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		urlCopy := *parsedURL
		query := urlCopy.Query()
		query.Set("apptoken", strings.TrimSpace(impExt.AppToken))
		if p := strings.TrimSpace(impExt.PlacementID); p != "" {
			query.Set("placementid", p)
		}
		urlCopy.RawQuery = query.Encode()

		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		headers.Add("Accept", "application/json")

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     urlCopy.String(),
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
		})
	}

	return requests, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	var outboundRequest openrtb2.BidRequest
	if err := jsonutil.Unmarshal(requestData.Body, &outboundRequest); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(outboundRequest.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getMediaTypeForBid(bid, outboundRequest.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}
		if bid.MType != 0 {
			switch bid.MType {
			case openrtb2.MarkupBanner:
				return openrtb_ext.BidTypeBanner, nil
			case openrtb2.MarkupVideo:
				return openrtb_ext.BidTypeVideo, nil
			case openrtb2.MarkupAudio:
				return openrtb_ext.BidTypeAudio, nil
			case openrtb2.MarkupNative:
				return openrtb_ext.BidTypeNative, nil
			default:
				return "", &errortypes.BadServerResponse{
					Message: fmt.Sprintf("unsupported bid.mtype %d for impression %s", bid.MType, bid.ImpID),
				}
			}
		}
		if imp.Banner != nil {
			return openrtb_ext.BidTypeBanner, nil
		}
		if imp.Video != nil {
			return openrtb_ext.BidTypeVideo, nil
		}
		if imp.Audio != nil {
			return openrtb_ext.BidTypeAudio, nil
		}
		if imp.Native != nil {
			return openrtb_ext.BidTypeNative, nil
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("failed to determine media type for impression \"%s\"", bid.ImpID),
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("failed to find impression \"%s\"", bid.ImpID),
	}
}
