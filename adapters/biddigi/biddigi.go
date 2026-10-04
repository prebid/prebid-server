package biddigi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the BidDigi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

// MakeRequests forwards the OpenRTB request to BidDigi's authenticated server-to-server endpoint.
//
// Two things about this adapter are not the usual thin-passthrough shape, and both come from how
// BidDigi's endpoint authenticates and resolves supply:
//
//  1. THE SEAT KEY TRAVELS AS A HEADER, NEVER IN THE BODY. BidDigi issues one credential per
//     integration ("seat") and authenticates with `Authorization: Bearer <seatKey>`. The key is
//     therefore stripped out of imp.ext before the request is marshalled -- forwarding a
//     credential inside the bid request would put it in every debug dump, stored auction record
//     and log line that ever handles the request.
//
//  2. IMPS ARE GROUPED BY SEAT KEY. One HTTP request carries exactly one Authorization header, so
//     imps authenticating as different seats cannot share a call. Grouping rather than erroring
//     keeps a mixed request working: each group becomes its own call, which is what a host running
//     several BidDigi integrations behind one PBS instance actually needs. Groups are emitted in a
//     deterministic (sorted) order so the request sequence is stable for tests and for debugging.
//
// Publisher resolution is deliberately NOT a parameter. BidDigi maps the authenticated seat plus
// site.domain / app.bundle to one of its publishers server-side, so there is no internal publisher
// id for a host to know, mistype, or spoof.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	// Preserve first-seen order of keys, then sort, so output ordering never depends on Go's
	// randomised map iteration.
	impsByKey := make(map[string][]openrtb2.Imp)

	for _, imp := range request.Imp {
		params, err := parseImpExt(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		cleaned, err := stripBidderExt(imp, params.PlacementID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		impsByKey[params.SeatKey] = append(impsByKey[params.SeatKey], cleaned)
	}

	if len(impsByKey) == 0 {
		return nil, errs
	}

	keys := make([]string, 0, len(impsByKey))
	for k := range impsByKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	requestData := make([]*adapters.RequestData, 0, len(keys))
	for _, key := range keys {
		imps := impsByKey[key]

		// Copy the request per group rather than mutating the caller's: PBS hands every adapter
		// the same *BidRequest, and writing Imp on it would corrupt the request other bidders see.
		outgoing := *request
		outgoing.Imp = imps

		body, err := jsonutil.Marshal(&outgoing)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to marshal openrtb request: %w", err))
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		headers.Add("Authorization", "Bearer "+key)
		headers.Add("X-Openrtb-Version", "2.6")

		requestData = append(requestData, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(imps),
		})
	}

	return requestData, errs
}

// parseImpExt pulls and validates this adapter's params off a single imp.
//
// seatKey is required here as well as in static/bidder-params/biddigi.json. The schema is the
// real gate, but a missing key would otherwise surface as an unauthenticated call and a 401 from
// BidDigi, which is a much worse error message than naming the imp.
func parseImpExt(imp openrtb2.Imp) (*openrtb_ext.ExtImpBiddigi, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse imp.ext: %s", imp.ID, err.Error()),
		}
	}

	var params openrtb_ext.ExtImpBiddigi
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse biddigi params: %s", imp.ID, err.Error()),
		}
	}

	if params.SeatKey == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: missing required biddigi param \"seatKey\"", imp.ID),
		}
	}

	return &params, nil
}

// stripBidderExt removes the params object (which holds the credential) from imp.ext and, when a
// placementId was supplied, republishes just that field under imp.ext.biddigi.
//
// Everything else in imp.ext is preserved rather than discarded -- ext.prebid, ext.gpid and
// ext.data belong to PBS and to other modules, and an adapter that flattens them breaks features
// it has nothing to do with. When nothing is left, imp.ext is dropped entirely rather than sent
// as an empty object.
//
// The values are held as json.RawMessage, NOT decoded into interface{}. Decoding a JSON number
// into interface{} yields a float64, and re-marshalling that float64 does not reproduce the bytes
// that arrived: an integer above 2^53 is silently rounded (9007199254740993 comes back out as
// ...992, and a uint64 deal id loses its last three digits entirely), while 1.10 becomes 1.1 and
// 1e2 becomes 100. This function's whole job is to remove one key and pass the rest through
// untouched, so every key except "bidder" is carried as the exact bytes the caller sent and is
// never parsed at all.
func stripBidderExt(imp openrtb2.Imp, placementID string) (openrtb2.Imp, error) {
	var ext map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return imp, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse imp.ext: %s", imp.ID, err.Error()),
		}
	}

	delete(ext, "bidder")

	if placementID != "" {
		encoded, err := jsonutil.Marshal(map[string]string{"placementId": placementID})
		if err != nil {
			return imp, fmt.Errorf("imp %s: unable to marshal imp.ext.biddigi: %w", imp.ID, err)
		}
		ext["biddigi"] = encoded
	}

	if len(ext) == 0 {
		imp.Ext = nil
		return imp, nil
	}

	encoded, err := jsonutil.Marshal(ext)
	if err != nil {
		return imp, fmt.Errorf("imp %s: unable to marshal imp.ext: %w", imp.ID, err)
	}
	imp.Ext = encoded
	return imp, nil
}

// MakeBids unpacks BidDigi's OpenRTB BidResponse.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	// BidDigi answers in USD and says so on every response, so this normally agrees with the
	// "USD" default. It is read rather than assumed anyway: the exchange retains an INR fallback
	// for the case where no FX rate is available for an auction, and silently treating one of
	// those prices as USD would overstate the bid by roughly ninety times. Reading the field the
	// server actually sent costs nothing and removes the entire class of error.
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidderResponse, errs
}

// getMediaTypeForBid reads the media type off bid.mtype only.
//
// BidDigi always sets mtype (its auction core assigns it when it picks the winner), so there is no
// need to fall back to matching the bid against the request's imps -- and inferring a type from
// the imp would be worse than erroring, because a multi-format imp makes the inference a guess.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}
}
