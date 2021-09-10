package appnexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const defaultPlatformID int = 5

type AppNexusAdapter struct {
	http           *adapters.HTTPAdapter
	URI            string
	iabCategoryMap map[string]string
	hbSource       int
}

// used for cookies and such
func (a *AppNexusAdapter) Name() string {
	return "adnxs"
}

func (a *AppNexusAdapter) SkipNoCookies() bool {
	return false
}

type KeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

type appnexusAdapterOptions struct {
	IabCategories map[string]string `json:"iab_categories"`
}

type appnexusParams struct {
	LegacyPlacementId       int             `json:"placementId"`
	LegacyInvCode           string          `json:"invCode"`
	LegacyTrafficSourceCode string          `json:"trafficSourceCode"`
	PlacementId             int             `json:"placement_id"`
	InvCode                 string          `json:"inv_code"`
	Member                  string          `json:"member"`
	Keywords                []KeyVal        `json:"keywords"`
	TrafficSourceCode       string          `json:"traffic_source_code"`
	Reserve                 float64         `json:"reserve"`
	Position                string          `json:"position"`
	UsePmtRule              *bool           `json:"use_pmt_rule"`
	PrivateSizes            json.RawMessage `json:"private_sizes"`
}

type appnexusImpExtAppnexus struct {
	PlacementID       int             `json:"placement_id,omitempty"`
	Keywords          string          `json:"keywords,omitempty"`
	TrafficSourceCode string          `json:"traffic_source_code,omitempty"`
	UsePmtRule        *bool           `json:"use_pmt_rule,omitempty"`
	PrivateSizes      json.RawMessage `json:"private_sizes,omitempty"`
}

type appnexusImpExt struct {
	Appnexus appnexusImpExtAppnexus `json:"appnexus"`
}

type appnexusBidExtVideo struct {
	Duration int `json:"duration"`
}

type appnexusBidExtCreative struct {
	Video appnexusBidExtVideo `json:"video"`
}

type appnexusBidExtAppnexus struct {
	BidType       int                    `json:"bid_ad_type"`
	BrandId       int                    `json:"brand_id"`
	BrandCategory int                    `json:"brand_category_id"`
	CreativeInfo  appnexusBidExtCreative `json:"creative_info"`
	DealPriority  int                    `json:"deal_priority"`
}

type appnexusBidExt struct {
	Appnexus appnexusBidExtAppnexus `json:"appnexus"`
}

type appnexusReqExtAppnexus struct {
	IncludeBrandCategory    *bool  `json:"include_brand_category,omitempty"`
	BrandCategoryUniqueness *bool  `json:"brand_category_uniqueness,omitempty"`
	IsAMP                   int    `json:"is_amp,omitempty"`
	HeaderBiddingSource     int    `json:"hb_source,omitempty"`
	AdPodId                 string `json:"adpod_id,omitempty"`
}

// Full request extension including appnexus extension object
type appnexusReqExt struct {
	openrtb_ext.ExtRequest
	Appnexus *appnexusReqExtAppnexus `json:"appnexus,omitempty"`
}

var maxImpsPerReq = 10

func (a *AppNexusAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	anReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes)

	if err != nil {
		return nil, err
	}
	uri := a.URI
	for i, unit := range bidder.AdUnits {
		var params appnexusParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		// Accept legacy Appnexus parameters if we don't have modern ones
		// Don't worry if both is set as validation rules should prevent, and this is temporary anyway.
		if params.PlacementId == 0 && params.LegacyPlacementId != 0 {
			params.PlacementId = params.LegacyPlacementId
		}
		if params.InvCode == "" && params.LegacyInvCode != "" {
			params.InvCode = params.LegacyInvCode
		}
		if params.TrafficSourceCode == "" && params.LegacyTrafficSourceCode != "" {
			params.TrafficSourceCode = params.LegacyTrafficSourceCode
		}

		if params.PlacementId == 0 && (params.InvCode == "" || params.Member == "") {
			return nil, &errortypes.BadInput{
				Message: "No placement or member+invcode provided",
			}
		}

		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(anReq.Imp) <= i {
			break
		}
		if params.InvCode != "" {
			anReq.Imp[i].TagID = params.InvCode
			if params.Member != "" {
				// this assumes that the same member ID is used across all tags, which should be the case
				uri = appendMemberId(a.URI, params.Member)
			}

		}
		if params.Reserve > 0 {
			anReq.Imp[i].BidFloor = params.Reserve // TODO: we need to factor in currency here if non-USD
		}
		if anReq.Imp[i].Banner != nil && params.Position != "" {
			if params.Position == "above" {
				anReq.Imp[i].Banner.Pos = openrtb2.AdPositionAboveTheFold.Ptr()
			} else if params.Position == "below" {
				anReq.Imp[i].Banner.Pos = openrtb2.AdPositionBelowTheFold.Ptr()
			}
		}

		kvs := make([]string, 0, len(params.Keywords)*2)
		for _, kv := range params.Keywords {
			if len(kv.Values) == 0 {
				kvs = append(kvs, kv.Key)
			} else {
				for _, val := range kv.Values {
					kvs = append(kvs, fmt.Sprintf("%s=%s", kv.Key, val))
				}

			}
		}

		keywordStr := strings.Join(kvs, ",")

		impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{
			PlacementID:       params.PlacementId,
			TrafficSourceCode: params.TrafficSourceCode,
			Keywords:          keywordStr,
			UsePmtRule:        params.UsePmtRule,
			PrivateSizes:      params.PrivateSizes,
		}}
		anReq.Imp[i].Ext, err = json.Marshal(&impExt)
	}

	reqJSON, err := json.Marshal(anReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: uri,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = anResp.StatusCode

	if anResp.StatusCode == 204 {
		return nil, nil
	}

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)

	if anResp.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", anResp.StatusCode, responseBody),
		}
	}

	if anResp.StatusCode != http.StatusOK {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", anResp.StatusCode, responseBody),
		}
	}

	if req.IsDebug {
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb2.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
				NURL:        bid.NURL,
			}

			var impExt appnexusBidExt
			if err := json.Unmarshal(bid.Ext, &impExt); err == nil {
				if mediaType, err := getMediaTypeForBid(&impExt); err == nil {
					pbid.CreativeMediaType = string(mediaType)
					bids = append(bids, &pbid)
				}
			}
		}
	}

	return bids, nil
}

func (a *AppNexusAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	memberIds := make(map[string]bool)
	errs := make([]error, 0, len(request.Imp))

	// AppNexus openrtb2 endpoint expects imp.displaymanagerver to be populated, but some SDKs will put it in imp.ext.prebid instead
	var defaultDisplayManagerVer string
	if request.App != nil {
		source, err1 := jsonparser.GetString(request.App.Ext, openrtb_ext.PrebidExtKey, "source")
		version, err2 := jsonparser.GetString(request.App.Ext, openrtb_ext.PrebidExtKey, "version")
		if (err1 == nil) && (err2 == nil) {
			defaultDisplayManagerVer = fmt.Sprintf("%s-%s", source, version)
		}
	}
	var adPodId *bool

	for i := 0; i < len(request.Imp); i++ {
		memberId, impAdPodId, err := preprocess(&request.Imp[i], defaultDisplayManagerVer)
		if memberId != "" {
			memberIds[memberId] = true
		}
		if adPodId == nil {
			adPodId = &impAdPodId
		} else if *adPodId != impAdPodId {
			errs = append(errs, errors.New("generate ad pod option should be same for all pods in request"))
			return nil, errs
		}

		// If the preprocessing failed, the server won't be able to bid on this Imp. Delete it, and note the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	thisURI := a.URI

	// The Appnexus API requires a Member ID in the URL. This means the request may fail if
	// different impressions have different member IDs.
	// Check for this condition, and log an error if it's a problem.
	if len(memberIds) > 0 {
		uniqueIds := keys(memberIds)
		memberId := uniqueIds[0]
		thisURI = appendMemberId(thisURI, memberId)

		if len(uniqueIds) > 1 {
			errs = append(errs, fmt.Errorf("All request.imp[i].ext.appnexus.member params must match. Request contained: %v", uniqueIds))
		}
	}

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	// Add Appnexus request level extension
	var isAMP, isVIDEO int
	if reqInfo.PbsEntryPoint == metrics.ReqTypeAMP {
		isAMP = 1
	} else if reqInfo.PbsEntryPoint == metrics.ReqTypeVideo {
		isVIDEO = 1
	}

	var reqExt appnexusReqExt
	if len(request.Ext) > 0 {
		if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
			errs = append(errs, err)
			return nil, errs
		}
	}
	if reqExt.Appnexus == nil {
		reqExt.Appnexus = &appnexusReqExtAppnexus{}
	}
	includeBrandCategory := reqExt.Prebid.Targeting != nil && reqExt.Prebid.Targeting.IncludeBrandCategory != nil
	if includeBrandCategory {
		reqExt.Appnexus.BrandCategoryUniqueness = &includeBrandCategory
		reqExt.Appnexus.IncludeBrandCategory = &includeBrandCategory
	}
	reqExt.Appnexus.IsAMP = isAMP
	reqExt.Appnexus.HeaderBiddingSource = a.hbSource + isVIDEO

	imps := request.Imp

	// For long form requests if adpodId feature enabled, adpod_id must be sent downstream.
	// Adpod id is a unique identifier for pod
	// All impressions in the same pod must have the same pod id in request extension
	// For this all impressions in  request should belong to the same pod
	// If impressions number per pod is more than maxImpsPerReq - divide those imps to several requests but keep pod id the same
	// If  adpodId feature disabled and impressions number per pod is more than maxImpsPerReq  - divide those imps to several requests but do not include ad pod id
	if isVIDEO == 1 && *adPodId {
		podImps := groupByPods(imps)

		requests := make([]*adapters.RequestData, 0, len(podImps))
		for _, podImps := range podImps {
			reqExt.Appnexus.AdPodId = generatePodId()

			reqs, errors := splitRequests(podImps, request, reqExt, thisURI, errs)
			requests = append(requests, reqs...)
			errs = append(errs, errors...)
		}
		return requests, errs
	}

	return splitRequests(imps, request, reqExt, thisURI, errs)
}

func generatePodId() string {
	val := rand.Int63()
	return fmt.Sprint(val)
}

func groupByPods(imps []openrtb2.Imp) map[string]([]openrtb2.Imp) {
	// find number of pods in response
	podImps := make(map[string][]openrtb2.Imp)
	for _, imp := range imps {
		pod := strings.Split(imp.ID, "_")[0]
		podImps[pod] = append(podImps[pod], imp)
	}
	return podImps
}

func marshalAndSetRequestExt(request *openrtb2.BidRequest, requestExtension appnexusReqExt, errs []error) {
	var err error
	request.Ext, err = json.Marshal(requestExtension)
	if err != nil {
		errs = append(errs, err)
	}
}

func splitRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, requestExtension appnexusReqExt, uri string, errs []error) ([]*adapters.RequestData, []error) {

	// Initial capacity for future array of requests, memory optimization.
	// Let's say there are 35 impressions and limit impressions per request equals to 10.
	// In this case we need to create 4 requests with 10, 10, 10 and 5 impressions.
	// With this formula initial capacity=(35+10-1)/10 = 4
	initialCapacity := (len(imps) + maxImpsPerReq - 1) / maxImpsPerReq
	resArr := make([]*adapters.RequestData, 0, initialCapacity)
	startInd := 0
	impsLeft := len(imps) > 0

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	marshalAndSetRequestExt(request, requestExtension, errs)

	for impsLeft {

		endInd := startInd + maxImpsPerReq
		if endInd >= len(imps) {
			endInd = len(imps)
			impsLeft = false
		}
		impsForReq := imps[startInd:endInd]
		request.Imp = impsForReq

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		resArr = append(resArr, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,
		})
		startInd = endInd
	}
	return resArr, errs
}

// get the keys from the map
func keys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// preprocess mutates the imp to get it ready to send to appnexus.
//
// It returns the member param, if it exists, and an error if anything went wrong during the preprocessing.
func preprocess(imp *openrtb2.Imp, defaultDisplayManagerVer string) (string, bool, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", false, err
	}

	var appnexusExt openrtb_ext.ExtImpAppnexus
	if err := json.Unmarshal(bidderExt.Bidder, &appnexusExt); err != nil {
		return "", false, err
	}

	// Accept legacy Appnexus parameters if we don't have modern ones
	// Don't worry if both is set as validation rules should prevent, and this is temporary anyway.
	if appnexusExt.PlacementId == 0 && appnexusExt.LegacyPlacementId != 0 {
		appnexusExt.PlacementId = appnexusExt.LegacyPlacementId
	}
	if appnexusExt.InvCode == "" && appnexusExt.LegacyInvCode != "" {
		appnexusExt.InvCode = appnexusExt.LegacyInvCode
	}
	if appnexusExt.TrafficSourceCode == "" && appnexusExt.LegacyTrafficSourceCode != "" {
		appnexusExt.TrafficSourceCode = appnexusExt.LegacyTrafficSourceCode
	}

	if appnexusExt.PlacementId == 0 && (appnexusExt.InvCode == "" || appnexusExt.Member == "") {
		return "", false, &errortypes.BadInput{
			Message: "No placement or member+invcode provided",
		}
	}

	if appnexusExt.InvCode != "" {
		imp.TagID = appnexusExt.InvCode
	}
	if imp.BidFloor <= 0 && appnexusExt.Reserve > 0 {
		imp.BidFloor = appnexusExt.Reserve // This will be broken for non-USD currency.
	}
	if imp.Banner != nil {
		bannerCopy := *imp.Banner
		if appnexusExt.Position == "above" {
			bannerCopy.Pos = openrtb2.AdPositionAboveTheFold.Ptr()
		} else if appnexusExt.Position == "below" {
			bannerCopy.Pos = openrtb2.AdPositionBelowTheFold.Ptr()
		}

		// Fixes #307
		if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
			firstFormat := bannerCopy.Format[0]
			bannerCopy.W = &(firstFormat.W)
			bannerCopy.H = &(firstFormat.H)
		}
		imp.Banner = &bannerCopy
	}

	// Populate imp.displaymanagerver if the SDK failed to do it.
	if len(imp.DisplayManagerVer) == 0 && len(defaultDisplayManagerVer) > 0 {
		imp.DisplayManagerVer = defaultDisplayManagerVer
	}

	impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{
		PlacementID:       appnexusExt.PlacementId,
		TrafficSourceCode: appnexusExt.TrafficSourceCode,
		Keywords:          makeKeywordStr(appnexusExt.Keywords),
		UsePmtRule:        appnexusExt.UsePmtRule,
		PrivateSizes:      appnexusExt.PrivateSizes,
	}}
	var err error
	if imp.Ext, err = json.Marshal(&impExt); err != nil {
		return appnexusExt.Member, appnexusExt.AdPodId, err
	}

	return appnexusExt.Member, appnexusExt.AdPodId, nil
}

func makeKeywordStr(keywords []*openrtb_ext.ExtImpAppnexusKeyVal) string {
	kvs := make([]string, 0, len(keywords)*2)
	for _, kv := range keywords {
		if len(kv.Values) == 0 {
			kvs = append(kvs, kv.Key)
		} else {
			for _, val := range kv.Values {
				kvs = append(kvs, fmt.Sprintf("%s=%s", kv.Key, val))
			}
		}
	}

	return strings.Join(kvs, ",")
}

func (a *AppNexusAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			var bidExt appnexusBidExt
			if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, err)
			} else {
				if bidType, err := getMediaTypeForBid(&bidExt); err == nil {
					if iabCategory, err := a.getIabCategoryForBid(&bidExt); err == nil {
						bid.Cat = []string{iabCategory}
					} else if len(bid.Cat) > 1 {
						//create empty categories array to force bid to be rejected
						bid.Cat = make([]string, 0, 0)
					}

					impVideo := &openrtb_ext.ExtBidPrebidVideo{
						Duration: bidExt.Appnexus.CreativeInfo.Video.Duration,
					}

					bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
						Bid:          &bid,
						BidType:      bidType,
						BidVideo:     impVideo,
						DealPriority: bidExt.Appnexus.DealPriority,
					})
				} else {
					errs = append(errs, err)
				}
			}
		}
	}
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}
	return bidResponse, errs
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *appnexusBidExt) (openrtb_ext.BidType, error) {
	switch bid.Appnexus.BidType {
	case 0:
		return openrtb_ext.BidTypeBanner, nil
	case 1:
		return openrtb_ext.BidTypeVideo, nil
	case 2:
		return openrtb_ext.BidTypeAudio, nil
	case 3:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unrecognized bid_ad_type in response from appnexus: %d", bid.Appnexus.BidType)
	}
}

// getIabCategoryForBid maps an appnexus brand id to an IAB category.
func (a *AppNexusAdapter) getIabCategoryForBid(bid *appnexusBidExt) (string, error) {
	brandIDString := strconv.Itoa(bid.Appnexus.BrandCategory)
	if iabCategory, ok := a.iabCategoryMap[brandIDString]; ok {
		return iabCategory, nil
	} else {
		return "", fmt.Errorf("category not in map: %s", brandIDString)
	}
}

func appendMemberId(uri string, memberId string) string {
	if strings.Contains(uri, "?") {
		return uri + "&member_id=" + memberId
	}

	return uri + "?member_id=" + memberId
}

// Builder builds a new instance of the AppNexus adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &AppNexusAdapter{
		URI:            config.Endpoint,
		iabCategoryMap: loadCategoryMapFromFileSystem(),
		hbSource:       resolvePlatformID(config.PlatformID),
	}
	return bidder, nil
}

// NewAppNexusLegacyAdapter builds a legacy version of the AppNexus adapter.
func NewAppNexusLegacyAdapter(httpConfig *adapters.HTTPAdapterConfig, endpoint, platformID string) *AppNexusAdapter {
	return &AppNexusAdapter{
		http:           adapters.NewHTTPAdapter(httpConfig),
		URI:            endpoint,
		iabCategoryMap: loadCategoryMapFromFileSystem(),
		hbSource:       resolvePlatformID(platformID),
	}
}

func resolvePlatformID(platformID string) int {
	if len(platformID) > 0 {
		if val, err := strconv.Atoi(platformID); err == nil {
			return val
		}
	}

	return defaultPlatformID
}

func loadCategoryMapFromFileSystem() map[string]string {
	// Load custom options for our adapter (currently just a lookup table to convert appnexus => iab categories)
	opts, err := ioutil.ReadFile("./static/adapter/appnexus/opts.json")
	if err == nil {
		var adapterOptions appnexusAdapterOptions

		if err := json.Unmarshal(opts, &adapterOptions); err == nil {
			return adapterOptions.IabCategories
		}
	}

	return nil
}
