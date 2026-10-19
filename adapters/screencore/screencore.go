package screencore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

type reqBodyExtBidder struct {
	SspPlacementID string `json:"sspPlacementId,omitempty"`
}

const sspPath = "/pbs-bid"

var regionHosts = map[string]string{
	"us":   "ssp-us",
	"eu":   "ssp-eu",
	"asia": "ssp-asia",
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	return &adapter{endpoint: endpointTemplate}, nil
}

func impError(impID string, err error) error {
	return &errortypes.BadInput{
		Message: fmt.Sprintf("imp %s: %s", impID, err.Error()),
	}
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		var screencoreExt openrtb_ext.ImpExtScreencore
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &screencoreExt); err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		if screencoreExt.SspPlacementID == "" {
			errs = append(errs, impError(imp.ID, fmt.Errorf("sspPlacementId is required")))
			continue
		}

		endpoint, err := a.buildSspEndpoint(screencoreExt)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		adapterReq, err := a.makeRequest(request, imp, screencoreExt, endpoint)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		adapterRequests = append(adapterRequests, adapterReq)
	}

	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, imp openrtb2.Imp, ext openrtb_ext.ImpExtScreencore, endpoint string) (*adapters.RequestData, error) {
	impExtJSON, err := buildImpExt(imp.Ext, ext.SspPlacementID)
	if err != nil {
		return nil, impError(imp.ID, err)
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

func buildImpExt(ext json.RawMessage, sspPlacementID string) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if len(ext) > 0 {
		if err := jsonutil.Unmarshal(ext, &fields); err != nil {
			return nil, err
		}
	}

	bidder, err := jsonutil.Marshal(reqBodyExtBidder{SspPlacementID: sspPlacementID})
	if err != nil {
		return nil, err
	}
	fields["bidder"] = bidder

	return jsonutil.Marshal(fields)
}

func (a *adapter) buildSspEndpoint(ext openrtb_ext.ImpExtScreencore) (string, error) {
	host, err := resolveRegionHost(ext.Region)
	if err != nil {
		return "", err
	}

	base, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{Host: host})
	if err != nil {
		return "", fmt.Errorf("unable to resolve endpoint macros: %v", err)
	}

	endpointURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid ssp endpoint %q: %v", base, err)
	}

	endpointURL.Path = strings.TrimSuffix(endpointURL.Path, "/") + sspPath
	query := endpointURL.Query()
	query.Set("pId", ext.SspPlacementID)
	endpointURL.RawQuery = query.Encode()

	return endpointURL.String(), nil
}

func resolveRegionHost(region string) (string, error) {
	if region == "" {
		return "", fmt.Errorf("region is required when sspPlacementId is provided")
	}
	host, ok := regionHosts[region]
	if !ok {
		return "", fmt.Errorf("unsupported region %q, expected one of us/eu/asia", region)
	}
	return host, nil
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %s", err.Error()),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) != 0 {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("could not define media type for bid: %s", bid.ImpID),
		}
	}
}
