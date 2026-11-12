// Package exgeniee sends bid requests to the Geniee Exchange endpoint.
package exgeniee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// The only currencies Geniee Exchange handles.
const (
	currencyJPY = "JPY"
	currencyUSD = "USD"
)

type adapter struct {
	endpointTemplate *template.Template
}

// Builder builds a new instance of the Geniee Exchange adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %w", err)
	}
	return &adapter{endpointTemplate: endpointTemplate}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if err := validate(request); err != nil {
		return nil, []error{err}
	}

	cur, err := normalizeCurrency(request.Cur)
	if err != nil {
		return nil, []error{err}
	}

	var requests []*adapters.RequestData
	var errs []error

	// Geniee Exchange accepts exactly one imp per request, so split them up.
	for _, imp := range request.Imp {
		params, err := takeParams(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri, err := a.buildEndpointURL(params)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Geniee Exchange reads the ad slot name off imp.tagid, and placementId names
		// that slot.
		if params.PlacementID != "" {
			imp.TagID = params.PlacementID
		}

		warnings, err := prepareFormats(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		errs = append(errs, warnings...)

		if err := convertBidFloor(&imp, requestInfo); err != nil {
			errs = append(errs, err)
		}

		requestCopy := *request
		requestCopy.ID = request.ID + "-" + imp.ID
		requestCopy.Cur = cur
		requestCopy.Imp = []openrtb2.Imp{imp}
		requestCopy.Ext = nil

		body, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

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

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	// Geniee Exchange answers in one of the request currencies (JPY or USD).
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:      bid,
				BidType:  bidType,
				BidVideo: getBidVideo(bid, bidType),
			})
		}
	}
	return bidderResponse, errs
}

// validate refuses a request Geniee Exchange cannot place: it identifies the
// inventory by site.page for the web and by app.bundle for an app.
func validate(request *openrtb2.BidRequest) error {
	if request.Site != nil && request.Site.Page == "" {
		return &errortypes.BadInput{Message: "site.page is required"}
	}
	if request.App != nil && request.App.Bundle == "" {
		return &errortypes.BadInput{Message: "app.bundle is required"}
	}
	return nil
}

// normalizeCurrency uppercases the codes in cur and refuses a request Geniee
// Exchange would not answer: it needs JPY or USD among them. The request is refused
// on purpose rather than forced to USD.
func normalizeCurrency(cur []string) ([]string, error) {
	if len(cur) == 0 {
		return nil, nil
	}
	upper := make([]string, len(cur))
	supported := false
	for i, c := range cur {
		upper[i] = strings.ToUpper(c)
		if upper[i] == currencyJPY || upper[i] == currencyUSD {
			supported = true
		}
	}
	if !supported {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("cur does not contain JPY or USD, got %s", strings.Join(cur, ",")),
		}
	}
	return upper, nil
}

// convertBidFloor moves the floor to JPY or USD. Geniee Exchange honours the floor
// only in those two currencies.
func convertBidFloor(imp *openrtb2.Imp, requestInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor <= 0 || imp.BidFloorCur == "" {
		return nil
	}

	cur := strings.ToUpper(imp.BidFloorCur)
	if cur == currencyJPY || cur == currencyUSD {
		// Geniee Exchange expects the code uppercase.
		imp.BidFloorCur = cur
		return nil
	}

	converted, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, currencyUSD)
	if err != nil {
		// Forward the request anyway, but say so: Geniee Exchange honours the floor
		// only in JPY or USD, so this one will not apply.
		return &errortypes.Warning{
			Message: fmt.Sprintf("imp %q: bidfloor left in %s, conversion to %s failed: %v",
				imp.ID, imp.BidFloorCur, currencyUSD, err),
		}
	}
	imp.BidFloor = converted
	imp.BidFloorCur = currencyUSD
	return nil
}

// takeParams reads the bidder params out of imp.ext and removes them from it so
// that the partner ID is not forwarded downstream.
func takeParams(imp *openrtb2.Imp) (*openrtb_ext.ExtImpExGeniee, error) {
	var ext map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: cannot read ext: %v", imp.ID, err),
		}
	}

	bidderParams, found := ext["bidder"]
	if !found {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: ext.bidder is not provided", imp.ID),
		}
	}

	var params openrtb_ext.ExtImpExGeniee
	if err := jsonutil.Unmarshal(bidderParams, &params); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: cannot read ext.bidder: %v", imp.ID, err),
		}
	}

	delete(ext, "bidder")
	// Internal to prebid, not for the endpoint.
	delete(ext, "prebid")
	if len(ext) == 0 {
		imp.Ext = nil
		return &params, nil
	}

	remaining, err := jsonutil.Marshal(ext)
	if err != nil {
		return nil, err
	}
	imp.Ext = remaining
	return &params, nil
}

// prepareFormats shapes each format of the imp the way Geniee Exchange expects it and
// refuses an imp it cannot place. A format that fails its check is dropped from
// the imp with a Warning as long as the other format passes.
func prepareFormats(imp *openrtb2.Imp) (warnings []error, err error) {
	if imp.Banner == nil && imp.Video == nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: has neither banner nor video", imp.ID),
		}
	}

	var bannerErr, videoErr error
	if imp.Banner != nil {
		bannerErr = prepareBanner(imp)
	}
	if imp.Video != nil {
		videoErr = prepareVideo(imp)
	}

	bannerOK := imp.Banner != nil && bannerErr == nil
	videoOK := imp.Video != nil && videoErr == nil
	if !bannerOK && !videoOK {
		if bannerErr != nil && videoErr != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("%v; %v", bannerErr, videoErr),
			}
		}
		if bannerErr != nil {
			return nil, bannerErr
		}
		return nil, videoErr
	}

	if bannerErr != nil {
		imp.Banner = nil
		warnings = append(warnings, &errortypes.Warning{
			Message: fmt.Sprintf("%v, forwarding video only", bannerErr),
		})
	}
	if videoErr != nil {
		imp.Video = nil
		warnings = append(warnings, &errortypes.Warning{
			Message: fmt.Sprintf("%v, forwarding banner only", videoErr),
		})
	}
	return warnings, nil
}

// prepareBanner derives banner.w and banner.h from format[0] when either is
// missing, and then refuses a banner still without one. Both are taken from
// format[0] on purpose, even when one of them was given.
func prepareBanner(imp *openrtb2.Imp) error {
	if (imp.Banner.W == nil || imp.Banner.H == nil) && len(imp.Banner.Format) > 0 {
		// imp is a copy, but Banner is a pointer into the caller's request, so
		// copy it before writing the size.
		w, h := imp.Banner.Format[0].W, imp.Banner.Format[0].H
		banner := *imp.Banner
		banner.W = &w
		banner.H = &h
		imp.Banner = &banner
	}
	if imp.Banner.W == nil || imp.Banner.H == nil || *imp.Banner.W <= 0 || *imp.Banner.H <= 0 {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: banner has no size", imp.ID),
		}
	}
	return nil
}

// prepareVideo refuses a video without mimes, the one required field of the
// video object at Geniee Exchange.
func prepareVideo(imp *openrtb2.Imp) error {
	if len(imp.Video.MIMEs) == 0 {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: video has no mimes", imp.ID),
		}
	}
	return nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpExGeniee) (string, error) {
	return macros.ResolveMacros(a.endpointTemplate, macros.EndpointTemplateParams{
		PartnerId: strconv.Itoa(params.PartnerID),
	})
}

// getMediaTypeForBid decides the type from bid.mtype alone.
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid of impression %q", bid.MType, bid.ImpID),
		}
	}
}

// getBidVideo carries bid.dur and the first bid.cat over for a video bid so
// that they reach seatbid.bid.ext.prebid.video.
func getBidVideo(bid *openrtb2.Bid, bidType openrtb_ext.BidType) *openrtb_ext.ExtBidPrebidVideo {
	if bidType != openrtb_ext.BidTypeVideo {
		return nil
	}
	bidVideo := &openrtb_ext.ExtBidPrebidVideo{Duration: int(bid.Dur)}
	if len(bid.Cat) > 0 {
		bidVideo.PrimaryCategory = bid.Cat[0]
	}
	return bidVideo
}
