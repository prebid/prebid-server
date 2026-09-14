package adswag

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// The Adswag bid endpoint speaks plain OpenRTB 2.6 JSON in/out and answers
// a no-bid as an empty-body HTTP 200 (its Prebid.js contract) as well as
// the conventional 204.
//
// Display (banner) bids carry markup in adm (a loader script tag that
// injects the composed creative into the containing frame at render time);
// it passes through unchanged. bid.ext.adswag.serve_url remains populated
// as a legacy fallback: if adm is ever absent, the adapter synthesizes an
// iframe tag around that URL so the bid still carries renderable markup
// for every PBS consumer (network-equivalent to the Prebid.js adapter's
// adUrl render).
// Video and audio bids carry VAST markup in adm and pass through unchanged.

const defaultCurrency = "EUR"

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Adswag adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "no impressions in the bid request"}}
	}

	// The endpoint resolves the supply-side publisher from
	// site.publisher.id / app.publisher.id, and the bidder param is the
	// source of truth for that value. Both are REQUEST-level fields, so imps
	// are grouped by their publisherId and each group is sent as its own
	// request: PBS hands an adapter every imp that named the bidder
	// regardless of the bidder params inside them (exchange.splitImps groups
	// by bidder name alone), and those imps may legitimately belong to
	// different Adswag publisher accounts. Folding them into one request
	// would report — and pay — every impression under whichever account
	// happened to come first. Groups keep the order of their first imp.
	//
	// Imps whose ext fails to parse or lacks a publisherId are dropped so
	// malformed data is never forwarded upstream; each error is still
	// reported to the caller.
	var errs []error
	var publisherIDs []string
	impsByPublisher := make(map[string][]openrtb2.Imp)
	for i := range request.Imp {
		imp := request.Imp[i]
		ext, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, seen := impsByPublisher[ext.PublisherID]; !seen {
			publisherIDs = append(publisherIDs, ext.PublisherID)
		}
		impsByPublisher[ext.PublisherID] = append(impsByPublisher[ext.PublisherID], imp)
	}
	if len(publisherIDs) == 0 {
		return nil, errs
	}

	// Request-level and identical for every group, so it is checked once.
	if request.Site == nil && request.App == nil {
		return nil, append(errs, &errortypes.BadInput{Message: "request must contain either site or app"})
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requests := make([]*adapters.RequestData, 0, len(publisherIDs))
	for _, publisherID := range publisherIDs {
		outgoing := *request
		outgoing.Imp = impsByPublisher[publisherID]
		setPublisherID(&outgoing, publisherID)

		body, err := jsonutil.Marshal(&outgoing)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(outgoing.Imp),
		})
	}
	if len(requests) == 0 {
		return nil, errs
	}
	return requests, errs
}

// parseImpExt validates the imp's bidder ext and rewrites imp.ext into the
// shape the Adswag endpoint expects: the prebid/bidder envelopes are removed
// and an optional placementId override moves to ext.adswag.placement_id.
// Standard placement-identity fields (gpid, tid, data.pbadslot) stay where
// PBS core put them. The imp is modified in place (the caller owns a copy).
func parseImpExt(imp *openrtb2.Imp) (openrtb_ext.ExtImpAdswag, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpAdswag{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext for imp %s: %s", imp.ID, err)}
	}
	var adswagExt openrtb_ext.ExtImpAdswag
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adswagExt); err != nil {
		return openrtb_ext.ExtImpAdswag{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext.bidder for imp %s: %s", imp.ID, err)}
	}
	adswagExt.PublisherID = strings.TrimSpace(adswagExt.PublisherID)
	if adswagExt.PublisherID == "" {
		return openrtb_ext.ExtImpAdswag{}, &errortypes.BadInput{Message: fmt.Sprintf("missing publisherId for imp %s", imp.ID)}
	}

	fields := map[string]jsonutil.RawMessage{}
	if err := jsonutil.Unmarshal(imp.Ext, &fields); err != nil {
		return openrtb_ext.ExtImpAdswag{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext for imp %s: %s", imp.ID, err)}
	}
	delete(fields, "bidder")
	delete(fields, "prebid")
	if adswagExt.PlacementID != "" {
		placement, err := jsonutil.Marshal(map[string]string{"placement_id": adswagExt.PlacementID})
		if err != nil {
			return openrtb_ext.ExtImpAdswag{}, err
		}
		fields["adswag"] = placement
	}
	if len(fields) == 0 {
		imp.Ext = nil
		return adswagExt, nil
	}
	ext, err := jsonutil.Marshal(fields)
	if err != nil {
		return openrtb_ext.ExtImpAdswag{}, err
	}
	imp.Ext = ext
	return adswagExt, nil
}

// setPublisherID writes the bidder-param publisherId to site.publisher.id or
// app.publisher.id (copy-on-write; other publisher fields are preserved).
// The caller has already established that one of the two is present.
func setPublisherID(request *openrtb2.BidRequest, publisherID string) {
	switch {
	case request.Site != nil:
		site := *request.Site
		publisher := openrtb2.Publisher{}
		if site.Publisher != nil {
			publisher = *site.Publisher
		}
		publisher.ID = publisherID
		site.Publisher = &publisher
		request.Site = &site
	case request.App != nil:
		app := *request.App
		publisher := openrtb2.Publisher{}
		if app.Publisher != nil {
			publisher = *app.Publisher
		}
		publisher.ID = publisherID
		app.Publisher = &publisher
		request.App = &app
	}
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}
	// The endpoint's Prebid-path no-bid is an empty-body HTTP 200.
	if len(strings.TrimSpace(string(responseData.Body))) == 0 {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = defaultCurrency
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			typedBid, err := makeTypedBid(request.Imp, &seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}
	return bidResponse, errs
}

// bidExt is the subset of the endpoint's bid.ext the adapter reads.
type bidExt struct {
	Adswag struct {
		ServeURL string `json:"serve_url"`
	} `json:"adswag"`
}

var vastMarkup = regexp.MustCompile(`(?i)<\s*VAST[\s/>]`)

// makeTypedBid resolves the bid's media type and materializes markup for
// bids served by URL: an iframe tag for display, a VAST wrapper for video
// and audio. The bid is modified in place (adapter-owned); bid.nurl is
// never written — it keeps its OpenRTB win-notice meaning.
//
// Type resolution mirrors the Prebid.js adapter and the endpoint's per-imp
// channel precedence (banner > audio > video): VAST markup on an imp that
// declares audio is an audio bid, VAST on a video imp is a video bid,
// everything else is banner. bid.mtype is honored first when present.
func makeTypedBid(imps []openrtb2.Imp, bid *openrtb2.Bid) (*adapters.TypedBid, error) {
	imp := findImp(imps, bid.ImpID)
	if imp == nil {
		return nil, &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s references unknown imp %s", bid.ID, bid.ImpID)}
	}

	var ext bidExt
	if len(bid.Ext) > 0 {
		// Malformed bid.ext only forfeits the serve_url fallback.
		_ = jsonutil.Unmarshal(bid.Ext, &ext)
	}
	serveURL := ext.Adswag.ServeURL
	if bid.AdM == "" && serveURL == "" {
		return nil, &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s has neither adm nor ext.adswag.serve_url", bid.ID)}
	}

	bidType, err := resolveBidType(imp, bid)
	if err != nil {
		return nil, err
	}

	switch bidType {
	case openrtb_ext.BidTypeBanner:
		w, h := bannerSize(imp)
		if bid.AdM == "" {
			bid.AdM = iframeMarkup(serveURL, w, h)
		}
		// The endpoint omits w/h; backfill from the requested primary size.
		if bid.W == 0 && bid.H == 0 && w > 0 && h > 0 {
			bid.W, bid.H = w, h
		}
		bid.MType = openrtb2.MarkupBanner
	case openrtb_ext.BidTypeVideo:
		if bid.AdM == "" {
			bid.AdM = vastWrapper(serveURL)
		}
		if bid.W == 0 && bid.H == 0 && imp.Video != nil && imp.Video.W != nil && imp.Video.H != nil {
			bid.W, bid.H = *imp.Video.W, *imp.Video.H
		}
		bid.MType = openrtb2.MarkupVideo
	case openrtb_ext.BidTypeAudio:
		if bid.AdM == "" {
			bid.AdM = vastWrapper(serveURL)
		}
		bid.MType = openrtb2.MarkupAudio
	}

	return &adapters.TypedBid{Bid: bid, BidType: bidType}, nil
}

func resolveBidType(imp *openrtb2.Imp, bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case 0:
		// The endpoint does not set mtype; resolve from the imp below.
	default:
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("unsupported bid.mtype %d for impression %s", bid.MType, bid.ImpID)}
	}

	// Audio is deliberately checked before video: the endpoint classifies
	// every imp into exactly ONE channel with precedence banner > audio >
	// video, so on a multi-format imp declaring both audio and video it
	// serves an AUDIO ad (audio uses VAST too — VAST 4.x absorbed DAAST).
	// The markup alone cannot distinguish the two; this ordering mirrors
	// the server's channel selection so the label matches what was served.
	isVast := vastMarkup.MatchString(bid.AdM)
	switch {
	case imp.Audio != nil && (isVast || imp.Banner == nil):
		return openrtb_ext.BidTypeAudio, nil
	case imp.Video != nil && (isVast || imp.Banner == nil):
		return openrtb_ext.BidTypeVideo, nil
	case imp.Banner != nil:
		return openrtb_ext.BidTypeBanner, nil
	}
	return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("unable to resolve media type for impression %s", bid.ImpID)}
}

func findImp(imps []openrtb2.Imp, impID string) *openrtb2.Imp {
	for i := range imps {
		if imps[i].ID == impID {
			return &imps[i]
		}
	}
	return nil
}

func bannerSize(imp *openrtb2.Imp) (int64, int64) {
	if imp.Banner == nil {
		return 0, 0
	}
	if len(imp.Banner.Format) > 0 {
		return imp.Banner.Format[0].W, imp.Banner.Format[0].H
	}
	if imp.Banner.W != nil && imp.Banner.H != nil {
		return *imp.Banner.W, *imp.Banner.H
	}
	return 0, 0
}

// vastWrapper wraps a VAST serve URL in a wrapper envelope — the same shape
// PBS core produces when it caches a video bid served by nurl
// (exchange.makeVAST). Synthesizing it here instead of writing bid.nurl
// keeps AUDIO bids renderable (core's nurl wrapping guards on BidTypeVideo
// only, so an audio bid served by URL would reach players with empty adm)
// and leaves bid.nurl to its OpenRTB meaning: the win-notice URL.
func vastWrapper(serveURL string) string {
	return `<VAST version="3.0"><Ad><Wrapper>` +
		`<AdSystem>adswag</AdSystem>` +
		`<VASTAdTagURI><![CDATA[` + serveURL + `]]></VASTAdTagURI>` +
		`<Impression></Impression><Creatives></Creatives>` +
		`</Wrapper></Ad></VAST>`
}

// iframeMarkup wraps a serve URL in an iframe tag; fetching the iframe src at
// render time is network-equivalent to Prebid.js rendering the URL as adUrl.
func iframeMarkup(serveURL string, w, h int64) string {
	var b strings.Builder
	b.WriteString(`<iframe src="`)
	b.WriteString(html.EscapeString(serveURL))
	b.WriteString(`"`)
	if w > 0 && h > 0 {
		b.WriteString(` width="` + strconv.FormatInt(w, 10) + `" height="` + strconv.FormatInt(h, 10) + `"`)
	}
	b.WriteString(` frameborder="0" scrolling="no" marginheight="0" marginwidth="0" style="border:0" title="Advertisement"></iframe>`)
	return b.String()
}
