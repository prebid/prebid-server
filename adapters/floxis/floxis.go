package floxis

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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

const (
	defaultRegion  = "us-e"
	defaultPartner = "floxis"
)

// hostLabelRegex mirrors Prebid.js HOST_LABEL_REGEX: region/partner are interpolated into the
// request host, so they must be valid DNS labels — a value carrying URL delimiters could
// otherwise rewrite the request origin. Case-insensitive, single label, max 63 chars.
var hostLabelRegex = regexp.MustCompile(`(?i)^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func isValidHostLabel(s string) bool {
	return hostLabelRegex.MatchString(s)
}

// resolveBidHost mirrors Prebid.js getBidHost: empty region/partner default to us-e/floxis,
// each must be a valid host label, and floxis itself carries no partner prefix. Routing
// (which region maps to which datacenter) is handled at DNS/LB level, not here.
func resolveBidHost(region, partner string) (string, error) {
	if region == "" {
		region = defaultRegion
	}
	if partner == "" {
		partner = defaultPartner
	}
	if !isValidHostLabel(region) || !isValidHostLabel(partner) {
		return "", &errortypes.BadInput{Message: fmt.Sprintf(
			"invalid region %q or partner %q; both must be valid host labels", region, partner)}
	}
	if partner == defaultPartner {
		return region + ".floxis.tech", nil
	}
	return partner + "-" + region + ".floxis.tech", nil
}

// Builder builds a new instance of the Floxis adapter for the given bidder with the given
// config. config.Endpoint is the {{.Host}} template; the host is filled per-request from the
// bidder's region/partner params (validated as host labels — never raw request-supplied hostnames).
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	tmpl, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{endpoint: tmpl}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "no impressions in the bid request"}}
	}

	ext, err := parseImpExt(request.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	// All imps in one call route under imp[0]'s seat/host. Reject a request whose imps carry
	// differing floxis seats, regions, or partners rather than silently mis-routing imp[1..].
	for i := 1; i < len(request.Imp); i++ {
		impExt, err := parseImpExt(request.Imp[i])
		if err != nil {
			return nil, []error{err}
		}
		if impExt.Seat != ext.Seat || impExt.Region != ext.Region || impExt.Partner != ext.Partner {
			return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf(
				"imp %s seat/region/partner (%q/%q/%q) differs from imp %s (%q/%q/%q); split into separate requests",
				request.Imp[i].ID, impExt.Seat, impExt.Region, impExt.Partner,
				request.Imp[0].ID, ext.Seat, ext.Region, ext.Partner)}}
		}
	}

	host, err := resolveBidHost(ext.Region, ext.Partner)
	if err != nil {
		return nil, []error{err}
	}
	endpoint, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{Host: host})
	if err != nil {
		return nil, []error{err}
	}
	// seat is not a standard endpoint macro, so it is appended in code as a query param.
	uri := fmt.Sprintf("%s?seat=%s", endpoint, url.QueryEscape(ext.Seat))

	// The request body is forwarded unchanged; no caller-owned struct is mutated, so
	// copy-on-write is satisfied by construction.
	body, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
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
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, typeErr := getMediaTypeForBid(request.Imp, seatBid.Bid[i])
			if typeErr != nil {
				errs = append(errs, typeErr)
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

func parseImpExt(imp openrtb2.Imp) (openrtb_ext.ExtImpFloxis, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpFloxis{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext for imp %s: %s", imp.ID, err)}
	}
	var floxisExt openrtb_ext.ExtImpFloxis
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &floxisExt); err != nil {
		return openrtb_ext.ExtImpFloxis{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext.bidder for imp %s: %s", imp.ID, err)}
	}
	return floxisExt, nil
}

// getMediaTypeForBid resolves the bid's media type. When bid.mtype (OpenRTB 2.6) is set it
// is treated as authoritative. When unset, a single-format imp's media type is used;
// multi-format imps without mtype cannot be disambiguated and return an error.
func getMediaTypeForBid(imps []openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
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
	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}
		formats := 0
		var resolved openrtb_ext.BidType
		if imp.Banner != nil {
			formats++
			resolved = openrtb_ext.BidTypeBanner
		}
		if imp.Video != nil {
			formats++
			resolved = openrtb_ext.BidTypeVideo
		}
		if imp.Audio != nil {
			formats++
			resolved = openrtb_ext.BidTypeAudio
		}
		if imp.Native != nil {
			formats++
			resolved = openrtb_ext.BidTypeNative
		}
		switch {
		case formats == 1:
			return resolved, nil
		case formats > 1:
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("bid for multi-format imp %s requires bid.mtype to disambiguate", bid.ImpID),
			}
		default:
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("unable to resolve media type for impression %s", bid.ImpID),
			}
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to find impression %s for bid", bid.ImpID),
	}
}
