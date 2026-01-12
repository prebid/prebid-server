package waardex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type waardexAdapter struct {
	EndpointTemplate *template.Template
}

// MakeRequests prepares request information for prebid-server core
func (adapter *waardexAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	impressionsByZone, impErrs := groupImpressionsByZone(request.Imp)
	errs = append(errs, impErrs...)
	if len(impressionsByZone) == 0 {
		return nil, errs
	}
	result := make([]*adapters.RequestData, 0, len(impressionsByZone))
	for k, imps := range impressionsByZone {
		bidRequest, err := adapter.buildAdapterRequest(request, &k, imps)
		if err != nil {
			errs = append(errs, err)
		} else {
			result = append(result, bidRequest)
		}
	}
	return result, errs
}

// groupImpressionsByZone validates imps and groups them by Waardex-specific parameter `zoneId`.
func groupImpressionsByZone(imps []openrtb2.Imp) (map[openrtb_ext.ExtImpWaardex][]openrtb2.Imp, []error) {
	res := make(map[openrtb_ext.ExtImpWaardex][]openrtb2.Imp)
	errors := make([]error, 0, len(imps))
	for idx := range imps {
		imp := imps[idx]
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		imp.Ext = nil
		// Additional validation is handled by the core JSON schema (static/bidder-params/waardex.json).
		if isMultiFormatImp(&imp) {
			splImps := splitMultiFormatImp(&imp)
			if len(splImps) == 0 {
				continue
			}
			if _, exists := res[*impExt]; !exists {
				res[*impExt] = make([]openrtb2.Imp, 0, 4)
			}
			res[*impExt] = append(res[*impExt], splImps...)
		} else {
			if _, exists := res[*impExt]; !exists {
				res[*impExt] = make([]openrtb2.Imp, 0, 4)
			}
			res[*impExt] = append(res[*impExt], imp)
		}
	}
	return res, errors
}

func isMultiFormatImp(imp *openrtb2.Imp) bool {
	formatCount := 0
	if imp.Video != nil {
		formatCount++
	}
	if imp.Audio != nil {
		formatCount++
	}
	if imp.Banner != nil {
		formatCount++
	}
	if imp.Native != nil {
		formatCount++
	}
	return formatCount > 1
}

func splitMultiFormatImp(imp *openrtb2.Imp) []openrtb2.Imp {
	splitImps := make([]openrtb2.Imp, 0, 2)
	if imp.Banner != nil {
		impCopy := *imp
		impCopy.Video = nil
		impCopy.Native = nil
		impCopy.Audio = nil
		splitImps = append(splitImps, impCopy)
	}
	if imp.Video != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Native = nil
		impCopy.Audio = nil
		splitImps = append(splitImps, impCopy)
	}
	return splitImps
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpWaardex, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var waardexExt openrtb_ext.ExtImpWaardex
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &waardexExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	return &waardexExt, nil
}

func (adapter *waardexAdapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpWaardex, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	newBidRequest := createBidRequest(prebidBidRequest, imps)
	reqJSON, err := json.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	url, err := adapter.buildEndpointURL(params)
	if err != nil {
		return nil, err
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps)}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, imps []openrtb2.Imp) *openrtb2.BidRequest {
	bidRequest := *prebidBidRequest
	bidRequest.Imp = imps
	if bidRequest.Site != nil {
		// Need to copy Site as Request is a shallow copy
		site := *bidRequest.Site
		site.Publisher = nil
		bidRequest.Site = &site
	}
	if bidRequest.App != nil {
		// Need to copy App as Request is a shallow copy
		app := *bidRequest.App
		app.Publisher = nil
		bidRequest.App = &app
	}
	return &bidRequest
}

// Builds endpoint url based on adapter-specific pub settings from imp.ext
func (adapter *waardexAdapter) buildEndpointURL(params *openrtb_ext.ExtImpWaardex) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ZoneID: strconv.Itoa(params.ZoneId)}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

// MakeBids translates Waardex bid response to prebid-server specific format
func (adapter *waardexAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)),
		}
	}
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Bad server response: %v", err)),
		}
	}

	if len(bidResp.SeatBid) != 1 {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Invalid SeatBids count: %d", len(bidResp.SeatBid))),
		}
	}

	seatBid := bidResp.SeatBid[0]
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}
	var errs []error
	for i := 0; i < len(seatBid.Bid); i++ {
		bid := seatBid.Bid[i]
		bidType, err := getMediaTypeForBid(&bid)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}
	return bidResponse, errs
}

// getMediaTypeForImp figures out which media type this bid is for
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
		}
	}
}

func newBadInputError(message string) error {
	return &errortypes.BadInput{
		Message: message,
	}
}

func newBadServerResponseError(message string) error {
	return &errortypes.BadServerResponse{
		Message: message,
	}
}

// Builder builds a new instance of the waardex adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	urlTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &waardexAdapter{
		EndpointTemplate: urlTemplate,
	}
	return bidder, nil
}
