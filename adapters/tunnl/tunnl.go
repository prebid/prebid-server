package tunnl

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Tunnl exposes one endpoint per region and media type. The media type selects
// the sid suffix, so a single bid request that covers several media types must be
// split into one call per (imp, media type).
const (
	formatBanner = "ban"
	formatVideo  = "vid"
	formatNative = "nat"
)

// supportedFormats is the full set of media types Tunnl serves, in the order the
// adapter splits impressions.
var supportedFormats = []string{formatBanner, formatVideo, formatNative}

type adapter struct {
	endpoint *template.Template

	// uriFormats maps each endpoint URI this adapter can emit back to the media
	// type it was built for. The mapping is derived from the configured endpoint
	// at build time rather than re-parsed out of the URI later, so recovering the
	// format in MakeBids does not depend on the endpoint carrying any particular
	// query parameter. A host may point tunnl at a proxy or a future Tunnl URL
	// scheme and the media type of a bid stays recoverable.
	uriFormats map[string]string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint:   endpoint,
		uriFormats: make(map[string]string, len(supportedFormats)),
	}

	// Resolving every format up front both builds the reverse mapping and fails
	// fast on an endpoint that cannot be resolved at all, rather than at auction
	// time on every request.
	for _, format := range supportedFormats {
		uri, err := bidder.buildEndpointURL(format)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve endpoint url for media type %s: %v", format, err)
		}
		bidder.uriFormats[uri] = format
	}

	// A configured endpoint that ignores {{.MediaType}} collapses all three
	// formats onto one URI. Tunnl serves a single media type per endpoint, so
	// such a configuration would send every format to the same place and make
	// responses unattributable. Reject it here instead of silently dropping
	// bids later.
	if len(bidder.uriFormats) != len(supportedFormats) {
		return nil, fmt.Errorf("endpoint url must vary by media type, add the {{.MediaType}} macro: %s", cfg.Endpoint)
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	headers := makeHeaders()

	for _, imp := range request.Imp {
		splitImps, err := splitImpByMediaType(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, split := range splitImps {
			requestData, err := a.makeRequest(request, split.imp, split.format, headers)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			requests = append(requests, requestData)
		}
	}

	return requests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, imp openrtb2.Imp, format string, headers http.Header) (*adapters.RequestData, error) {
	endpoint, err := a.buildEndpointURL(format)
	if err != nil {
		return nil, err
	}

	// A shallow copy is enough: only Imp is replaced, and imp elements are
	// already shallow copies owned by this adapter. Nothing else is mutated.
	reqCopy := *request
	reqCopy.Imp = []openrtb2.Imp{imp}

	body, err := jsonutil.Marshal(&reqCopy)
	if err != nil {
		return nil, err
	}

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
	}, nil
}

// buildEndpointURL resolves the media type into the host configured endpoint.
// The region is part of that configuration, not something the adapter chooses.
func (a *adapter) buildEndpointURL(format string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		MediaType: format,
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

type splitImp struct {
	imp    openrtb2.Imp
	format string
}

// splitImpByMediaType turns one impression into one impression per media type it
// declares, keeping only the matching media type object on each copy. This is
// required because each Tunnl endpoint serves a single format.
func splitImpByMediaType(imp openrtb2.Imp) ([]splitImp, error) {
	splits := make([]splitImp, 0, 3)

	// Each copy keeps exactly one media type object. Clearing every other type
	// explicitly rather than by subtraction means an unsupported media type,
	// such as audio, is never forwarded to a format specific endpoint.
	if imp.Banner != nil {
		impCopy := imp
		impCopy.Video, impCopy.Native, impCopy.Audio = nil, nil, nil
		splits = append(splits, splitImp{imp: impCopy, format: formatBanner})
	}
	if imp.Video != nil {
		impCopy := imp
		impCopy.Banner, impCopy.Native, impCopy.Audio = nil, nil, nil
		splits = append(splits, splitImp{imp: impCopy, format: formatVideo})
	}
	if imp.Native != nil {
		impCopy := imp
		impCopy.Banner, impCopy.Video, impCopy.Audio = nil, nil, nil
		splits = append(splits, splitImp{imp: impCopy, format: formatNative})
	}

	if len(splits) == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unsupported media type, Tunnl supports banner, video and native", imp.ID),
		}
	}

	return splits, nil
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
		return nil, []error{&errortypes.BadServerResponse{Message: err.Error()}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	// This response came from a format specific endpoint, so the format the
	// request was built for is a reliable fallback when the bid omits mtype.
	requestFormat := a.uriFormats[requestData.Uri]

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i], requestFormat)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid, requestFormat string) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupAudio:
		// Audio impressions are never sent, so an audio bid cannot be matched to
		// anything that was requested. Reject it rather than letting it fall
		// through to the request format and reach the publisher mislabelled.
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported media type audio for bid %q in imp %q, Tunnl supports banner, video and native", bid.ID, bid.ImpID),
		}
	}

	switch requestFormat {
	case formatBanner:
		return openrtb_ext.BidTypeBanner, nil
	case formatVideo:
		return openrtb_ext.BidTypeVideo, nil
	case formatNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to determine media type for bid %q in imp %q", bid.ID, bid.ImpID),
	}
}

func makeHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-OpenRTB-Version", "2.6")
	return headers
}
