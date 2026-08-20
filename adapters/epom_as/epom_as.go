package epom_as

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

		applyBidFloor(&imp, impExt)

		if err := enrichImpExt(&imp, impExt); err != nil {
			errs = append(errs, err)
			continue
		}

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

		body, err := jsonutil.Marshal(hostRequest)
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
		return nil, []error{&errortypes.BadServerResponse{Message: err.Error()}}
	}

	impsByID := make(map[string]*openrtb2.Imp, len(request.Imp))
	for i := range request.Imp {
		impsByID[request.Imp[i].ID] = &request.Imp[i]
	}

	result := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if bidResponse.Cur != "" {
		result.Currency = bidResponse.Cur
	}

	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i], impsByID)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			result.Bids = append(result.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return result, errs
}

// applyBidFloor fills the floor from the bidder params only when the request
// carries none of its own, so a floor already resolved by the Price Floors
// module — or set by the publisher on the impression — always wins.
func applyBidFloor(imp *openrtb2.Imp, impExt *openrtb_ext.ExtImpEpomAs) {
	if imp.BidFloor != 0 || impExt.BidFloor <= 0 {
		return
	}
	imp.BidFloor = impExt.BidFloor
	if impExt.BidFloorCur != "" {
		imp.BidFloorCur = impExt.BidFloorCur
	} else {
		imp.BidFloorCur = "USD"
	}
}

// enrichImpExt moves the Epom-specific params out of imp.ext.bidder and into the
// shape the ad server reads: channel under our own namespace, custom parameters
// merged into imp.ext.data, the standard first-party-data home, so that data
// contributed by RTD modules lands in the same object.
func enrichImpExt(imp *openrtb2.Imp, impExt *openrtb_ext.ExtImpEpomAs) error {
	if impExt.Channel == "" && len(impExt.CustomParams) == 0 {
		return nil
	}

	ext := map[string]json.RawMessage{}
	if len(imp.Ext) > 0 {
		if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("imp %s: malformed ext: %s", imp.ID, err.Error())}
		}
	}

	if impExt.Channel != "" {
		namespace, err := jsonutil.Marshal(map[string]string{"channel": impExt.Channel})
		if err != nil {
			return err
		}
		ext[string(openrtb_ext.BidderEpomAs)] = namespace
	}

	if merged := mergeCustomParams(ext["data"], impExt.CustomParams); merged != nil {
		ext["data"] = merged
	}

	encoded, err := jsonutil.Marshal(ext)
	if err != nil {
		return err
	}
	imp.Ext = encoded
	return nil
}

// mergeCustomParams folds the custom params into any existing imp.ext.data.
// Existing keys win — data already on the imp came from the publisher's own
// first-party configuration. Values are stringified because the ad server reads
// custom targeting as text; the schema already restricts them to scalars, so a
// value this cannot stringify only reaches here through a host that skipped
// param validation, and is skipped rather than written as a Go rendering of a
// map. Nothing is dropped for size, which is what keeps the marshalled imp.ext
// independent of Go's randomised map iteration order.
func mergeCustomParams(existing json.RawMessage, params map[string]interface{}) json.RawMessage {
	out := map[string]interface{}{}
	for key, value := range params {
		if asString, ok := scalarToString(value); ok {
			out[key] = asString
		}
	}
	if len(out) == 0 {
		return nil
	}

	if len(existing) > 0 {
		current := map[string]interface{}{}
		if err := jsonutil.Unmarshal(existing, &current); err == nil {
			for key, value := range current {
				out[key] = value
			}
		}
	}

	encoded, err := jsonutil.Marshal(out)
	if err != nil {
		return nil
	}
	return encoded
}

func scalarToString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case json.Number:
		return v.String(), true
	default:
		return "", false
	}
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

// getMediaTypeForBid resolves the bid's media type from mtype, falling back to
// the impression the bid answers when the ad server omits it. Nothing is
// assumed: a bid that matches no banner impression is a defect on the wire, and
// rendering it as a banner would hide that.
func getMediaTypeForBid(bid openrtb2.Bid, impsByID map[string]*openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case 0:
		if imp, ok := impsByID[bid.ImpID]; ok && imp.Banner != nil {
			return openrtb_ext.BidTypeBanner, nil
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unresolved mtype for bid %s: no banner imp %s", bid.ID, bid.ImpID),
		}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}
}
