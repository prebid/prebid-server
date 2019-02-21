package adform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"golang.org/x/net/context/ctxhttp"
)

type AdformAdapter struct {
	http    *adapters.HTTPAdapter
	URL     *url.URL
	version string
}

type adformRequest struct {
	tid           string
	userAgent     string
	ip            string
	advertisingId string
	bidderCode    string
	isSecure      bool
	referer       string
	userId        string
	adUnits       []*adformAdUnit
	gdprApplies   string
	consent       string
	digitrust     *adformDigitrust
	cur           string
}

type adformDigitrust struct {
	Id      string                 `json:"id"`
	Version int                    `json:"version"`
	Keyv    int                    `json:"keyv"`
	Privacy adformDigitrustPrivacy `json:"privacy"`
}

type adformDigitrustPrivacy struct {
	Optout bool `json:"optout"`
}

type adformAdUnit struct {
	MasterTagId json.Number `json:"mid"`
	PriceType   string      `json:"priceType,omitempty"`

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
	CreativeId   string  `json:"win_crid,omitempty"`
}

const priceTypeGross = "gross"
const priceTypeNet = "net"

func isPriceTypeValid(priceType string) (string, bool) {
	pt := strings.ToLower(priceType)
	valid := pt == priceTypeNet || pt == priceTypeGross

	return pt, valid
}

// ADAPTER Interface

func NewAdformAdapter(config *adapters.HTTPAdapterConfig, endpointURL string) *AdformAdapter {
	return NewAdformBidder(adapters.NewHTTPAdapter(config).Client, endpointURL)
}

// used for cookies and such
func (a *AdformAdapter) Name() string {
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

	if response.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", response.StatusCode, responseBody),
		}
	}

	if response.StatusCode != 200 {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", response.StatusCode, responseBody),
		}
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
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
		if mid <= 0 {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("master tag(placement) id is invalid=%s", adformAdUnit.MasterTagId),
			}
		}
		adformAdUnit.bidId = adUnit.BidID
		adformAdUnit.adUnitCode = adUnit.Code
		adUnits = append(adUnits, &adformAdUnit)
	}

	userId, _, _ := request.Cookie.GetUID(a.Name())

	gdprApplies := request.ParseGDPR()
	if gdprApplies != "0" && gdprApplies != "1" {
		gdprApplies = ""
	}
	consent := request.ParseConsent()
	var digitrustData *openrtb_ext.ExtUserDigiTrust
	if request.User != nil {
		var extUser *openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			digitrustData = extUser.DigiTrust
		}
	}

	var digitrust *adformDigitrust = nil
	if digitrustData != nil {
		digitrust = new(adformDigitrust)
		digitrust.Id = digitrustData.ID
		digitrust.Keyv = digitrustData.KeyV
		digitrust.Version = 1
		digitrust.Privacy = adformDigitrustPrivacy{
			Optout: digitrustData.Pref != 0,
		}
	}

	return &adformRequest{
		adUnits:       adUnits,
		ip:            request.Device.IP,
		advertisingId: request.Device.IFA,
		userAgent:     request.Device.UA,
		bidderCode:    bidder.BidderCode,
		isSecure:      request.Secure == 1,
		referer:       request.Url,
		userId:        userId,
		tid:           request.Tid,
		gdprApplies:   gdprApplies,
		consent:       consent,
		digitrust:     digitrust,
		cur:           "USD",
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
			Creative_id:       bid.CreativeId,
			CreativeMediaType: string(openrtb_ext.BidTypeBanner),
		}

		bids = append(bids, &pbsBid)
	}

	return bids
}

// COMMON

func (r *adformRequest) buildAdformUrl(a *AdformAdapter) string {
	parameters := url.Values{}

	if r.advertisingId != "" {
		parameters.Add("adid", r.advertisingId)
	}
	parameters.Add("CC", "1")
	parameters.Add("rp", "4")
	parameters.Add("fd", "1")
	parameters.Add("stid", r.tid)
	parameters.Add("ip", r.ip)

	priceType := getValidPriceTypeParameter(r.adUnits)
	if priceType != "" {
		parameters.Add("pt", priceType)
	}

	parameters.Add("gdpr", r.gdprApplies)
	parameters.Add("gdpr_consent", r.consent)

	URL := *a.URL
	URL.RawQuery = parameters.Encode()

	uri := URL.String()
	if r.isSecure {
		uri = strings.Replace(uri, "http://", "https://", 1)
	}

	adUnitsParams := make([]string, 0, len(r.adUnits))
	for _, adUnit := range r.adUnits {
		str := fmt.Sprintf("mid=%s&rcur=%s", adUnit.MasterTagId, r.cur)
		adUnitsParams = append(adUnitsParams, base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(str)))
	}

	return fmt.Sprintf("%s&%s", uri, strings.Join(adUnitsParams, "&"))
}

func getValidPriceTypeParameter(adUnits []*adformAdUnit) string {
	priceTypeParameter := ""
	priceType := priceTypeNet
	valid := false
	for _, adUnit := range adUnits {
		pt, v := isPriceTypeValid(adUnit.PriceType)
		if v {
			valid = v
			if pt == priceTypeGross {
				priceType = pt
				break
			}
		}
	}

	if valid {
		priceTypeParameter = priceType
	}
	return priceTypeParameter
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

	cookie := make([]string, 0, 2)
	if r.userId != "" {
		cookie = append(cookie, fmt.Sprintf("uid=%s", r.userId))
	}
	if r.digitrust != nil {
		if digitrustBytes, err := json.Marshal(r.digitrust); err == nil {
			digitrust := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(digitrustBytes)
			// Cookie name and structure are described here: https://github.com/digi-trust/dt-cdn/wiki/Cookies-for-Platforms
			cookie = append(cookie, fmt.Sprintf("DigiTrust.v1.identity=%s", digitrust))
		}
	}
	header.Set("Cookie", strings.Join(cookie, ";"))

	return header
}

func parseAdformBids(response []byte) ([]*adformBid, error) {
	var bids []*adformBid
	if err := json.Unmarshal(response, &bids); err != nil {
		return nil, &errortypes.BadServerResponse{
			Message: err.Error(),
		}
	}

	return bids, nil
}

// BIDDER Interface

func NewAdformBidder(client *http.Client, endpointURL string) *AdformAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	var uriObj *url.URL
	uriObj, err := url.Parse(endpointURL)
	if err != nil {
		panic(fmt.Sprintf("Incorrect Adform request url %s, check the configuration, please.", endpointURL))
	}

	return &AdformAdapter{
		http:    a,
		URL:     uriObj,
		version: "0.1.2",
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
		params, _, _, err := jsonparser.Get(imp.Ext, "bidder")
		if err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		var adformAdUnit adformAdUnit
		if err := json.Unmarshal(params, &adformAdUnit); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		mid, err := adformAdUnit.MasterTagId.Int64()
		if err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		if mid <= 0 {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("master tag(placement) id is invalid=%s", adformAdUnit.MasterTagId),
			})
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

	gdprApplies := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
		}
		if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
			gdprApplies = strconv.Itoa(int(*extRegs.GDPR))
		}
	}

	consent := ""
	var digitrustData *openrtb_ext.ExtUserDigiTrust
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			consent = extUser.Consent
			digitrustData = extUser.DigiTrust
		}
	}

	var digitrust *adformDigitrust = nil
	if digitrustData != nil {
		digitrust = new(adformDigitrust)
		digitrust.Id = digitrustData.ID
		digitrust.Keyv = digitrustData.KeyV
		digitrust.Version = 1
		digitrust.Privacy = adformDigitrustPrivacy{
			Optout: digitrustData.Pref != 0,
		}
	}

	cur := "USD"
	if request.Cur != nil && len(request.Cur) > 0 {
		/* If USD is one of the supported currencies, then we should send that to the adserver */
		usdSupported := false
		for _, c := range request.Cur {
			if c == "USD" {
				usdSupported = true
				break
			}
		}

		/* If USD is not a supported currency, then we'll just choose the top level currency */
		if usdSupported == false {
			cur = request.Cur[0]
		}
	}

	return &adformRequest{
		adUnits:       adUnits,
		ip:            getIPSafely(request.Device),
		advertisingId: getIFASafely(request.Device),
		userAgent:     getUASafely(request.Device),
		isSecure:      secure,
		referer:       referer,
		userId:        getBuyerUIDSafely(request.User),
		tid:           tid,
		gdprApplies:   gdprApplies,
		consent:       consent,
		digitrust:     digitrust,
		cur:           cur,
	}, errors
}

func getIPSafely(device *openrtb.Device) string {
	if device == nil {
		return ""
	}
	return device.IP
}

func getIFASafely(device *openrtb.Device) string {
	if device == nil {
		return ""
	}
	return device.IFA
}

func getUASafely(device *openrtb.Device) string {
	if device == nil {
		return ""
	}
	return device.UA
}

func getBuyerUIDSafely(user *openrtb.User) string {
	if user == nil {
		return ""
	}
	return user.BuyerUID
}

func (a *AdformAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	adformOutput, err := parseAdformBids(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	bidResponse := toOpenRtbBidResponse(adformOutput, internalRequest)

	return bidResponse, nil
}

func toOpenRtbBidResponse(adformBids []*adformBid, r *openrtb.BidRequest) *adapters.BidderResponse {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adformBids))

	if len(adformBids) > 0 {
		bidResponse.Currency = adformBids[0].Currency
	}

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
			CrID:   bid.CreativeId,
		}

		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{Bid: &openRtbBid, BidType: openrtb_ext.BidTypeBanner})
	}

	return bidResponse
}
