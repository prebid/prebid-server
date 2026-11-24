// Package riptide is the Riptide bid adapter for Prebid Server: OpenRTB 2.6 to the endpoint in the
// bidder's bidder-info file (the parent's `riptide.yaml`, or a white-label alias's own file with
// `aliasOf: riptide` and its own endpoint), one request per publisher, `tagid` and
// `site/app.publisher.id` taken from the imp params, and bids typed by `mtype` (the imp's media
// type when a bid has none). Riptide answers an uncoded no-bid as 204 and a coded one as 200
// `{"id","nbr"}`; both are an empty bid list here.
package riptide

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	openRTBVersion = "2.6"
	floorCurrency  = "USD"
)

type adapter struct {
	endpoint string
}

// Builder builds the adapter for the parent or an alias. The endpoint is a literal absolute URL:
// the serving host is fixed per bidder code, never taken from the request.
func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" || strings.Contains(cfg.Endpoint, "{{") {
		return nil, fmt.Errorf("riptide: endpoint %q must be an absolute URL without macros", cfg.Endpoint)
	}
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	byPublisher := map[string][]openrtb2.Imp{}
	for _, imp := range request.Imp {
		publisherID, err := rewriteImp(&imp, reqInfo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		byPublisher[publisherID] = append(byPublisher[publisherID], imp)
	}
	publishers := make([]string, 0, len(byPublisher))
	for p := range byPublisher {
		publishers = append(publishers, p)
	}
	sort.Strings(publishers)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", openRTBVersion)
	out := make([]*adapters.RequestData, 0, len(publishers))
	for _, publisherID := range publishers {
		req := *request
		req.Imp = byPublisher[publisherID]
		withPublisher(&req, publisherID)
		body, err := jsonutil.Marshal(req)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, &adapters.RequestData{
			Method: http.MethodPost, Uri: a.endpoint, Body: body, Headers: headers,
			ImpIDs: openrtb_ext.GetImpIDs(req.Imp),
		})
	}
	return out, errs
}

// rewriteImp applies the imp's Riptide params (tagid, a floor when the imp has none), converts an
// imp floor in another currency to USD, and drops the Prebid-only ext members, keeping the rest
// (gpid, data); it returns the publisher id.
func rewriteImp(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) (string, error) {
	var ext map[string]json.RawMessage
	var p openrtb_ext.ExtImpRiptide
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("imp %s: %v", imp.ID, err)}
	}
	if err := jsonutil.Unmarshal(ext["bidder"], &p); err != nil {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("imp %s: bidder params: %v", imp.ID, err)}
	}
	if p.PublisherID == "" || p.PlacementID == "" {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("imp %s: publisherId and placementId are required", imp.ID)}
	}
	imp.TagID = p.PlacementID
	if p.BidFloor > 0 && imp.BidFloor == 0 {
		imp.BidFloor, imp.BidFloorCur = p.BidFloor, floorCurrency
	}
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, floorCurrency) {
		if reqInfo == nil {
			return "", &errortypes.BadInput{Message: fmt.Sprintf("imp %s: no currency conversion for a %s floor", imp.ID, imp.BidFloorCur)}
		}
		usd, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, floorCurrency)
		if err != nil {
			return "", &errortypes.BadInput{Message: fmt.Sprintf("imp %s: floor %s: %v", imp.ID, imp.BidFloorCur, err)}
		}
		imp.BidFloor, imp.BidFloorCur = usd, floorCurrency
	}
	delete(ext, "bidder")
	delete(ext, "prebid")
	imp.Ext = nil
	if len(ext) > 0 {
		b, err := jsonutil.Marshal(ext)
		if err != nil {
			return "", err
		}
		imp.Ext = b
	}
	return p.PublisherID, nil
}

// withPublisher sets the Riptide publisher public id on the inventory object the request carries:
// app, DOOH, else site (a request with none gets a site).
func withPublisher(req *openrtb2.BidRequest, publisherID string) {
	if req.App != nil {
		app := *req.App
		app.Publisher = publisherOf(app.Publisher, publisherID)
		req.App = &app
		return
	}
	if req.DOOH != nil {
		dooh := *req.DOOH
		dooh.Publisher = publisherOf(dooh.Publisher, publisherID)
		req.DOOH = &dooh
		return
	}
	site := openrtb2.Site{}
	if req.Site != nil {
		site = *req.Site
	}
	site.Publisher = publisherOf(site.Publisher, publisherID)
	req.Site = &site
}

func publisherOf(existing *openrtb2.Publisher, id string) *openrtb2.Publisher {
	pub := openrtb2.Publisher{}
	if existing != nil {
		pub = *existing
	}
	pub.ID = id
	return &pub
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	switch response.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusOK:
	default:
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("unexpected status code %d", response.StatusCode)}}
	}
	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "bad server response: " + err.Error()}}
	}
	if resp.NBR != nil || len(resp.SeatBid) == 0 {
		return nil, nil
	}
	imps := make(map[string]*openrtb2.Imp, len(request.Imp))
	for i := range request.Imp {
		imps[request.Imp[i].ID] = &request.Imp[i]
	}
	out := adapters.NewBidderResponseWithBidsCapacity(len(resp.SeatBid[0].Bid))
	if resp.Cur != "" {
		out.Currency = resp.Cur
	}
	var errs []error
	for _, seat := range resp.SeatBid {
		for i := range seat.Bid {
			bid := &seat.Bid[i]
			bidType, err := bidTypeOf(bid, imps[bid.ImpID])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			typed := &adapters.TypedBid{Bid: bid, BidType: bidType}
			if bidType == openrtb_ext.BidTypeVideo {
				// Duration and the first category drive PBS pod dedup and hb_pb_cat_dur.
				typed.BidVideo = &openrtb_ext.ExtBidPrebidVideo{Duration: int(bid.Dur)}
				if len(bid.Cat) > 0 {
					typed.BidVideo.PrimaryCategory = bid.Cat[0]
				}
			}
			out.Bids = append(out.Bids, typed)
		}
	}
	return out, errs
}

var mtypeBidType = map[openrtb2.MarkupType]openrtb_ext.BidType{
	openrtb2.MarkupBanner: openrtb_ext.BidTypeBanner,
	openrtb2.MarkupVideo:  openrtb_ext.BidTypeVideo,
	openrtb2.MarkupAudio:  openrtb_ext.BidTypeAudio,
	openrtb2.MarkupNative: openrtb_ext.BidTypeNative,
}

// bidTypeOf types a bid by its mtype, or, for a bid without one, by the single media type its imp
// asks for.
func bidTypeOf(bid *openrtb2.Bid, imp *openrtb2.Imp) (openrtb_ext.BidType, error) {
	if imp == nil {
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s answers unknown imp %s", bid.ID, bid.ImpID)}
	}
	if bid.MType != 0 {
		if t, ok := mtypeBidType[bid.MType]; ok {
			return t, nil
		}
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s has unknown mtype %d", bid.ID, bid.MType)}
	}
	var present []openrtb_ext.BidType
	for t, set := range map[openrtb_ext.BidType]bool{
		openrtb_ext.BidTypeBanner: imp.Banner != nil,
		openrtb_ext.BidTypeVideo:  imp.Video != nil,
		openrtb_ext.BidTypeAudio:  imp.Audio != nil,
		openrtb_ext.BidTypeNative: imp.Native != nil,
	} {
		if set {
			present = append(present, t)
		}
	}
	if len(present) != 1 {
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s has no mtype and imp %s is not single-format", bid.ID, bid.ImpID)}
	}
	return present[0], nil
}
