package waardex

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

const auctionPriceMacro = "${AUCTION_PRICE}"

var markupTypes = map[openrtb_ext.BidType]openrtb2.MarkupType{
	openrtb_ext.BidTypeBanner: openrtb2.MarkupBanner,
	openrtb_ext.BidTypeVideo:  openrtb2.MarkupVideo,
}

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Waardex adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{endpoint: endpoint}, nil
}

// MakeRequests sends every impression in a request of its own, once per media type it offers.
// The Waardex exchange runs one auction per request and answers with the winning buyer's
// response alone, so impressions sharing a request would also share a single winner; it also
// accepts exactly one media type per impression.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requests := make([]*adapters.RequestData, 0, len(request.Imp))
	var errs []error
	for _, imp := range request.Imp {
		zoneID, err := prepareImp(&imp, reqInfo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		singleFormatImps := splitByMediaType(imp)
		if len(singleFormatImps) == 0 {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp %s has no supported media type (banner or video)", imp.ID),
			})
			continue
		}
		for _, singleFormatImp := range singleFormatImps {
			requestData, err := a.makeRequest(request, singleFormatImp, zoneID)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			requests = append(requests, requestData)
		}
	}
	return requests, errs
}

// prepareImp reads the zone id from the imp's bidder params and rewrites the imp in place into the
// shape the exchange reads: a rewarded flag in imp.rwdd and a floor in US dollars.
func prepareImp(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) (int, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("imp.ext is invalid: %s", err.Error()),
		}
	}
	var params openrtb_ext.ExtImpWaardex
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("imp.ext.bidder is invalid: %s", err.Error()),
		}
	}

	// Prebid Server moves imp.rwdd into imp.ext.prebid for OpenRTB 2.5 bidders, where the exchange
	// does not look for it.
	if prebid := bidderExt.Prebid; prebid != nil && prebid.IsRewardedInventory != nil && *prebid.IsRewardedInventory == 1 {
		imp.Rwdd = 1
	}

	if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, "USD") {
		bidFloor, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return 0, err
		}
		imp.BidFloor = bidFloor
		imp.BidFloorCur = "USD"
	}

	return params.ZoneID, nil
}

// splitByMediaType returns one copy of imp for each media type Waardex bids on, banner and video,
// each carrying that media type alone.
func splitByMediaType(imp openrtb2.Imp) []openrtb2.Imp {
	withoutMedia := imp
	withoutMedia.Banner, withoutMedia.Video, withoutMedia.Audio, withoutMedia.Native = nil, nil, nil, nil

	var imps []openrtb2.Imp
	if imp.Banner != nil {
		bannerImp := withoutMedia
		bannerImp.Banner = withPrimarySize(imp.Banner)
		imps = append(imps, bannerImp)
	}
	if imp.Video != nil {
		videoImp := withoutMedia
		videoImp.Video = imp.Video
		imps = append(imps, videoImp)
	}
	return imps
}

// withPrimarySize fills banner.w/h from the first format when they are missing. The exchange matches
// size allowlists and prices banners by w/h, while Prebid usually sends the sizes in format alone.
func withPrimarySize(banner *openrtb2.Banner) *openrtb2.Banner {
	if (banner.W != nil && banner.H != nil) || len(banner.Format) == 0 {
		return banner
	}
	sized := *banner
	w, h := banner.Format[0].W, banner.Format[0].H
	sized.W = &w
	sized.H = &h
	return &sized
}

// makeRequest wraps a single-format imp into a request of its own for the imp's zone.
func (a *adapter) makeRequest(request *openrtb2.BidRequest, imp openrtb2.Imp, zoneID int) (*adapters.RequestData, error) {
	uri, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{ZoneID: strconv.Itoa(zoneID)})
	if err != nil {
		return nil, err
	}

	outgoing := *request
	// The exchange hands a hash of the request id to its buyers as the auction id, so the requests
	// split from one auction need ids of their own.
	outgoing.ID = fmt.Sprintf("%s-%s-%s", request.ID, imp.ID, getImpMediaType(&imp))
	outgoing.Imp = []openrtb2.Imp{imp}
	// The exchange rejects buyer responses in any other currency; Prebid Server converts the
	// dollar bids back to the request currency.
	outgoing.Cur = []string{"USD"}

	body, err := json.Marshal(outgoing)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  []string{imp.ID},
	}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	mediaTypes := getRequestedMediaTypes(internalRequest, externalRequest)
	bidderResponse := adapters.NewBidderResponse()
	if bidResponse.Cur != "" {
		bidderResponse.Currency = bidResponse.Cur
	}
	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			mediaType, ok := mediaTypes[bid.ImpID]
			if !ok {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("bid %s is for imp %s, which was not requested", bid.ID, bid.ImpID),
				})
				continue
			}
			bid.MType = markupTypes[mediaType]
			resolveAuctionPrice(bid)
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{Bid: bid, BidType: mediaType})
		}
	}
	return bidderResponse, errs
}

// getRequestedMediaTypes maps every impression of an outgoing request to the single media type it
// was requested with. That media type, not bid.mtype, decides the bid type (and the bid's mtype is
// set to match): many buyers behind the exchange leave mtype unset, and the exchange passes the
// buyer's mtype through unchanged, even for in-banner video creatives it answers a banner
// impression with as banner markup. A stored bid response has no outgoing request, so its
// impressions are read from the incoming one.
func getRequestedMediaTypes(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData) map[string]openrtb_ext.BidType {
	var request openrtb2.BidRequest
	if err := jsonutil.Unmarshal(externalRequest.Body, &request); err != nil {
		request = *internalRequest
	}
	mediaTypes := make(map[string]openrtb_ext.BidType, len(request.Imp))
	for i := range request.Imp {
		mediaTypes[request.Imp[i].ID] = getImpMediaType(&request.Imp[i])
	}
	return mediaTypes
}

// getImpMediaType is the media type of an impression split for the exchange: video when it carries
// video alone, banner otherwise.
func getImpMediaType(imp *openrtb2.Imp) openrtb_ext.BidType {
	if imp.Video != nil && imp.Banner == nil {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}

// resolveAuctionPrice fills the clearing price into the bid's markup and notice urls. Prebid Server
// runs a first-price auction, so it is the bid price. The exchange reads it from its tracking urls,
// and not every renderer resolves the macro (VAST served from Prebid Cache never does).
func resolveAuctionPrice(bid *openrtb2.Bid) {
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.ReplaceAll(bid.AdM, auctionPriceMacro, price)
	bid.NURL = strings.ReplaceAll(bid.NURL, auctionPriceMacro, price)
	bid.BURL = strings.ReplaceAll(bid.BURL, auctionPriceMacro, price)
}
