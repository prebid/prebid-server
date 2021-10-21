package adform

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

const version = "0.1.3"

type AdformAdapter struct {
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
	currency      string
	eids          string
	url           string
}

type adformAdUnit struct {
	MasterTagId json.Number `json:"mid"`
	PriceType   string      `json:"priceType,omitempty"`
	KeyValues   string      `json:"mkv,omitempty"`
	KeyWords    string      `json:"mkw,omitempty"`
	CDims       string      `json:"cdims,omitempty"`
	MinPrice    float64     `json:"minp,omitempty"`
	Url         string      `json:"url,omitempty"`

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
	VastContent  string  `json:"vast_content,omitempty"`
}

const priceTypeGross = "gross"
const priceTypeNet = "net"
const defaultCurrency = "USD"

func isPriceTypeValid(priceType string) (string, bool) {
	pt := strings.ToLower(priceType)
	valid := pt == priceTypeNet || pt == priceTypeGross

	return pt, valid
}

// ADAPTER Interface

// Builder builds a new instance of the Adform adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	uri, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, errors.New("unable to parse endpoint")
	}

	bidder := &AdformAdapter{
		URL:     uri,
		version: version,
	}
	return bidder, nil
}

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
	if r.eids != "" {
		parameters.Add("eids", r.eids)
	}

	if r.url != "" {
		parameters.Add("url", r.url)
	}

	URL := *a.URL
	URL.RawQuery = parameters.Encode()

	uri := URL.String()
	if r.isSecure {
		uri = strings.Replace(uri, "http://", "https://", 1)
	}

	adUnitsParams := make([]string, 0, len(r.adUnits))
	for _, adUnit := range r.adUnits {
		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("mid=%s&rcur=%s", adUnit.MasterTagId, r.currency))
		if adUnit.KeyValues != "" {
			buffer.WriteString(fmt.Sprintf("&mkv=%s", adUnit.KeyValues))
		}
		if adUnit.KeyWords != "" {
			buffer.WriteString(fmt.Sprintf("&mkw=%s", adUnit.KeyWords))
		}
		if adUnit.CDims != "" {
			buffer.WriteString(fmt.Sprintf("&cdims=%s", adUnit.CDims))
		}
		if adUnit.MinPrice > 0 {
			buffer.WriteString(fmt.Sprintf("&minp=%.2f", adUnit.MinPrice))
		}
		adUnitsParams = append(adUnitsParams, base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(buffer.Bytes()))
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

	if r.userId != "" {
		header.Set("Cookie", fmt.Sprintf("uid=%s;", r.userId))
	}

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

func (a *AdformAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

func openRtbToAdformRequest(request *openrtb2.BidRequest) (*adformRequest, []error) {
	adUnits := make([]*adformAdUnit, 0, len(request.Imp))
	errors := make([]error, 0, len(request.Imp))
	secure := false
	url := ""

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

		if url == "" {
			url = adformAdUnit.Url
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
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
		}
		if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
			gdprApplies = strconv.Itoa(int(*extRegs.GDPR))
		}
	}

	eids := ""
	consent := ""
	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			consent = extUser.Consent
			eids = encodeEids(extUser.Eids)
		}
	}

	requestCurrency := defaultCurrency
	if len(request.Cur) != 0 {
		hasDefaultCurrency := false
		for _, c := range request.Cur {
			if defaultCurrency == c {
				hasDefaultCurrency = true
				break
			}
		}
		if !hasDefaultCurrency {
			requestCurrency = request.Cur[0]
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
		currency:      requestCurrency,
		eids:          eids,
		url:           url,
	}, errors
}

func encodeEids(eids []openrtb_ext.ExtUserEid) string {
	if eids == nil {
		return ""
	}

	eidsMap := make(map[string]map[string][]int)
	for _, eid := range eids {
		_, ok := eidsMap[eid.Source]
		if !ok {
			eidsMap[eid.Source] = make(map[string][]int)
		}
		for _, uid := range eid.Uids {
			eidsMap[eid.Source][uid.ID] = append(eidsMap[eid.Source][uid.ID], uid.Atype)
		}
	}

	encodedEids := ""
	if eidsString, err := json.Marshal(eidsMap); err == nil {
		encodedEids = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(eidsString)
	}

	return encodedEids
}

func getIPSafely(device *openrtb2.Device) string {
	if device == nil {
		return ""
	}
	return device.IP
}

func getIFASafely(device *openrtb2.Device) string {
	if device == nil {
		return ""
	}
	return device.IFA
}

func getUASafely(device *openrtb2.Device) string {
	if device == nil {
		return ""
	}
	return device.UA
}

func getBuyerUIDSafely(user *openrtb2.User) string {
	if user == nil {
		return ""
	}
	return user.BuyerUID
}

func (a *AdformAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

func toOpenRtbBidResponse(adformBids []*adformBid, r *openrtb2.BidRequest) *adapters.BidderResponse {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adformBids))
	currency := bidResponse.Currency

	if len(adformBids) > 0 {
		bidResponse.Currency = adformBids[0].Currency
	}

	for i, bid := range adformBids {
		adm, bidType := getAdAndType(bid)
		if adm == "" {
			continue
		}

		openRtbBid := openrtb2.Bid{
			ID:     r.Imp[i].ID,
			ImpID:  r.Imp[i].ID,
			Price:  bid.Price,
			AdM:    adm,
			W:      int64(bid.Width),
			H:      int64(bid.Height),
			DealID: bid.DealId,
			CrID:   bid.CreativeId,
		}

		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{Bid: &openRtbBid, BidType: bidType})
		currency = bid.Currency
	}

	bidResponse.Currency = currency

	return bidResponse
}

func getAdAndType(bid *adformBid) (string, openrtb_ext.BidType) {
	if bid.ResponseType == "banner" {
		return bid.Banner, openrtb_ext.BidTypeBanner
	} else if bid.ResponseType == "vast_content" {
		return bid.VastContent, openrtb_ext.BidTypeVideo
	}
	return "", ""
}
