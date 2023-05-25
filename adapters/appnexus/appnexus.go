package appnexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const defaultPlatformID int = 5

type adapter struct {
	URI            string
	iabCategoryMap map[string]string
	hbSource       int
}

var maxImpsPerReq = 10

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
			errs = append(errs, fmt.Errorf("All request.imp[i].ext.prebid.bidder.appnexus.member params must match. Request contained: %v", uniqueIds))
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
			reqExt.Appnexus.AdPodId = generatePodID()

			reqs, errors := splitRequests(podImps, request, reqExt, thisURI, errs)
			requests = append(requests, reqs...)
			errs = append(errs, errors...)
		}
		return requests, errs
	}

	return splitRequests(imps, request, reqExt, thisURI, errs)
}

func generatePodID() string {
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
			bannerCopy.Pos = adcom1.PositionAboveFold.Ptr()
		} else if appnexusExt.Position == "below" {
			bannerCopy.Pos = adcom1.PositionBelowFold.Ptr()
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

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
						bid.Cat = make([]string, 0)
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
func (a *adapter) getIabCategoryForBid(bid *appnexusBidExt) (string, error) {
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
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		URI: config.Endpoint,
		iabCategoryMap: map[string]string{
			"1": "IAB20-3",
			"9": "IAB5-3",
		},
		hbSource: resolvePlatformID(config.PlatformID),
	}
	return bidder, nil
}

func resolvePlatformID(platformID string) int {
	if len(platformID) > 0 {
		if val, err := strconv.Atoi(platformID); err == nil {
			return val
		}
	}

	return defaultPlatformID
}
