package anzuprebid

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Values of the zone bidder param. Each zone is one of Anzu's bid endpoints.
const (
	zone1 = "zone1"
	zone2 = "zone2"
)

// maxQueryStringLength caps queryString so the request URL stays well
// within the length servers and proxies accept.
const maxQueryStringLength = 2048

// queryStringPattern allows only characters that are valid in a URL query
// (RFC 3986), minus '#' and ';'. Without '#' the param can't end the query early,
// and since it always follows the '?', it can never change the host or path.
var queryStringPattern = regexp.MustCompile(`^[A-Za-z0-9._~!$&'()*+,=:@/?%-]+$`)

type adapter struct {
	zone1Endpoint string
	zone2Endpoint string
}

// extraInfo is the adapter's extra_info config: the URL of the zone2 endpoint. The
// zone1 endpoint is the adapter's main endpoint.
type extraInfo struct {
	Zone2Endpoint string `json:"zone2Endpoint"`
}

// Builder builds a new instance of the Anzu Prebid adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	if err := checkEndpoint("endpoint", config.Endpoint); err != nil {
		return nil, err
	}

	var info extraInfo
	if err := jsonutil.Unmarshal([]byte(config.ExtraAdapterInfo), &info); err != nil {
		return nil, fmt.Errorf("invalid extra_info: %v", err)
	}
	if err := checkEndpoint("extra_info zone2Endpoint", info.Zone2Endpoint); err != nil {
		return nil, err
	}

	return &adapter{zone1Endpoint: config.Endpoint, zone2Endpoint: info.Zone2Endpoint}, nil
}

// checkEndpoint makes sure an endpoint is an https URL without a query string,
// since the adapter appends the publisher's queryString as the query.
func checkEndpoint(name, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("%s must be an https URL, got %q", name, endpoint)
	}
	if strings.ContainsAny(endpoint, "?#") {
		return fmt.Errorf("%s must not include a query string or fragment, got %q", name, endpoint)
	}
	return nil
}

// MakeRequests sends one request per distinct endpoint URL, so imps share a
// request when they have the same zone and queryString.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var uris []string
	groups := make(map[string][]openrtb2.Imp)

	for _, imp := range request.Imp {
		uri, err := a.getEndpoint(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, exists := groups[uri]; !exists {
			uris = append(uris, uri)
		}
		groups[uri] = append(groups[uri], imp)
	}

	if len(uris) == 0 {
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requests := make([]*adapters.RequestData, 0, len(uris))
	for _, uri := range uris {
		requestCopy := *request
		requestCopy.Imp = groups[uri]

		body, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     uri,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
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

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(requestData.ImpIDs))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(request.Imp, seatBid.Bid[i])
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

// getEndpoint builds the request URL for an imp from its bidder params: the
// endpoint of the zone they select, followed by "?" and queryString.
func (a *adapter) getEndpoint(imp openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext for imp %s: %s", imp.ID, err)}
	}
	var anzuExt openrtb_ext.ExtImpAnzuPrebid
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &anzuExt); err != nil {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext.bidder for imp %s: %s", imp.ID, err)}
	}

	var base string
	switch anzuExt.Zone {
	case zone1:
		base = a.zone1Endpoint
	case zone2:
		base = a.zone2Endpoint
	default:
		return "", &errortypes.BadInput{Message: fmt.Sprintf("invalid zone %q for imp %s: must be %q or %q", anzuExt.Zone, imp.ID, zone1, zone2)}
	}

	query, err := normalizeQueryString(anzuExt.QueryString)
	if err != nil {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("invalid queryString %q for imp %s: %s", anzuExt.QueryString, imp.ID, err)}
	}
	return base + "?" + query, nil
}

// normalizeQueryString drops one leading '?' and checks that what remains is
// a well-formed URL query string.
func normalizeQueryString(raw string) (string, error) {
	if len(raw) > maxQueryStringLength {
		return "", fmt.Errorf("longer than %d characters", maxQueryStringLength)
	}
	query := strings.TrimPrefix(raw, "?")
	if !queryStringPattern.MatchString(query) {
		return "", errors.New("must be a URL query string such as a=1&b=2, without '#', ';' or spaces")
	}
	if _, err := url.ParseQuery(query); err != nil {
		return "", err
	}
	return query, nil
}

// getMediaTypeForBid uses bid.mtype when Anzu sets it. Without it, the media type
// is the format of the bid's imp when that imp offers exactly one format, and is
// otherwise detected from the bid's markup.
func getMediaTypeForBid(imps []openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		// No mtype: fall back to the imp's format or the markup, below.
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported bid.mtype %d for impression %s", bid.MType, bid.ImpID),
		}
	}

	i := slices.IndexFunc(imps, func(imp openrtb2.Imp) bool { return imp.ID == bid.ImpID })
	if i < 0 {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unable to find impression %s for bid", bid.ImpID),
		}
	}
	imp := imps[i]

	var types []openrtb_ext.BidType
	if imp.Banner != nil {
		types = append(types, openrtb_ext.BidTypeBanner)
	}
	if imp.Video != nil {
		types = append(types, openrtb_ext.BidTypeVideo)
	}
	if imp.Native != nil {
		types = append(types, openrtb_ext.BidTypeNative)
	}
	if len(types) == 1 {
		return types[0], nil
	}
	detected := detectMediaType(bid.AdM)
	if slices.Contains(types, detected) {
		return detected, nil
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("bid for impression %s has no mtype and its markup looks like %s, which the impression doesn't offer", bid.ImpID, detected),
	}
}

// detectMediaType works out a bid's media type from its markup: VAST XML is
// video, a native response JSON object is native, and anything else is banner.
func detectMediaType(adm string) openrtb_ext.BidType {
	markup := strings.TrimSpace(adm)
	if containsVASTTag(markup) {
		return openrtb_ext.BidTypeVideo
	}
	if strings.HasPrefix(markup, "{") {
		var fields struct {
			Native json.RawMessage `json:"native"`
			Assets json.RawMessage `json:"assets"`
		}
		if err := jsonutil.Unmarshal([]byte(markup), &fields); err == nil && (fields.Native != nil || fields.Assets != nil) {
			return openrtb_ext.BidTypeNative
		}
	}
	return openrtb_ext.BidTypeBanner
}

// containsVASTTag reports whether markup contains "<vast" in any letter case,
// without copying the markup.
func containsVASTTag(markup string) bool {
	const tag = "<vast"
	for i := strings.IndexByte(markup, '<'); i >= 0; {
		if len(markup)-i >= len(tag) && strings.EqualFold(markup[i:i+len(tag)], tag) {
			return true
		}
		next := strings.IndexByte(markup[i+1:], '<')
		if next < 0 {
			return false
		}
		i += next + 1
	}
	return false
}
