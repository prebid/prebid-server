package adgeneration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/version"
)

// To keep the request/response format in parity with Prebid.js v1.6.6
// (modules/adgenerationBidAdapter.js), this adapter targets the /adgen/prebid
// endpoint (POST with a JSON body). Only id / posall / sdktype are sent as URL
// query parameters; everything else travels in the ortb body.

var (
	newlineRegex     = regexp.MustCompile(`\r?\n`)
	closingBodyRegex = regexp.MustCompile(`<\/\s?body>`)
)

type adapter struct {
	endpoint        string
	version         string
	defaultCurrency string
}

// adgRequestBody is the JSON structure of the POST body. It mirrors the
// Prebid.js `data` object (currency / pbver / sdkname / adapterver / ortb / imark).
type adgRequestBody struct {
	Currency   string              `json:"currency"`
	Pbver      string              `json:"pbver"`
	Sdkname    string              `json:"sdkname"`
	Adapterver string              `json:"adapterver"`
	Ortb       openrtb2.BidRequest `json:"ortb"`
	// imark is set to 1 only for non-native (i.e. banner) requests. This mirrors
	// the Prebid.js adapter, whose comment notes it must be revisited if support
	// for other media types such as video is added.
	Imark int `json:"imark,omitempty"`
}

// adgServerResponse is the response format from the backend
// (d.socdm.com/adgen/prebid). Prebid.js reads body.results[0], so results is
// treated as the primary source.
type adgServerResponse struct {
	Locationid     string             `json:"locationid"`
	LocationParams *adgLocationParams `json:"location_params,omitempty"`
	Results        []adgResult        `json:"results"`
}

type adgLocationParams struct {
	Option *adgLocationOption `json:"option,omitempty"`
}

type adgLocationOption struct {
	AdType string `json:"ad_type,omitempty"`
}

type adgResult struct {
	Ad         string              `json:"ad"`
	Beacon     string              `json:"beacon"`
	Beaconurl  string              `json:"beaconurl"`
	Cpm        float64             `json:"cpm"`
	Creativeid string              `json:"creativeid"`
	Dealid     string              `json:"dealid"`
	H          uint64              `json:"h"`
	W          uint64              `json:"w"`
	Ttl        uint64              `json:"ttl"`
	Vastxml    string              `json:"vastxml,omitempty"`
	LandingUrl string              `json:"landing_url"`
	Scheduleid string              `json:"scheduleid"`
	Adomain    []string            `json:"adomain,omitempty"`
	Native     jsonutil.RawMessage `json:"native,omitempty"`
}

func (adg *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No impression in the bid request"}}
	}

	bidRequestArray := make([]*adapters.RequestData, 0, len(request.Imp))
	var errs []error

	// Prebid.js issues one request per imp; Prebid Server does the same.
	for index := range request.Imp {
		req, err := adg.buildRequest(request, index)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		bidRequestArray = append(bidRequestArray, req)
	}

	return bidRequestArray, errs
}

func (adg *adapter) buildRequest(request *openrtb2.BidRequest, index int) (*adapters.RequestData, error) {
	imp := request.Imp[index]
	adgExt, err := unmarshalExtImpAdgeneration(&imp)
	if err != nil {
		return nil, &errortypes.BadInput{Message: err.Error()}
	}

	uri, err := adg.buildUri(adgExt.Id, request)
	if err != nil {
		return nil, &errortypes.BadInput{Message: err.Error()}
	}

	body, err := adg.buildBody(request, imp)
	if err != nil {
		return nil, err
	}

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: buildHeaders(request),
		ImpIDs:  []string{imp.ID},
	}, nil
}

func buildHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}
		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}
	return headers
}

func (adg *adapter) buildUri(id string, request *openrtb2.BidRequest) (string, error) {
	uriObj, err := url.Parse(adg.endpoint)
	if err != nil {
		return "", err
	}
	v := url.Values{}
	v.Set("id", id)
	// posall=SSPLOC mirrors the query parameter sent by the Prebid.js adapter.
	v.Set("posall", "SSPLOC")
	v.Set("sdktype", detectSdkType(request))
	uriObj.RawQuery = v.Encode()
	return uriObj.String(), nil
}

// detectSdkType derives the sdktype from the request origin (channel) and
// device.os. The backend `/adgen/prebid` switches its delivery logic on
// sdktype, so web traffic gets "0", Prebid Mobile (Android) gets "1", and
// Prebid Mobile (iOS) gets "2". Prebid.js (client-side header bidding) always
// sends "0", but PBS is reached through multiple paths (PBJS+PBS, or PBS-only
// via a Mobile SDK / AMP), so it must detect the origin.
//
// Resolution order:
//  1. ext.prebid.channel.name == "app" -> mobile SDK
//  2. no channel: fall back to the presence of BidRequest.App (App means mobile)
//  3. otherwise -> web (sdktype "0")
//
// When 1 or 2 matches, device.os selects 1/2; an unknown OS yields "0".
func detectSdkType(request *openrtb2.BidRequest) string {
	if !isAppContext(request) {
		return "0"
	}
	if request.Device != nil {
		switch strings.ToLower(request.Device.OS) {
		case "android":
			return "1"
		case "ios":
			return "2"
		}
	}
	return "0"
}

func isAppContext(request *openrtb2.BidRequest) bool {
	if name := requestChannelName(request); name != "" {
		return strings.EqualFold(name, "app")
	}
	// Fallback when no channel is present: treat it as a mobile app if
	// BidRequest.App is set. (AMP / web rarely populate App, whereas the Prebid
	// Mobile SDK does.)
	return request.App != nil
}

func requestChannelName(request *openrtb2.BidRequest) string {
	if request == nil || len(request.Ext) == 0 {
		return ""
	}
	var reqExt openrtb_ext.ExtRequest
	if err := jsonutil.Unmarshal(request.Ext, &reqExt); err != nil {
		return ""
	}
	if reqExt.Prebid.Channel == nil {
		return ""
	}
	return reqExt.Prebid.Channel.Name
}

func (adg *adapter) buildBody(request *openrtb2.BidRequest, imp openrtb2.Imp) ([]byte, error) {
	// ortb carries a BidRequest reduced to a single imp (same as Prebid.js). The
	// other fields of the original request (site/app/device/user/source/regs/ext,
	// etc.) are preserved as-is. The bidder info does not declare OpenRTB 2.6, so
	// PBS down-converts schain/eids/GDPR/USP to their 2.5 ext locations while SUA
	// and GPP stay in place. This matches the shape Prebid.js sends after its own
	// FPD normalization (source.ext.schain, user.ext.eids, device.sua, regs.gpp).
	ortbReq := *request
	// imp.ext is forwarded unchanged (bidder params under imp.ext.bidder). The
	// backend reads the placement ID from the "id" query parameter.
	ortbReq.Imp = []openrtb2.Imp{imp}

	pbver := version.Ver
	if pbver == "" {
		pbver = version.VerUnknown
	}

	body := adgRequestBody{
		Currency:   adg.getCurrency(request),
		Pbver:      pbver,
		Sdkname:    "prebidserver",
		Adapterver: adg.version,
		Ortb:       ortbReq,
	}
	// imark: mirrors Prebid.js, which sets it to 1 for non-native requests.
	if imp.Native == nil {
		body.Imark = 1
	}

	return jsonutil.Marshal(body)
}

func unmarshalExtImpAdgeneration(imp *openrtb2.Imp) (*openrtb_ext.ExtImpAdgeneration, error) {
	var bidderExt adapters.ExtImpBidder
	var adgExt openrtb_ext.ExtImpAdgeneration
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adgExt); err != nil {
		return nil, err
	}
	if adgExt.Id == "" {
		return nil, errors.New("No Location ID in ExtImpAdgeneration.")
	}
	return &adgExt, nil
}

// getCurrency follows the same either/or logic as Prebid.js
// (adgenerationBidAdapter.js: getCurrencyType): return "USD" if request.Cur
// contains USD, otherwise "JPY". Falling back to the first listed currency is
// intentionally not supported (passing EUR/GBP etc. through is out of spec).
func (adg *adapter) getCurrency(request *openrtb2.BidRequest) string {
	for _, c := range request.Cur {
		if strings.EqualFold(c, "USD") {
			return "USD"
		}
	}
	return adg.defaultCurrency
}

func (adg *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp adgServerResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	if len(bidResp.Results) == 0 {
		return nil, nil
	}

	// Like Prebid.js, only results[0] is used (one imp per request).
	adResult := bidResp.Results[0]

	// Prebid.js references bidRequests.data.ortb.imp[0] directly, so we do the
	// same: take imp[0].id from the sent body and look up the matching imp. This
	// avoids a silent no-bid when the backend omits locationid or returns a
	// mismatched value.
	if externalRequest == nil || len(externalRequest.Body) == 0 {
		return nil, nil
	}
	var sentBody adgRequestBody
	if err := jsonutil.Unmarshal(externalRequest.Body, &sentBody); err != nil {
		return nil, []error{err}
	}
	if len(sentBody.Ortb.Imp) == 0 {
		return nil, nil
	}
	targetImpID := sentBody.Ortb.Imp[0].ID
	var matchedImp *openrtb2.Imp
	for i := range internalRequest.Imp {
		if internalRequest.Imp[i].ID == targetImpID {
			matchedImp = &internalRequest.Imp[i]
			break
		}
	}
	if matchedImp == nil {
		return nil, nil
	}

	bidType, adm, err := buildAdMarkup(&adResult, bidResp.LocationParams, matchedImp)
	if err != nil {
		return nil, []error{err}
	}

	bid := openrtb2.Bid{
		// locationid is the same for every bid of a placement, so derive a
		// deterministic ID that is unique per imp within the auction instead.
		ID:     internalRequest.ID + "-" + matchedImp.ID,
		ImpID:  matchedImp.ID,
		AdM:    adm,
		Price:  adResult.Cpm,
		W:      int64(adResult.W),
		H:      int64(adResult.H),
		CrID:   adResult.Creativeid,
		DealID: adResult.Dealid,
	}
	if len(adResult.Adomain) > 0 {
		bid.ADomain = adResult.Adomain
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	// The backend has no currency field and prices in the currency sent in the request body.
	bidResponse.Currency = adg.getCurrency(internalRequest)
	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     &bid,
		BidType: bidType,
	})
	return bidResponse, nil
}

// buildAdMarkup builds the AdM from results[0]. A native response takes
// precedence; otherwise it is returned as a banner (injecting a video tag when
// vastxml is present).
func buildAdMarkup(adResult *adgResult, locationParams *adgLocationParams, imp *openrtb2.Imp) (openrtb_ext.BidType, string, error) {
	// Native: assumes the native object returned by the backend is compatible
	// with an OpenRTB native response ({"native": {...assets, link,
	// imptrackers...}}). Like Prebid.js (isNative), it is treated as native only
	// when assets is non-empty.
	if len(adResult.Native) > 0 && imp.Native != nil && hasNativeAssets(adResult.Native) {
		// AdM is the JSON string of the OpenRTB native admarkup. Whether the
		// backend returns {"native": {...}} or the assets at the top level, it is
		// normalized to {"native":{...}} and beaconurl is appended to imptrackers
		// (matching Prebid.js createNativeAd, which pushes beaconurl onto
		// impressionTrackers).
		admBytes, err := wrapNativeAdm(adResult.Native, adResult.Beaconurl)
		if err != nil {
			return "", "", err
		}
		return openrtb_ext.BidTypeNative, string(admBytes), nil
	}
	// The response is not usable as native. Falling back to banner is only
	// valid when the imp actually requested a banner.
	if imp.Banner == nil {
		return "", "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("no usable native assets in the response for imp %s and the imp has no banner", imp.ID),
		}
	}

	// Banner / Video-in-Banner
	ad := adResult.Ad
	if adResult.Vastxml != "" {
		// Prebid.js injects the ADGBrowserM tag when
		// location_params.option.ad_type === "upper_billboard"; otherwise it uses
		// the APV tag.
		if isUpperBillboard(locationParams) {
			ad = wrapWithADGBrowserM(adResult.Vastxml, extractMarginTop(imp))
		} else {
			ad = wrapWithAPV(imp.ID, adResult.Vastxml)
		}
	}
	ad = appendChildToBody(ad, adResult.Beacon)
	if unwrapped := removeWrapper(ad); unwrapped != "" {
		ad = unwrapped
	}
	return openrtb_ext.BidTypeBanner, ad, nil
}

// hasNativeAssets reports whether the raw JSON of results[0].native contains at
// least one entry in assets[]. This matches Prebid.js isNative()
// (adResult.native.assets.length > 0) and accepts both the {"native":{...}} and
// top-level {assets:...} shapes.
func hasNativeAssets(raw jsonutil.RawMessage) bool {
	var top map[string]jsonutil.RawMessage
	if err := jsonutil.Unmarshal(raw, &top); err != nil {
		return false
	}
	var assets jsonutil.RawMessage
	if inner, ok := top["native"]; ok {
		var nat map[string]jsonutil.RawMessage
		if err := jsonutil.Unmarshal(inner, &nat); err != nil {
			return false
		}
		assets = nat["assets"]
	} else {
		assets = top["assets"]
	}
	if len(assets) == 0 {
		return false
	}
	var arr []jsonutil.RawMessage
	if err := jsonutil.Unmarshal(assets, &arr); err != nil {
		return false
	}
	return len(arr) > 0
}

// wrapNativeAdm wraps the raw JSON of results[0].native for use as AdM and
// appends beaconUrl to native.imptrackers. It absorbs both the case where the
// backend already returns {"native":{...}} and the case where it returns the
// assets at the top level.
func wrapNativeAdm(raw jsonutil.RawMessage, beaconUrl string) ([]byte, error) {
	var top map[string]jsonutil.RawMessage
	if err := jsonutil.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	var native map[string]jsonutil.RawMessage
	if inner, ok := top["native"]; ok {
		if err := jsonutil.Unmarshal(inner, &native); err != nil {
			return nil, err
		}
	} else {
		native = top
	}

	if beaconUrl != "" {
		var trackers []string
		if rawTrackers, ok := native["imptrackers"]; ok {
			if err := jsonutil.Unmarshal(rawTrackers, &trackers); err != nil {
				return nil, err
			}
		}
		duplicate := false
		for _, t := range trackers {
			if t == beaconUrl {
				duplicate = true
				break
			}
		}
		if !duplicate {
			trackers = append(trackers, beaconUrl)
			encoded, err := jsonutil.Marshal(trackers)
			if err != nil {
				return nil, err
			}
			native["imptrackers"] = encoded
		}
	}

	nativeBytes, err := jsonutil.Marshal(native)
	if err != nil {
		return nil, err
	}
	// Compact the result so that the adm does not depend on the whitespace of the
	// backend response (RawMessage values are copied as-is by jsonutil.Marshal
	// unless the process registered jsonutil.RawMessageExtension).
	var adm bytes.Buffer
	if err := json.Compact(&adm, []byte(`{"native":`+string(nativeBytes)+`}`)); err != nil {
		return nil, err
	}
	return adm.Bytes(), nil
}

func isUpperBillboard(p *adgLocationParams) bool {
	if p == nil || p.Option == nil {
		return false
	}
	return p.Option.AdType == "upper_billboard"
}

// encodeVastForJS percent-encodes VAST XML (every non-unreserved byte becomes
// %XX) so that the JS side can restore it with decodeURIComponent. If the adm
// contains a raw "<VAST" string, Prebid Mobile iOS (PBMTransactionFactory)
// misidentifies the HTML banner as a VAST creative and fails with "VAST Parsing
// failed", so VAST embedded in a JS string literal must always be encoded.
// (decodeURIComponent interprets %XX as UTF-8, so multibyte input is safe too.)
func encodeVastForJS(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func wrapWithAPV(impID, vastxml string) string {
	replaced := newlineRegex.ReplaceAllString(vastxml, "")
	// imp.id comes from the request, so escape it for the HTML attribute and the
	// JS string literal respectively.
	return "<body><div id=\"apvad-" + html.EscapeString(impID) + "\"></div>" +
		"<script type=\"text/javascript\" id=\"apv\" src=\"https://cdn.apvdr.com/js/VideoAd.min.js\"></script>" +
		"<script type=\"text/javascript\"> (function(){ new APV.VideoAd({s:\"" + template.JSEscapeString(impID) + "\"}).load(decodeURIComponent('" + encodeVastForJS(replaced) + "')); })(); </script>" +
		"</body>"
}

func wrapWithADGBrowserM(vastxml, marginTop string) string {
	// Prebid.js passes bidder params.marginTop to ADGBrowserM.init({marginTop}).
	// In Prebid Server it lives at imp.ext.bidder.marginTop
	// (ExtImpAdgeneration.MarginTop). When unset it defaults to '0', same as
	// Prebid.js.
	if marginTop == "" {
		marginTop = "0"
	}
	// marginTop is a publisher-supplied string, so escape it before placing it
	// inside a JS single-quoted literal (e.g. ' becomes \' and < becomes \u003C).
	// Plain values such as "10" or "10px" are left unchanged.
	marginTop = template.JSEscapeString(marginTop)
	replaced := newlineRegex.ReplaceAllString(vastxml, "")
	return "<body>" +
		"<script type=\"text/javascript\" src=\"https://i.socdm.com/sdk/js/adg-browser-m.js\"></script>" +
		"<script type=\"text/javascript\">window.ADGBrowserM.init({vastXml: decodeURIComponent('" + encodeVastForJS(replaced) + "'), marginTop: '" + marginTop + "'});</script>" +
		"</body>"
}

// extractMarginTop extracts imp.ext.bidder.marginTop. It returns an empty string on failure.
func extractMarginTop(imp *openrtb2.Imp) string {
	if imp == nil || len(imp.Ext) == 0 {
		return ""
	}
	adgExt, err := unmarshalExtImpAdgeneration(imp)
	if err != nil {
		return ""
	}
	return adgExt.MarginTop
}

// appendChildToBody inserts data before the first closing body tag, like the
// non-global String.replace in Prebid.js. data is inserted literally.
func appendChildToBody(ad string, data string) string {
	loc := closingBodyRegex.FindStringIndex(ad)
	if loc == nil {
		return ad
	}
	return ad[:loc[0]] + data + "</body>" + ad[loc[1]:]
}

func removeWrapper(ad string) string {
	bodyIndex := strings.Index(ad, "<body>")
	lastBodyIndex := strings.LastIndex(ad, "</body>")
	if bodyIndex == -1 || lastBodyIndex == -1 {
		return ""
	}
	str := strings.TrimSpace(strings.Replace(strings.Replace(ad[bodyIndex:lastBodyIndex], "<body>", "", 1), "</body>", "", 1))
	return str
}

// Builder builds a new instance of the Adgeneration adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		config.Endpoint,
		// Aligned with Prebid.js v1.6.6 (ADGENE_PREBID_VERSION); managed as the shared ADG protocol version.
		"1.6.6",
		"JPY",
	}
	return bidder, nil
}
