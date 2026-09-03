package tunnl

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
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

// Tunnl issues one sid per publisher and region. The sid carries its region,
// and the configured endpoint's host carries it a second time as a subdomain,
// so the two have to agree: a request sent to us1 with an eu sid would bid
// against the wrong region and be attributed to the wrong place.
//
// endpointRegions maps each endpoint host prefix to the sid region segments it
// accepts. The US east and west sids share a single host.
var endpointRegions = map[string][]string{
	"us1": {"use", "usw"},
	"eu1": {"eu"},
	"ap1": {"ap"},
}

type adapter struct {
	endpoint *template.Template

	// sidRegions is the set of sid region segments the configured endpoint
	// accepts, derived from its host at build time. Empty when the host is not
	// one of Tunnl's own, in which case the region check is skipped: a host may
	// point the adapter at a proxy, and rejecting every sid there would be worse
	// than not checking.
	sidRegions map[string]struct{}
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint:   endpoint,
		sidRegions: sidRegionsForEndpoint(cfg.Endpoint),
	}

	return bidder, nil
}

// sidRegionsForEndpoint resolves the sid regions the endpoint's host serves.
// The endpoint is inspected as configured, before macro resolution, because the
// host is fixed by the host company's config and never varies per request.
func sidRegionsForEndpoint(endpoint string) map[string]struct{} {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil
	}

	host := parsed.Hostname()
	prefix, _, found := strings.Cut(host, ".")
	if !found {
		return nil
	}

	regions, ok := endpointRegions[prefix]
	if !ok {
		return nil
	}

	set := make(map[string]struct{}, len(regions))
	for _, region := range regions {
		set[region] = struct{}{}
	}
	return set
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	// The sid travels in the endpoint URL, so impressions can only share a
	// request when they share a sid. Grouping keeps the common case, where a
	// publisher configures one sid across its ad units, down to a single call.
	sidOrder := make([]string, 0, len(request.Imp))
	impsBySid := make(map[string][]openrtb2.Imp, len(request.Imp))

	for _, imp := range request.Imp {
		if err := validateImpMediaType(imp); err != nil {
			errs = append(errs, err)
			continue
		}

		sid, err := parseSid(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := a.validateSidRegion(sid, imp.ID); err != nil {
			errs = append(errs, err)
			continue
		}

		if _, seen := impsBySid[sid]; !seen {
			sidOrder = append(sidOrder, sid)
		}
		impsBySid[sid] = append(impsBySid[sid], imp)
	}

	requests := make([]*adapters.RequestData, 0, len(sidOrder))
	headers := makeHeaders(request)

	for _, sid := range sidOrder {
		requestData, err := a.makeRequest(request, impsBySid[sid], sid, headers)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, requestData)
	}

	return requests, errs
}

// validateImpMediaType drops an impression Tunnl cannot serve. Audio is not a
// declared capability, so forwarding an audio imp would invite a bid that
// MakeBids then has to reject: catching it here keeps the request honest and
// gives the publisher a clear reason.
func validateImpMediaType(imp openrtb2.Imp) error {
	if imp.Banner != nil || imp.Video != nil || imp.Native != nil {
		return nil
	}

	return &errortypes.BadInput{
		Message: fmt.Sprintf("imp %s: unsupported media type, Tunnl supports banner, video and native", imp.ID),
	}
}

// parseSid reads the sid out of imp.ext.bidder. The JSON schema already
// guarantees a non-empty string, so anything wrong here is a malformed request
// rather than a publisher misconfiguration.
func parseSid(imp openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext: %v", imp.ID, err),
		}
	}

	var impExt openrtb_ext.ExtImpTunnl
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext.bidder: %v", imp.ID, err),
		}
	}

	if impExt.SID == "" {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: missing required parameter sid", imp.ID),
		}
	}

	return impExt.SID, nil
}

// validateSidRegion rejects a sid whose region does not match the region the
// configured endpoint serves. Without this the mismatch is silent: the bid
// request reaches the wrong regional endpoint and its revenue is attributed to
// a region the traffic never came from.
func (a *adapter) validateSidRegion(sid, impID string) error {
	if len(a.sidRegions) == 0 {
		return nil
	}

	region, ok := sidRegion(sid)
	if !ok {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: sid %q is malformed, expected the form tunnl_<partner>_<region>_g", impID, sid),
		}
	}

	if _, ok := a.sidRegions[region]; !ok {
		accepted := make([]string, 0, len(a.sidRegions))
		for r := range a.sidRegions {
			accepted = append(accepted, r)
		}
		sort.Strings(accepted)

		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: sid %q targets region %q but this endpoint serves %s", impID, sid, region, strings.Join(accepted, ", ")),
		}
	}

	return nil
}

// sidRegion extracts the region from an underscore separated sid, where it is
// the second to last segment. It is read from the end because the segments
// before it may themselves contain underscores.
func sidRegion(sid string) (string, bool) {
	segments := strings.Split(sid, "_")
	if len(segments) < 4 {
		return "", false
	}
	return segments[len(segments)-2], true
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, imps []openrtb2.Imp, sid string, headers http.Header) (*adapters.RequestData, error) {
	endpoint, err := a.buildEndpointURL(sid)
	if err != nil {
		return nil, err
	}

	// A shallow copy is enough: only Imp is replaced, and imp elements are
	// already shallow copies owned by this adapter. Nothing else is mutated.
	reqCopy := *request
	reqCopy.Imp = imps

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

// buildEndpointURL resolves the sid into the host configured endpoint. The
// region is part of the sid and of the endpoint host, not something the adapter
// chooses.
func (a *adapter) buildEndpointURL(sid string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		SourceId: url.QueryEscape(sid),
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
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

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i], request.Imp)
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

// getMediaTypeForBid resolves a bid's media type from mtype where the response
// carries it, and falls back to the impression it bid on. The fallback matters
// because mtype only exists from ORTB 2.6 onwards, and a 2.5 response omits it
// entirely.
func getMediaTypeForBid(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
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
		// through to the impression and reach the publisher mislabelled.
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported media type audio for bid %q in imp %q, Tunnl supports banner, video and native", bid.ID, bid.ImpID),
		}
	}

	// Without mtype the impression is the only remaining signal. An impression
	// that declared several media types is ambiguous here, so such a response
	// needs mtype to be attributed correctly.
	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}

		switch {
		case imp.Banner != nil && imp.Video == nil && imp.Native == nil:
			return openrtb_ext.BidTypeBanner, nil
		case imp.Video != nil && imp.Banner == nil && imp.Native == nil:
			return openrtb_ext.BidTypeVideo, nil
		case imp.Native != nil && imp.Banner == nil && imp.Video == nil:
			return openrtb_ext.BidTypeNative, nil
		}

		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %q in imp %q is missing mtype, which is required to attribute a bid on a multi format impression", bid.ID, bid.ImpID),
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to determine media type for bid %q in imp %q", bid.ID, bid.ImpID),
	}
}

// makeHeaders forwards the device signals Tunnl uses for geo lookup and device
// detection. They matter most for app traffic, where the ad server has no
// browser request of its own to read them from.
func makeHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.6")

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Add("User-Agent", request.Device.UA)
		}
		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
		if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}

	return headers
}
