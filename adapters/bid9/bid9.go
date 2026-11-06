package bid9

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

const defaultCurrency = "USD"

var (
	vastPattern = regexp.MustCompile(`(?i)^\s*(<\?xml|<VAST)`)
	// auctionMacroPattern matches any OpenRTB auction macro, such as ${AUCTION_PRICE}.
	auctionMacroPattern = regexp.MustCompile(`\$\{AUCTION_[A-Z_]+\}`)
	// emptyMacroParamPattern matches a query parameter whose whole value is a macro left without a value, with the
	// separator before it and an optional "&" after it.
	emptyMacroParamPattern = regexp.MustCompile(`([?&])[^=&?#\s"'<>]+=\$\{AUCTION_[A-Z_]+\}(&?)`)
)

type adapter struct {
	endpoint *url.URL
}

// Builder builds a new instance of the bid9 adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint: %w", err)
	}

	return &adapter{endpoint: endpoint}, nil
}

// MakeRequests sends each impression as its own request, because the bid9 endpoint reads only the first impression.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	for _, imp := range request.Imp {
		requestData, err := a.makeRequest(*request, imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, requestData)
	}

	return requests, errs
}

// makeRequest takes the request and impression by value, so changes stay out of what other bidders and impressions see.
func (a *adapter) makeRequest(request openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	var impExt map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: invalid imp.ext: %s", imp.ID, err.Error())}
	}

	var params openrtb_ext.ExtImpBid9
	if err := jsonutil.Unmarshal(impExt["bidder"], &params); err != nil {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: invalid bidder params: %s", imp.ID, err.Error())}
	}

	impExt["bid9"] = impExt["bidder"]
	delete(impExt, "bidder")

	ext, err := jsonutil.Marshal(impExt)
	if err != nil {
		return nil, err
	}
	imp.Ext = ext
	imp.TagID = params.ChannelID

	if imp.BidFloor == 0 && params.BidFloor > 0 {
		imp.BidFloor = params.BidFloor
		imp.BidFloorCur = params.BidFloorCur
		if imp.BidFloorCur == "" {
			imp.BidFloorCur = defaultCurrency
		}
	}

	if params.PublisherID != "" {
		setPublisherID(&request, params.PublisherID)
	}

	request.Imp = []openrtb2.Imp{imp}
	body, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpointFor(params.ChannelID),
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) endpointFor(channelID string) string {
	endpoint := *a.endpoint
	query := endpoint.Query()
	query.Set("tagid", channelID)
	endpoint.RawQuery = query.Encode()
	return endpoint.String()
}

// setPublisherID replaces Site or App and its Publisher with copies, because other bidders' requests share the originals.
func setPublisherID(request *openrtb2.BidRequest, publisherID string) {
	if request.Site != nil {
		site := *request.Site
		site.Publisher = publisherWithID(site.Publisher, publisherID)
		request.Site = &site
	} else if request.App != nil {
		app := *request.App
		app.Publisher = publisherWithID(app.Publisher, publisherID)
		request.App = &app
	}
}

func publisherWithID(publisher *openrtb2.Publisher, publisherID string) *openrtb2.Publisher {
	if publisher == nil {
		return &openrtb2.Publisher{ID: publisherID}
	}
	if publisher.ID != "" {
		return publisher
	}
	publisherCopy := *publisher
	publisherCopy.ID = publisherID
	return &publisherCopy
}

// MakeBids maps one endpoint response to typed bids. Each response answers only the impression in its own request,
// so bids are matched against the request that was sent, not against the auction's full list of impressions.
func (a *adapter) MakeBids(_ *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("invalid response body: %s", err.Error())}}
	}

	var sentRequest openrtb2.BidRequest
	if err := jsonutil.Unmarshal(requestData.Body, &sentRequest); err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("invalid request body: %s", err.Error())}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(sentRequest.Imp))
	bidResponse.Currency = defaultCurrency
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			imp := findImp(sentRequest.Imp, bid.ImpID)
			if imp == nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("bid %s: impid %s does not match the request's impression", bid.ID, bid.ImpID),
				})
				continue
			}
			resolveMacros(bid, map[string]string{
				"${AUCTION_ID}":       sentRequest.ID,
				"${AUCTION_BID_ID}":   response.BidID,
				"${AUCTION_IMP_ID}":   bid.ImpID,
				"${AUCTION_SEAT_ID}":  seatBid.Seat,
				"${AUCTION_AD_ID}":    bid.AdID,
				"${AUCTION_PRICE}":    strconv.FormatFloat(bid.Price, 'f', -1, 64),
				"${AUCTION_CURRENCY}": bidResponse.Currency,
			})
			moveWinNoticeToBillingURL(bid)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: getMediaType(bid, imp),
			})
		}
	}

	return bidResponse, errs
}

func findImp(imps []openrtb2.Imp, impID string) *openrtb2.Imp {
	for i := range imps {
		if imps[i].ID == impID {
			return &imps[i]
		}
	}
	return nil
}

// resolveMacros fills in the auction macros in adm, nurl and burl, because Prebid Server core does not resolve them.
// A macro without a value (for example ${AUCTION_SEAT_ID} when the response has no seat) is not sent: a query
// parameter that holds only that macro is removed, and the macro is removed anywhere else.
func resolveMacros(bid *openrtb2.Bid, values map[string]string) {
	replacements := make([]string, 0, 2*len(values))
	for macro, value := range values {
		if value != "" {
			replacements = append(replacements, macro, url.QueryEscape(value))
		}
	}
	replacer := strings.NewReplacer(replacements...)
	bid.AdM = removeUnresolvedMacros(replacer.Replace(bid.AdM))
	bid.NURL = removeUnresolvedMacros(replacer.Replace(bid.NURL))
	bid.BURL = removeUnresolvedMacros(replacer.Replace(bid.BURL))
}

func removeUnresolvedMacros(text string) string {
	if !strings.Contains(text, "${AUCTION_") {
		return text
	}
	// repeat, because a match may use up the "&" that the next parameter starts with
	for {
		next := emptyMacroParamPattern.ReplaceAllStringFunc(text, func(param string) string {
			separator, rest := param[:1], param[1:]
			if strings.HasSuffix(rest, "&") {
				return separator
			}
			return ""
		})
		if next == text {
			break
		}
		text = next
	}
	return auctionMacroPattern.ReplaceAllString(text, "")
}

// moveWinNoticeToBillingURL makes sure bid9 hears about its wins. Neither Prebid Server nor Prebid.js fires nurl on
// the server-side path, but Prebid.js fires burl when the ad renders. A bid with markup and no burl therefore sends
// its nurl as burl. Leaving nurl set as well would make Prebid.js treat a video bid's win notice as its VAST URL.
// A bid without markup keeps nurl, because there nurl is where the markup is fetched from.
func moveWinNoticeToBillingURL(bid *openrtb2.Bid) {
	if bid.AdM == "" || bid.NURL == "" || bid.BURL != "" {
		return
	}
	bid.BURL = bid.NURL
	bid.NURL = ""
}

func getMediaType(bid *openrtb2.Bid, imp *openrtb2.Imp) openrtb_ext.BidType {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	}

	if imp.Video != nil && imp.Banner == nil {
		return openrtb_ext.BidTypeVideo
	}
	if imp.Banner != nil && imp.Video == nil {
		return openrtb_ext.BidTypeBanner
	}
	if vastPattern.MatchString(bid.AdM) {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}
