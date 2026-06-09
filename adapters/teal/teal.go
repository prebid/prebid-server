package teal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"unicode"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	msgAccountValidation   = "account parameter failed validation"
	msgPlacementValidation = "placement parameter failed validation"
	msgImpExtParseFmt      = "Error parsing imp.ext for impression %s"
	msgMixedAccountFmt     = "mixed-account requests are not supported: imp %q account %q differs from request account %q"
)

// adapter is the Teal openrtb2 bidder.
type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Teal adapter for the given bidder with
// the given config.
func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

// MakeRequests transforms the openrtb2.BidRequest into a single Teal-bound HTTP
// request body:
//
//  1. Each imp's bidder-slot is decoded into ExtImpTeal and validated for
//     non-blank account and (when present) non-blank placement.
//  2. Failed imps are dropped; their parse / validation errors are collected.
//  3. The first surviving imp's account is propagated to Site.Publisher.ID and
//     App.Publisher.ID. Later imps must use that same account; an imp with a
//     divergent account is rejected (BadInput) and dropped.
//  4. Each surviving imp gets imp.ext.prebid.storedrequest.id = placement when
//     placement is set.
//  5. Request.Ext.bids is stamped with {"pbs": 1}.
//
// If no imp survives validation, returns (nil, errs) without dispatching.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, nil
	}

	modifiedImps := make([]openrtb2.Imp, 0, len(request.Imp))
	var errs []error
	var account string

	for i := range request.Imp {
		imp := request.Imp[i]
		ext, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			continue
		}
		if err := validateImpExt(ext); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			continue
		}

		// All imps in a request must agree on account, which drives the single
		// request-level publisher.id. The first valid imp establishes it; a later
		// imp with a different account is rejected and dropped.
		if account == "" {
			account = ext.Account
		} else if ext.Account != account {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf(msgMixedAccountFmt, imp.ID, ext.Account, account),
			})
			continue
		}

		modified, err := modifyImp(&imp, ext.Placement)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			continue
		}
		modifiedImps = append(modifiedImps, *modified)
	}

	if len(modifiedImps) == 0 {
		return nil, errs
	}

	modifiedRequest, err := modifyBidRequest(request, account, modifiedImps)
	if err != nil {
		return nil, append(errs, err)
	}

	body, err := jsonutil.Marshal(modifiedRequest)
	if err != nil {
		return nil, append(errs, err)
	}

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: standardHeaders(),
		ImpIDs:  openrtb_ext.GetImpIDs(modifiedRequest.Imp),
	}}, errs
}

// parseImpExt decodes imp.ext.bidder into ExtImpTeal.
func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpTeal, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, fmt.Errorf(msgImpExtParseFmt, imp.ID)
	}
	var ext openrtb_ext.ExtImpTeal
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &ext); err != nil {
		return nil, fmt.Errorf(msgImpExtParseFmt, imp.ID)
	}
	return &ext, nil
}

// validateImpExt requires a non-blank account and, when placement is present
// (non-nil), a non-blank placement. Absent placement is allowed.
func validateImpExt(ext *openrtb_ext.ExtImpTeal) error {
	if isBlank(ext.Account) {
		return errors.New(msgAccountValidation)
	}
	if ext.Placement != nil && isBlank(*ext.Placement) {
		return errors.New(msgPlacementValidation)
	}
	return nil
}

// isBlank returns true if s is empty or contains only Unicode whitespace runes.
func isBlank(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// modifyImp returns a copy of imp with imp.ext.prebid.storedrequest.id set to
// *placement. Returns the imp unchanged when placement is nil. Existing
// prebid or storedrequest sub-keys that are not JSON objects are tolerated by
// replacing them with fresh objects.
func modifyImp(imp *openrtb2.Imp, placement *string) (*openrtb2.Imp, error) {
	if placement == nil {
		return imp, nil
	}

	ext, err := decodeJSONObject(imp.Ext)
	if err != nil {
		return nil, fmt.Errorf(msgImpExtParseFmt, imp.ID)
	}

	prebid := decodeOrEmptyObject(ext["prebid"])
	storedRequest := decodeOrEmptyObject(prebid["storedrequest"])

	placementJSON, err := jsonutil.Marshal(*placement)
	if err != nil {
		return nil, err
	}
	storedRequest["id"] = placementJSON

	storedRequestJSON, err := jsonutil.Marshal(storedRequest)
	if err != nil {
		return nil, err
	}
	prebid["storedrequest"] = storedRequestJSON

	prebidJSON, err := jsonutil.Marshal(prebid)
	if err != nil {
		return nil, err
	}
	ext["prebid"] = prebidJSON

	extJSON, err := jsonutil.Marshal(ext)
	if err != nil {
		return nil, err
	}

	modified := *imp
	modified.Ext = extJSON
	return &modified, nil
}

// decodeOrEmptyObject decodes raw as a JSON object map. Returns an empty map
// when raw is absent / "null" / not a JSON object. The returned map is NEVER
// nil — it is always safe to assign into.
func decodeOrEmptyObject(raw json.RawMessage) map[string]json.RawMessage {
	out, _ := decodeJSONObject(raw)
	return out
}

// decodeJSONObject decodes raw into a JSON object map. Treats absent input
// AND the JSON literal `null` as "empty object". Returns the parse error
// untouched on invalid JSON or non-object root types so callers can surface a
// meaningful failure. The returned map is NEVER nil, even on error — callers
// can safely assign into it.
func decodeJSONObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return make(map[string]json.RawMessage), nil
	}
	var parsed map[string]json.RawMessage
	if err := jsonutil.Unmarshal(raw, &parsed); err != nil {
		return make(map[string]json.RawMessage), err
	}
	if parsed == nil {
		return make(map[string]json.RawMessage), nil
	}
	return parsed, nil
}

// modifyBidRequest applies the request-level mutations:
//
//   - Site.Publisher.ID is overwritten with account when site is non-nil
//   - App.Publisher.ID is overwritten with account when app is non-nil
//   - Request.Ext.bids is stamped with {"pbs":1}
//
// Returns a value-copy with mutated fields; the caller's request is untouched.
func modifyBidRequest(request *openrtb2.BidRequest, account string, modifiedImps []openrtb2.Imp) (*openrtb2.BidRequest, error) {
	modified := *request
	modified.Imp = modifiedImps

	if request.Site != nil {
		site := *request.Site
		site.Publisher = clonePublisherWithID(site.Publisher, account)
		modified.Site = &site
	}
	if request.App != nil {
		app := *request.App
		app.Publisher = clonePublisherWithID(app.Publisher, account)
		modified.App = &app
	}

	extJSON, err := mergeBidsPBSFlag(request.Ext)
	if err != nil {
		return nil, err
	}
	modified.Ext = extJSON
	return &modified, nil
}

// clonePublisherWithID returns a copy of publisher with ID overwritten,
// creating a fresh Publisher when publisher is nil.
func clonePublisherWithID(publisher *openrtb2.Publisher, id string) *openrtb2.Publisher {
	if publisher == nil {
		return &openrtb2.Publisher{ID: id}
	}
	pub := *publisher
	pub.ID = id
	return &pub
}

// mergeBidsPBSFlag returns existingExt with the "bids" property set to
// {"pbs":1}; other ext fields are preserved. The "pbs":1 marker is a Teal-side
// reporting/billing signal — it tells Teal's exchange the request is being
// routed via prebid-server, distinguishing it from direct integrations.
func mergeBidsPBSFlag(existingExt json.RawMessage) (json.RawMessage, error) {
	ext, err := decodeJSONObject(existingExt)
	if err != nil {
		return nil, fmt.Errorf("teal: failed parsing request.ext: %w", err)
	}
	ext["bids"] = json.RawMessage(`{"pbs":1}`)
	return jsonutil.Marshal(ext)
}

// MakeBids parses the Teal bid response body and packages the bids into a
// BidderResponse. Status handling follows the canonical adapters helpers:
// 204 → no-content shortcut, 4xx/5xx → BadServerResponse, 200 → parse body.
// Bids whose media type cannot be resolved from the matching imp are skipped
// and their error collected, rather than silently mis-typed as banner.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: err.Error()}}
	}

	impsByID := make(map[string]openrtb2.Imp, len(request.Imp))
	for _, imp := range request.Imp {
		impsByID[imp.ID] = imp
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if bidResponse.Cur != "" {
		bidderResponse.Currency = bidResponse.Cur
	}
	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getBidType(bid, impsByID)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}
	return bidderResponse, errs
}

// getBidType resolves the bid's media type from its matching impression, looked
// up by ID via impsByID (built once in MakeBids to avoid a per-bid scan of
// request.Imp). Priority order is banner > video > native.
//
// When no imp matches the bid's ImpID, or the matching imp declares no
// recognized media type, getBidType returns an error so MakeBids can skip the
// bid and surface the problem in logs rather than silently mis-typing it as
// banner.
func getBidType(bid *openrtb2.Bid, impsByID map[string]openrtb2.Imp) (openrtb_ext.BidType, error) {
	if imp, ok := impsByID[bid.ImpID]; ok {
		switch {
		case imp.Banner != nil:
			return openrtb_ext.BidTypeBanner, nil
		case imp.Video != nil:
			return openrtb_ext.BidTypeVideo, nil
		case imp.Native != nil:
			return openrtb_ext.BidTypeNative, nil
		}
	}
	return "", fmt.Errorf("failed to determine bid type for imp %q", bid.ImpID)
}

// standardHeaders returns the headers Teal expects on every outbound request.
func standardHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return headers
}
