package epom_as

import (
	"encoding/json"
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
	"github.com/prebid/prebid-server/v4/util/urlutil"
)

// adapter talks to the Epom Ad Server, the sell side of the Epom platform.
// It is a different product from the `epom` adapter, which is the Epom DSP:
// the DSP buys impressions, this one sells a publisher's own inventory.
//
// Epom is white-label, so every network serves from its own domain and the
// host arrives per impression in imp.ext.bidder.host rather than from config.
type adapter struct {
	endpointTemplate *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{endpointTemplate: endpointTemplate}, nil
}

// MakeRequests emits one request per host, carrying every impression addressed
// to that host.
//
// Keeping a host's impressions together is a requirement, not an optimisation:
// the ad server decides a page as a unit, so its roadblock and
// one-campaign-per-page rules only hold when every slot is resolved in the same
// auction. Splitting per impression would make those rules race each other.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	// Preserve the order hosts were first seen so the emitted requests are
	// deterministic — Go map iteration is not.
	hostOrder := make([]string, 0, len(request.Imp))
	impsByHost := make(map[string][]openrtb2.Imp, len(request.Imp))

	for _, imp := range request.Imp {
		impExt, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// The placement travels as imp.tagid so the wire format is identical to
		// the one the Prebid.js adapter sends, and the ad server has a single
		// place to read it from.
		imp.TagID = impExt.PlacementKey

		if _, seen := impsByHost[impExt.Host]; !seen {
			hostOrder = append(hostOrder, impExt.Host)
		}
		impsByHost[impExt.Host] = append(impsByHost[impExt.Host], imp)
	}

	if len(impsByHost) == 0 {
		return nil, errs
	}

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	requests := make([]*adapters.RequestData, 0, len(impsByHost))
	for _, host := range hostOrder {
		imps := impsByHost[host]

		url, err := macros.ResolveMacros(a.endpointTemplate, macros.EndpointTemplateParams{Host: host})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		hostRequest := *request
		hostRequest.Imp = imps

		body, err := json.Marshal(hostRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     url,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(imps),
		})
	}

	return requests, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	result := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if bidResponse.Cur != "" {
		result.Currency = bidResponse.Cur
	}

	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			result.Bids = append(result.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return result, errs
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpEpomAs, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: missing bidder ext: %s", imp.ID, err.Error()),
		}
	}

	var impExt openrtb_ext.ExtImpEpomAs
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: cannot resolve host or placementKey: %s", imp.ID, err.Error()),
		}
	}

	// The host is the only publisher-controlled part of the outbound URL, so it
	// must be a bare hostname; anything carrying a path, query or userinfo could
	// redirect the bid request to an unintended destination.
	if !urlutil.IsSafeHost(impExt.Host) {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid host", imp.ID),
		}
	}
	if impExt.PlacementKey == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: missing placementKey", imp.ID),
		}
	}

	return &impExt, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case 0:
		// The adapter only declares banner capability, so an omitted mtype can
		// only mean banner. Older ad server builds do not populate the field.
		return openrtb_ext.BidTypeBanner, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ImpID),
		}
	}
}
