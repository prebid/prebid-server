package escalax

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

const defaultRegion = "us"

var exchangeRegionSubdomains = map[string]string{
	"us":   "bidder-us",
	"eu":   "bidder_eu",
	"apac": "bidder_apac",
}

var sspRegionSubdomains = map[string]string{
	"us":   "s-us",
	"eu":   "s-eu",
	"apac": "s-as",
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

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	requests := make([]*adapters.RequestData, 0, len(openRTBRequest.Imp))

	for _, imp := range openRTBRequest.Imp {
		ext, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		endpoint, err := a.resolveEndpoint(ext)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		impExt, err := stripBidderParams(imp.Ext)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}
		imp.Ext = impExt

		reqCopy := *openRTBRequest
		reqCopy.Imp = []openrtb2.Imp{imp}

		reqJSON, err := jsonutil.Marshal(reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Body:    reqJSON,
			Uri:     endpoint,
			Headers: buildHeaders(&reqCopy),
			ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
		})
	}

	return requests, errs
}

func stripBidderParams(ext json.RawMessage) (json.RawMessage, error) {
	if len(ext) == 0 {
		return nil, nil
	}

	var fields map[string]json.RawMessage
	if err := jsonutil.Unmarshal(ext, &fields); err != nil {
		return nil, err
	}

	delete(fields, "bidder")
	if len(fields) == 0 {
		return nil, nil
	}

	return jsonutil.Marshal(fields)
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtEscalax, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{Message: "Error parsing escalaxExt - " + err.Error()}
	}

	var escalaxExt openrtb_ext.ExtEscalax
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &escalaxExt); err != nil {
		return nil, &errortypes.BadInput{Message: "Error parsing bidderExt - " + err.Error()}
	}

	return &escalaxExt, nil
}

func (a *adapter) resolveEndpoint(ext *openrtb_ext.ExtEscalax) (string, error) {
	hasExchangeParams := ext.AccountID != nil || ext.SourceID != nil
	isExchange := ext.AccountID != nil && ext.SourceID != nil
	isSsp := ext.SupplyPlacementID != nil

	switch {
	case hasExchangeParams && isSsp:
		return "", &errortypes.BadInput{
			Message: "provide either accountId and sourceId, or supplyPlacementId, not both",
		}
	case isExchange:
		return a.buildExchangeEndpoint(ext)
	case isSsp:
		return a.buildSspEndpoint(ext)
	case hasExchangeParams:
		return "", &errortypes.BadInput{
			Message: "accountId and sourceId must be provided together",
		}
	default:
		return "", &errortypes.BadInput{
			Message: "either accountId and sourceId, or supplyPlacementId is required",
		}
	}
}

func (a *adapter) buildExchangeEndpoint(ext *openrtb_ext.ExtEscalax) (string, error) {
	subdomain, err := resolveSubdomain(exchangeRegionSubdomains, ext.Region)
	if err != nil {
		return "", err
	}

	return a.buildURL(subdomain, "/", map[string]string{
		"partner": *ext.SourceID,
		"token":   *ext.AccountID,
		"type":    "pbs",
	})
}

func (a *adapter) buildSspEndpoint(ext *openrtb_ext.ExtEscalax) (string, error) {
	subdomain, err := resolveSubdomain(sspRegionSubdomains, ext.Region)
	if err != nil {
		return "", err
	}

	return a.buildURL(subdomain, "/pbsb", map[string]string{
		"placementId": *ext.SupplyPlacementID,
	})
}

func (a *adapter) buildURL(subdomain string, path string, params map[string]string) (string, error) {
	base, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{Host: subdomain})
	if err != nil {
		return "", fmt.Errorf("unable to resolve endpoint macros: %v", err)
	}

	endpointURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint %q: %v", base, err)
	}

	endpointURL.Path = strings.TrimSuffix(endpointURL.Path, "/") + path
	query := endpointURL.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	endpointURL.RawQuery = query.Encode()

	return endpointURL.String(), nil
}

func resolveSubdomain(regionMap map[string]string, region *string) (string, error) {
	key := defaultRegion
	if region != nil {
		key = *region
	}

	subdomain, ok := regionMap[key]
	if !ok {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("unsupported region %q, expected one of us/eu/apac", key),
		}
	}
	return subdomain, nil
}

func buildHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device == nil {
		return headers
	}

	if len(request.Device.UA) > 0 {
		headers.Add("User-Agent", request.Device.UA)
	}
	if len(request.Device.IPv6) > 0 {
		headers.Add("X-Forwarded-For", request.Device.IPv6)
	}
	if len(request.Device.IP) > 0 {
		headers.Add("X-Forwarded-For", request.Device.IP)
	}

	return headers
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(bidderRawResponse); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "Bad Server Response"}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{Message: "Empty SeatBid array"}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(requestToBidder.ImpIDs))
	if len(bidResp.Cur) != 0 {
		bidResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for idx := range sb.Bid {
			bidType, err := determineImpressionMediaType(sb.Bid[idx])
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[idx],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func determineImpressionMediaType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}