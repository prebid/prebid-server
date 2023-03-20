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
	"github.com/prebid/prebid-server/util/httputil"

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

	// appnexus adapter expects imp.displaymanagerver to be populated in openrtb2 endpoint
	// but some SDKs will put it in imp.ext.prebid instead
	displayManagerVer := buildDefaultDisplayManageVer(request)

	var (
		shouldGenerateAdPodId *bool
		uniqueMemberIds       []string
		memberIds             = make(map[string]struct{})
		errs                  = make([]error, 0, len(request.Imp))
	)

	validImps := []openrtb2.Imp{}
	for i := 0; i < len(request.Imp); i++ {
		// If the preprocessing failed, the server won't be able to bid on this Imp. Delete it, and note the error.
		memberId, shouldGenerateAdPodIdForImp, err := preprocess(&request.Imp[i], displayManagerVer)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if memberId != "" {
			if _, ok := memberIds[memberId]; !ok {
				memberIds[memberId] = struct{}{}
				uniqueMemberIds = append(uniqueMemberIds, memberId)
			}
		}
		if shouldGenerateAdPodId == nil {
			shouldGenerateAdPodId = &shouldGenerateAdPodIdForImp
		} else if *shouldGenerateAdPodId != shouldGenerateAdPodIdForImp {
			errs = append(errs, errors.New("generate ad pod option should be same for all pods in request"))
			return nil, errs
		}

		validImps = append(validImps, request.Imp[i])
	}
	request.Imp = validImps

	requestURI := a.URI
	// The Appnexus API requires a Member ID in the URL. This means the request may fail if
	// different impressions have different member IDs.
	// Check for this condition, and log an error if it's a problem.
	if len(uniqueMemberIds) > 0 {
		requestURI = appendMemberId(requestURI, uniqueMemberIds[0])
		if len(uniqueMemberIds) > 1 {
			errs = append(errs, fmt.Errorf("All request.imp[i].ext.prebid.bidder.appnexus.member params must match. Request contained: %v", uniqueMemberIds))
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
	if isVIDEO == 1 && *shouldGenerateAdPodId {
		requests, errors := buildAdPodRequests(imps, request, reqExt, requestURI)
		return requests, append(errs, errors...)
	}

	requests, errors := splitRequests(imps, request, reqExt, requestURI)
	return requests, append(errs, errors...)
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

func splitRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, requestExtension appnexusReqExt, uri string) ([]*adapters.RequestData, []error) {
	errs := []error{}
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

	var err error
	request.Ext, err = json.Marshal(requestExtension)
	if err != nil {
		errs = append(errs, err)
	}

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
	if appnexusExt.PlacementId == 0 && appnexusExt.DeprecatedPlacementId != 0 {
		appnexusExt.PlacementId = appnexusExt.DeprecatedPlacementId
	}
	if appnexusExt.InvCode == "" && appnexusExt.LegacyInvCode != "" {
		appnexusExt.InvCode = appnexusExt.LegacyInvCode
	}
	if appnexusExt.TrafficSourceCode == "" && appnexusExt.LegacyTrafficSourceCode != "" {
		appnexusExt.TrafficSourceCode = appnexusExt.LegacyTrafficSourceCode
	}
	if appnexusExt.UsePaymentRule == nil && appnexusExt.DeprecatedUsePaymentRule != nil {
		appnexusExt.UsePaymentRule = appnexusExt.DeprecatedUsePaymentRule
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
		PlacementID:       int(appnexusExt.PlacementId),
		TrafficSourceCode: appnexusExt.TrafficSourceCode,
		Keywords:          makeKeywordStr(appnexusExt.Keywords),
		UsePmtRule:        appnexusExt.UsePaymentRule,
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
	if httputil.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := httputil.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var appnexusResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &appnexusResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, sb := range appnexusResponse.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]

			var bidExt appnexusBidExt
			if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, err)
				continue
			}

			bidType, err := getMediaTypeForBid(&bidExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			iabCategory, found := a.findIabCategoryForBid(&bidExt)
			if found {
				bid.Cat = []string{iabCategory}
			} else if len(bid.Cat) > 1 {
				//create empty categories array to force bid to be rejected
				bid.Cat = make([]string, 0)
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:          &bid,
				BidType:      bidType,
				BidVideo:     &openrtb_ext.ExtBidPrebidVideo{Duration: bidExt.Appnexus.CreativeInfo.Video.Duration},
				DealPriority: bidExt.Appnexus.DealPriority,
			})
		}
	}

	if appnexusResponse.Cur != "" {
		bidderResponse.Currency = appnexusResponse.Cur
	}

	return bidderResponse, errs
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
func (a *adapter) findIabCategoryForBid(bid *appnexusBidExt) (string, bool) {
	brandIDString := strconv.Itoa(bid.Appnexus.BrandCategory)
	iabCategory, ok := a.iabCategoryMap[brandIDString]
	return iabCategory, ok
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

func buildDefaultDisplayManageVer(req *openrtb2.BidRequest) string {
	if req.App == nil {
		return ""
	}

	source, err := jsonparser.GetString(req.App.Ext, openrtb_ext.PrebidExtKey, "source")
	if err != nil {
		return ""
	}

	version, err := jsonparser.GetString(req.App.Ext, openrtb_ext.PrebidExtKey, "version")
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s-%s", source, version)
}

func buildAdPodRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, requestExtension appnexusReqExt, uri string) ([]*adapters.RequestData, []error) {
	var errs []error
	podImps := groupByPods(imps)
	requests := make([]*adapters.RequestData, 0, len(podImps))
	for _, podImps := range podImps {
		requestExtension.Appnexus.AdPodId = fmt.Sprint(rand.Int63())

		reqs, errors := splitRequests(podImps, request, requestExtension, uri)
		requests = append(requests, reqs...)
		errs = append(errs, errors...)
	}

	return requests, errs
}
