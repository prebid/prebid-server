package optima

import (
	"encoding/json"
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

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Optima adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{endpoint: endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	// Optima only serves in-app traffic. Any request carrying a site object is
	// rejected by the exchange with a 400, so drop it here as bad input
	// rather than sending it and burning a round trip.
	if request.Site != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "optima: site traffic is not supported, request dropped",
		}}
	}
	if request.App == nil {
		return nil, []error{&errortypes.BadInput{
			Message: "optima: request.app is required",
		}}
	}
	// The exchange identifies in-app inventory by bundle and answers 400
	// missing_app_bundle without it, so fail here instead of burning a round trip.
	if request.App.Bundle == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "optima: request.app.bundle is required",
		}}
	}
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "optima: request has no impressions",
		}}
	}
	// Optima clears first price (at 0/1) or second price (at 2) and answers 400
	// unsupported_auction_type for anything else, so a fixed-price or
	// exchange-specific auction type cannot be served. Reject it here rather than
	// silently reselling the impression under different clearing rules.
	if request.AT != 0 && request.AT != 1 && request.AT != 2 {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("optima: auction type %d is not supported; only first price (1) and second price (2) are served", request.AT),
		}}
	}

	// Work on a shallow copy so we never mutate the request Prebid Server
	// shares with other bidders.
	reqCopy := *request

	// Optima is USD-only: it rejects the whole request with 400
	// unsupported_request_currency when request.cur lacks USD. Every imp floor is
	// converted to USD below and the exchange bids in USD, so advertise USD only.
	// Prebid Server converts the USD bids back using the original request.cur.
	reqCopy.Cur = []string{"USD"}

	var errs []error

	// The request is OpenRTB 2.6 (optima.yaml declares version 2.6, so PBS does not
	// down-convert). Current Optima builds read CCPA opt-out from regs.us_privacy, but
	// earlier ones read only the 2.5 location regs.ext.us_privacy and would drop the
	// opt-out silently, so mirror the 2.6 value into it as well.
	if regs, err := mirrorUSPrivacy(request.Regs); err != nil {
		errs = append(errs, err)
	} else {
		reqCopy.Regs = regs
	}

	// supply_id is an imp-level param and selects the exchange endpoint (and with
	// it the supply source's margin, floors and billing), so imps carrying
	// different keys must not share a request. Group them by key, keeping the
	// order in which each key first appears so the output is deterministic.
	groups := make(map[string]*impGroup)
	var order []string
	for _, imp := range request.Imp {
		mapped, err := mapImp(imp, reqInfo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		id, err := supplyID(mapped)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		group, ok := groups[id]
		if !ok {
			uri, err := a.endpointFor(id)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			group = &impGroup{uri: uri}
			groups[id] = group
			order = append(order, id)
		}
		group.imps = append(group.imps, mapped)
	}
	if len(order) == 0 {
		errs = append(errs, &errortypes.BadInput{
			Message: "optima: no valid impressions in request",
		})
		return nil, errs
	}

	requests := make([]*adapters.RequestData, 0, len(order))
	for _, id := range order {
		group := groups[id]
		groupReq := reqCopy
		groupReq.Imp = group.imps
		reqJSON, err := jsonutil.Marshal(groupReq)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     group.uri,
			Body:    reqJSON,
			Headers: requestHeaders(),
			ImpIDs:  openrtb_ext.GetImpIDs(group.imps),
		})
	}
	return requests, errs
}

// impGroup is the set of imps that share one supply key and so one request.
type impGroup struct {
	uri  string
	imps []openrtb2.Imp
}

// endpointFor resolves the exchange endpoint for a supply key. The key comes
// from publisher-controlled imp params, so it is escaped as a single path
// segment: otherwise "/", ".." or "?" in it would rewrite the request path or
// query and could steer the request to other routes on the exchange host.
func (a *adapter) endpointFor(supplyID string) (string, error) {
	return macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{
		SupplyId: url.PathEscape(supplyID),
	})
}

// requestHeaders returns a fresh header set per request so no map is shared
// between the requests built from one bid request.
func requestHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.6")
	return headers
}

// supplyID pulls the Optima supply key out of imp.ext.bidder.
func supplyID(imp openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("optima: imp %s has invalid ext: %v", imp.ID, err),
		}
	}
	var optimaExt openrtb_ext.ExtImpOptima
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &optimaExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("optima: imp %s has invalid ext.bidder: %v", imp.ID, err),
		}
	}
	if optimaExt.SupplyId == "" {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("optima: imp %s is missing required param supply_id", imp.ID),
		}
	}
	return optimaExt.SupplyId, nil
}

// mirrorUSPrivacy copies a top-level regs.us_privacy (OpenRTB 2.6) into
// regs.ext.us_privacy (OpenRTB 2.5), the only CCPA location that earlier Optima
// builds read. An existing regs.ext.us_privacy is never overwritten. Returns the
// original regs unchanged when there is nothing to mirror.
func mirrorUSPrivacy(regs *openrtb2.Regs) (*openrtb2.Regs, error) {
	if regs == nil || regs.USPrivacy == "" {
		return regs, nil
	}

	ext := map[string]json.RawMessage{}
	if len(regs.Ext) > 0 {
		if err := jsonutil.Unmarshal(regs.Ext, &ext); err != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("optima: invalid regs.ext: %v", err),
			}
		}
	}
	if _, ok := ext["us_privacy"]; ok {
		return regs, nil
	}

	usPrivacy, err := jsonutil.Marshal(regs.USPrivacy)
	if err != nil {
		return nil, err
	}
	ext["us_privacy"] = usPrivacy
	newExt, err := jsonutil.Marshal(ext)
	if err != nil {
		return nil, err
	}

	regsCopy := *regs
	regsCopy.Ext = newExt
	return &regsCopy, nil
}

// mapImp validates and normalizes a single imp for the Optima contract: Optima serves
// banner, interstitial and video (incl. rewarded), so an imp needs a banner or a
// video object (native/audio imps would otherwise cause a 400 unsupported_imp).
// An imp carrying both is passed on as is: Optima treats it as video and the banner
// is a companion. For a multi-size banner slot that only specifies
// imp.banner.format, the first format is flattened into w/h since Optima reads the
// top-level dimensions. The floor is converted to USD, the only currency Optima
// accepts.
func mapImp(imp openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) (openrtb2.Imp, error) {
	if imp.Banner == nil && imp.Video == nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: fmt.Sprintf("optima: imp %s has no banner or video object; only banner, interstitial and video imps are supported", imp.ID),
		}
	}
	// Optima requires video.mimes and fails the whole request with 400
	// missing_video_mimes without it. Prebid Server rejects such an imp before the
	// adapter runs, so this only guards paths that bypass that validation; dropping
	// the one imp keeps the rest of the request sellable.
	if imp.Video != nil && len(imp.Video.MIMEs) == 0 {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: fmt.Sprintf("optima: imp %s has a video object without mimes", imp.ID),
		}
	}

	// imp is a copy of the request's imp, so the floor fields can be set directly.
	if err := floorToUSD(&imp, reqInfo); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: fmt.Sprintf("optima: imp %s: cannot convert bid floor to USD: %v", imp.ID, err),
		}
	}

	if imp.Banner != nil {
		imp.Banner = flattenBannerFormat(imp.Banner)
	}
	return imp, nil
}

// floorToUSD rewrites imp.bidfloor/bidfloorcur into USD. Optima fails the whole
// request with 400 unsupported_floor_currency if any imp names another currency,
// and it compares the code case-sensitively, so "usd" is normalized as well. A
// currency code without an amount has nothing to convert and is just set to USD.
func floorToUSD(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloorCur == "" || imp.BidFloorCur == "USD" {
		return nil
	}
	if imp.BidFloor > 0 && !strings.EqualFold(imp.BidFloorCur, "USD") {
		converted, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return err
		}
		imp.BidFloor = converted
	}
	imp.BidFloorCur = "USD"
	return nil
}

// flattenBannerFormat fills banner.w/h from banner.format when the banner has no
// complete top-level size, since Optima reads only w/h. Entries with a zero
// dimension are unusable and skipped. When w or h is already set, the first entry
// consistent with it is preferred so an explicit dimension is kept; otherwise the
// first complete entry is used. Either way w and h always come from one entry, so
// the result is a size the publisher actually offered. Returns the original
// banner unchanged when it already has a size or no usable entry exists.
func flattenBannerFormat(banner *openrtb2.Banner) *openrtb2.Banner {
	if isSet(banner.W) && isSet(banner.H) {
		return banner
	}

	var chosen *openrtb2.Format
	for i := range banner.Format {
		f := &banner.Format[i]
		if f.W <= 0 || f.H <= 0 {
			continue
		}
		if agrees(banner.W, f.W) && agrees(banner.H, f.H) {
			chosen = f
			break
		}
		if chosen == nil {
			chosen = f // fallback if no entry agrees with the explicit dimension
		}
	}
	if chosen == nil {
		return banner
	}

	bannerCopy := *banner
	w, h := chosen.W, chosen.H
	bannerCopy.W = &w
	bannerCopy.H = &h
	return &bannerCopy
}

// isSet reports whether a banner dimension carries a usable (positive) value.
func isSet(dim *int64) bool {
	return dim != nil && *dim > 0
}

// agrees reports whether a format dimension is compatible with the banner's
// top-level dimension: always when that one is unset, else only when equal.
func agrees(dim *int64, formatDim int64) bool {
	return !isSet(dim) || *dim == formatDim
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		// 204 = no-bid (filter, timeout, no DSP, shed). Not an adapter error.
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidCount := 0
	for _, sb := range bidResp.SeatBid {
		bidCount += len(sb.Bid)
	}
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(bidCount)
	// Keep the "USD" default set by NewBidderResponseWithBidsCapacity when the
	// response omits cur, rather than blanking Currency (a required field).
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getBidType(bid, request.Imp)
			if err != nil {
				// A bid of unknown type cannot be rendered correctly; skip it
				// rather than guess, and keep the other bids.
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

// getBidType returns the media type of a bid. Optima passes through the mtype its
// DSP set (1 banner, 2 video) and leaves it unset when the DSP omitted it, so
// mtype is trusted when present and otherwise inferred from the imp the bid
// answers. The imp check puts video first because Optima sells an imp that has both
// banner and video as video.
func getBidType(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case 0:
		for _, imp := range imps {
			if imp.ID != bid.ImpID {
				continue
			}
			switch {
			case imp.Video != nil:
				return openrtb_ext.BidTypeVideo, nil
			case imp.Banner != nil:
				return openrtb_ext.BidTypeBanner, nil
			}
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("optima: cannot determine media type of bid %s: mtype is not set and impid %q has no banner or video imp in the request", bid.ID, bid.ImpID),
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("optima: unsupported mtype %d in bid %s (impid %q); only banner and video are served", bid.MType, bid.ID, bid.ImpID),
	}
}
