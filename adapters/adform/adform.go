package adform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type AdformAdapter struct {
	http    *adapters.HTTPAdapter
	URI     string
	version string
}

type adformRequest struct {
	tid        string
	userAgent  string
	ip         string
	bidderCode string
	isSecure   bool
	referer    string
	userId     string
	adUnits    []*adformAdUnit
}

type adformAdUnit struct {
	MasterTagId json.Number `json:"mid"`

	bidId      string
	adUnitCode string
}

type adformBid struct {
	ResponseType string  `json:"response,omitempty"`
	Banner       string  `json:"banner,omitempty"`
	Price        float64 `json:"win_bid,omitempty"`
	Currency     string  `json:"win_cur,omitempty"`
	Width        uint64  `json:"width,omitempty"`
	Height       uint64  `json:"height,omitempty"`
	DealId       string  `json:"deal_id,omitempty"`
}

// ADAPTER Interface

func NewAdformAdapter(config *adapters.HTTPAdapterConfig, endpointURL string) *AdformAdapter {
	return NewAdformBidder(adapters.NewHTTPAdapter(config).Client, endpointURL)
}

/* Name - export adapter name */
func (a *AdformAdapter) Name() string {
	return "Adform"
}

// used for cookies and such
func (a *AdformAdapter) FamilyName() string {
	return "adform"
}

func (a *AdformAdapter) SkipNoCookies() bool {
	return false
}

func (a *AdformAdapter) Call(ctx context.Context, request *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	adformRequest, err := pbsRequestToAdformRequest(a, request, bidder)
	if err != nil {
		return nil, err
	}

	uri := adformRequest.buildAdformUrl(a)

	debug := &pbs.BidderDebug{RequestURI: uri}
	if request.IsDebug {
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpRequest, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	httpRequest.Header = adformRequest.buildAdformHeaders(a)

	response, err := ctxhttp.Do(ctx, a.http.Client, httpRequest)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = response.StatusCode

	if response.StatusCode == 204 {
		return nil, nil
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d; body: %s", response.StatusCode, responseBody)
	}

	if request.IsDebug {
		debug.ResponseBody = responseBody
	}

	adformBids, err := parseAdformBids(body)
	if err != nil {
		return nil, err
	}

	bids := toPBSBidSlice(adformBids, adformRequest)

	return bids, nil
}

func pbsRequestToAdformRequest(a *AdformAdapter, request *pbs.PBSRequest, bidder *pbs.PBSBidder) (*adformRequest, error) {
	adUnits := make([]*adformAdUnit, 0, len(bidder.AdUnits))
	for _, adUnit := range bidder.AdUnits {
		var adformAdUnit adformAdUnit
		if err := json.Unmarshal(adUnit.Params, &adformAdUnit); err != nil {
			return nil, err
		}
		mid, err := adformAdUnit.MasterTagId.Int64()
		if err != nil {
			return nil, err
		}
		if mid <= 0 {
			return nil, fmt.Errorf("master tag(placement) id is invalid=%s", adformAdUnit.MasterTagId)
		}
		adformAdUnit.bidId = adUnit.BidID
		adformAdUnit.adUnitCode = adUnit.Code
		adUnits = append(adUnits, &adformAdUnit)
	}

	userId, _, _ := request.Cookie.GetUID(a.FamilyName())

	return &adformRequest{
		adUnits:    adUnits,
		ip:         request.Device.IP,
		userAgent:  request.Device.UA,
		bidderCode: bidder.BidderCode,
		isSecure:   request.Secure == 1,
		referer:    request.Url,
		userId:     userId,
		tid:        request.Tid,
	}, nil
}

func toPBSBidSlice(adformBids []*adformBid, r *adformRequest) pbs.PBSBidSlice {
	bids := make(pbs.PBSBidSlice, 0)

	for i, bid := range adformBids {
		if bid.Banner == "" || bid.ResponseType != "banner" {
			continue
		}
		pbsBid := pbs.PBSBid{
			BidID:             r.adUnits[i].bidId,
			AdUnitCode:        r.adUnits[i].adUnitCode,
			BidderCode:        r.bidderCode,
			Price:             bid.Price,
			Adm:               bid.Banner,
			Width:             bid.Width,
			Height:            bid.Height,
			DealId:            bid.DealId,
			CreativeMediaType: string(openrtb_ext.BidTypeBanner),
		}

		bids = append(bids, &pbsBid)
	}

	return bids
}

// COMMON

func (r *adformRequest) buildAdformUrl(a *AdformAdapter) string {
	adUnitsParams := make([]string, 0, len(r.adUnits))
	for _, adUnit := range r.adUnits {
		str := fmt.Sprintf("mid=%s", adUnit.MasterTagId)
		adUnitsParams = append(adUnitsParams, base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(str)))
	}
	uri := a.URI
	if r.isSecure {
		uri = strings.Replace(uri, "http://", "https://", 1)
	}
	return fmt.Sprintf("%s/?CC=1&rp=4&fd=1&stid=%s&%s", uri, r.tid, strings.Join(adUnitsParams, "&"))
}

func (r *adformRequest) buildAdformHeaders(a *AdformAdapter) http.Header {
	header := http.Header{}

	header.Set("Content-Type", "application/json;charset=utf-8")
	header.Set("Accept", "application/json")
	header.Set("X-Request-Agent", fmt.Sprintf("PrebidAdapter %s", a.version))
	header.Set("User-Agent", r.userAgent)
	header.Set("X-Forwarded-For", r.ip)
	if r.referer != "" {
		header.Set("Referer", r.referer)
	}
	if r.userId != "" {
		header.Set("Cookie", fmt.Sprintf("uid=%s", r.userId))
	}

	return header
}

func parseAdformBids(response []byte) ([]*adformBid, error) {
	var bids []*adformBid
	if err := json.Unmarshal(response, &bids); err != nil {
		return nil, err
	}

	return bids, nil
}

// BIDDER Interface

func NewAdformBidder(client *http.Client, endpointURL string) *AdformAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &AdformAdapter{
		http:    a,
		URI:     endpointURL,
		version: "0.1.0",
	}
}

func (a *AdformAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	adformRequest, errors := openRtbToAdformRequest(request)
	if len(adformRequest.adUnits) == 0 {
		return nil, errors
	}

	requestData := adapters.RequestData{
		Method:  "GET",
		Uri:     adformRequest.buildAdformUrl(a),
		Body:    nil,
		Headers: adformRequest.buildAdformHeaders(a),
	}

	requests := []*adapters.RequestData{&requestData}

	return requests, errors
}

func openRtbToAdformRequest(request *openrtb.BidRequest) (*adformRequest, []error) {
	adUnits := make([]*adformAdUnit, 0, len(request.Imp))
	errors := make([]error, 0, len(request.Imp))
	secure := false
	for _, imp := range request.Imp {
		if imp.Banner == nil {
			errors = append(errors, fmt.Errorf("Adform adapter supports only banner Imps for now. Ignoring Imp ID=%s", imp.ID))
			continue
		}

		params, _, _, err := jsonparser.Get(imp.Ext, "bidder")
		if err != nil {
			errors = append(errors, err)
			continue
		}
		var adformAdUnit adformAdUnit
		if err := json.Unmarshal(params, &adformAdUnit); err != nil {
			errors = append(errors, err)
			continue
		}

		mid, err := adformAdUnit.MasterTagId.Int64()
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if mid <= 0 {
			errors = append(errors, fmt.Errorf("master tag(placement) id is invalid=%s", adformAdUnit.MasterTagId))
			continue
		}

		if !secure && imp.Secure != nil && *imp.Secure == 1 {
			secure = true
		}

		adformAdUnit.bidId = imp.ID
		adformAdUnit.adUnitCode = imp.ID
		adUnits = append(adUnits, &adformAdUnit)
	}

	referer := ""
	if request.Site != nil {
		referer = request.Site.Page
	}

	tid := ""
	if request.Source != nil {
		tid = request.Source.TID
	}

	return &adformRequest{
		adUnits:   adUnits,
		ip:        request.Device.IP,
		userAgent: request.Device.UA,
		isSecure:  secure,
		referer:   referer,
		userId:    request.User.BuyerUID,
		tid:       tid,
	}, errors
}

func (a *AdformAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	adformOutput, err := parseAdformBids(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	bids := toOpenRtbBids(adformOutput, internalRequest)

	return bids, nil
}

func toOpenRtbBids(adformBids []*adformBid, r *openrtb.BidRequest) []*adapters.TypedBid {
	bids := make([]*adapters.TypedBid, 0, len(adformBids))

	for i, bid := range adformBids {
		if bid.Banner == "" || bid.ResponseType != "banner" {
			continue
		}
		openRtbBid := openrtb.Bid{
			ID:     r.Imp[i].ID,
			ImpID:  r.Imp[i].ID,
			Price:  bid.Price,
			AdM:    bid.Banner,
			W:      bid.Width,
			H:      bid.Height,
			DealID: bid.DealId,
		}

		bids = append(bids, &adapters.TypedBid{Bid: &openRtbBid, BidType: openrtb_ext.BidTypeBanner})
	}

	return bids
}
