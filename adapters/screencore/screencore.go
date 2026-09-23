package screencore

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct{}

type reqBodyExt struct {
	ScreencoreBidderExt reqBodyExtBidder `json:"bidder"`
}

type reqBodyExtBidder struct {
	SspPlacementID string `json:"sspPlacementId,omitempty"`
}

var regionEndpoints = map[string]string{
	"us":   "https://ssp-us.screencore.io/pbs-bid",
	"eu":   "https://ssp-eu.screencore.io/pbs-bid",
	"asia": "https://ssp-asia.screencore.io/pbs-bid",
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		var screencoreExt openrtb_ext.ImpExtScreencore
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &screencoreExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if screencoreExt.SspPlacementID == "" {
			errs = append(errs, fmt.Errorf("imp %s: sspPlacementId is required", imp.ID))
			continue
		}

		adapterReq, err := a.makeRequest(request, imp, screencoreExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		adapterRequests = append(adapterRequests, adapterReq)
	}

	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, imp openrtb2.Imp, ext openrtb_ext.ImpExtScreencore) (*adapters.RequestData, error) {
	endpoint, err := buildSspEndpoint(ext)
	if err != nil {
		return nil, fmt.Errorf("imp %s: %w", imp.ID, err)
	}

	impExt := reqBodyExt{
		ScreencoreBidderExt: reqBodyExtBidder{
			SspPlacementID: ext.SspPlacementID,
		},
	}
	impExtJSON, err := jsonutil.Marshal(impExt)
	if err != nil {
		return nil, err
	}

	reqCopy := *request
	impCopy := imp
	impCopy.Ext = impExtJSON
	reqCopy.Imp = []openrtb2.Imp{impCopy}

	reqJSON, err := jsonutil.Marshal(reqCopy)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
	}, nil
}

func buildSspEndpoint(ext openrtb_ext.ImpExtScreencore) (string, error) {
	base, err := resolveRegionEndpoint(ext.Region)
	if err != nil {
		return "", err
	}

	endpointURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid ssp endpoint %q: %w", base, err)
	}

	query := endpointURL.Query()
	query.Set("pId", ext.SspPlacementID)
	endpointURL.RawQuery = query.Encode()

	return endpointURL.String(), nil
}

func resolveRegionEndpoint(region string) (string, error) {
	if region == "" {
		return "", fmt.Errorf("region is required when sspPlacementId is provided")
	}
	endpoint, ok := regionEndpoints[region]
	if !ok {
		return "", fmt.Errorf("unsupported region %q, expected one of us/eu/asia", region)
	}
	return endpoint, nil
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) != 0 {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, nil
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	// determine media type by bid response field mtype
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("could not define media type for bid: %s", bid.ImpID)
	}
}
