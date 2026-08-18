package tunnl

import (
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

// Tunnl exposes one endpoint per region and media type. The media type selects
// the sid suffix, so a single bid request that covers several media types must be
// split into one call per (imp, media type).
const (
	formatBanner = "ban"
	formatVideo  = "vid"
	formatNative = "nat"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{endpoint: endpoint}, nil
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
	requestFormat := formatFromRequestURI(requestData.Uri)

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

// formatFromRequestURI recovers the media type from the sid query parameter of
// the outgoing request, which always ends with the format suffix. Returns an
// empty string if it cannot be determined.
func formatFromRequestURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	sid := parsed.Query().Get("sid")
	for _, format := range []string{formatBanner, formatVideo, formatNative} {
		if strings.HasSuffix(sid, format) {
			return format
		}
	}
	return ""
}

func getMediaTypeForBid(bid openrtb2.Bid, requestFormat string) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
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
