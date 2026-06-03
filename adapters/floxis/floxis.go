package floxis

import (
	"fmt"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct{}

// regionHosts is a fixed allowlist mapping the bidder's region param to a Floxis RTB
// host. Routing is never derived from request-supplied hostnames; an unknown or empty
// region falls back to us-e. This satisfies PBS's "no fully dynamic hostnames" rule.
var regionHosts = map[string]string{
	"us-e": "rtb-us-e.floxis.tech",
	"eu":   "rtb-eu.floxis.tech",
	"apac": "rtb-apac.floxis.tech",
}

const defaultRegion = "us-e"

// resolveHost returns the Floxis RTB host for the given region, defaulting to us-e for
// unknown or empty regions.
func resolveHost(region string) string {
	if host, ok := regionHosts[region]; ok {
		return host
	}
	return regionHosts[defaultRegion]
}

// Builder builds a new instance of the Floxis adapter for the given bidder with the given
// config. The endpoint is resolved per-request from the bidder's region param via a fixed
// host allowlist, so config.Endpoint is intentionally unused.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "no impressions in the bid request"}}
	}

	ext, err := parseImpExt(request.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	host := resolveHost(ext.Region)
	uri := fmt.Sprintf("https://%s/pbs?seat=%s", host, url.QueryEscape(ext.Seat))

	// The request body is forwarded unchanged; no caller-owned struct is mutated, so
	// copy-on-write is satisfied by construction.
	body, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method: "POST",
		Uri:    uri,
		Body:   body,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
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
