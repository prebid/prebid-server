package appnexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/util/randomutil"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	defaultPlatformID int = 5
	maxImpsPerReq         = 10
)

type adapter struct {
	uri             url.URL
	hbSource        int
	randomGenerator randomutil.RandomGenerator
}

// Builder builds a new instance of the AppNexus adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	uri, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}

	bidder := &adapter{
		uri:             *uri,
		hbSource:        resolvePlatformID(config.PlatformID),
		randomGenerator: randomutil.RandomNumberGenerator{},
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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	// appnexus adapter expects imp.displaymanagerver to be populated in openrtb2 endpoint
	// but some SDKs will put it in imp.ext.prebid instead
	displayManagerVer := buildDisplayManageVer(request)

	var (
		shouldGenerateAdPodId *bool
		uniqueMemberID        string
		errs                  = make([]error, 0, len(request.Imp))
	)

	validImps := []openrtb2.Imp{}
	for i := 0; i < len(request.Imp); i++ {
		appnexusExt, err := validateAndBuildAppNexusExt(&request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := buildRequestImp(&request.Imp[i], &appnexusExt, displayManagerVer); err != nil {
			errs = append(errs, err)
			continue
		}

		memberId := appnexusExt.Member
		if memberId != "" {
			// The Appnexus API requires a Member ID in the URL. This means the request may fail if
			// different impressions have different member IDs.
			// Check for this condition, and log an error if it's a problem.
			if uniqueMemberID == "" {
				uniqueMemberID = memberId
			} else if uniqueMemberID != memberId {
				errs = append(errs, fmt.Errorf("all request.imp[i].ext.prebid.bidder.appnexus.member params must match. Request contained member IDs %s and %s", uniqueMemberID, memberId))
				return nil, errs
			}
		}

		shouldGenerateAdPodIdForImp := appnexusExt.AdPodId
		if shouldGenerateAdPodId == nil {
			shouldGenerateAdPodId = &shouldGenerateAdPodIdForImp
		} else if *shouldGenerateAdPodId != shouldGenerateAdPodIdForImp {
			errs = append(errs, errors.New("generate ad pod option should be same for all pods in request"))
			return nil, errs
		}

		validImps = append(validImps, request.Imp[i])
	}
	request.Imp = validImps

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	requestURI := a.uri

	if uniqueMemberID != "" {
		requestURI = appendMemberId(requestURI, uniqueMemberID)
	}

	// Add Appnexus request level extension
	var isAMP, isVIDEO int
	if reqInfo.PbsEntryPoint == metrics.ReqTypeAMP {
		isAMP = 1
	} else if reqInfo.PbsEntryPoint == metrics.ReqTypeVideo {
		isVIDEO = 1
	}

	reqExt, err := a.getRequestExt(request.Ext, isAMP, isVIDEO)
	if err != nil {
		return nil, append(errs, err)
	}
	// For long form requests if adpodId feature enabled, adpod_id must be sent downstream.
	// Adpod id is a unique identifier for pod
	// All impressions in the same pod must have the same pod id in request extension
	// For this all impressions in  request should belong to the same pod
	// If impressions number per pod is more than maxImpsPerReq - divide those imps to several requests but keep pod id the same
	// If  adpodId feature disabled and impressions number per pod is more than maxImpsPerReq  - divide those imps to several requests but do not include ad pod id
	if isVIDEO == 1 && *shouldGenerateAdPodId {
		requests, errors := a.buildAdPodRequests(request.Imp, request, reqExt, requestURI.String())
		return requests, append(errs, errors...)
	}

	requests, errors := splitRequests(request.Imp, request, reqExt, requestURI.String())
	return requests, append(errs, errors...)
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
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

			var bidExt bidExt
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
				bid.Cat = []string{}
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

func (a *adapter) getRequestExt(ext json.RawMessage, isAMP int, isVIDEO int) (bidReqExt, error) {
	var reqExt bidReqExt
	if len(ext) > 0 {
		if err := json.Unmarshal(ext, &reqExt); err != nil {
			return bidReqExt{}, err
		}
	}
	if reqExt.Appnexus == nil {
		reqExt.Appnexus = &bidReqExtAppnexus{}
	}
	includeBrandCategory := reqExt.Prebid.Targeting != nil && reqExt.Prebid.Targeting.IncludeBrandCategory != nil
	if includeBrandCategory {
		reqExt.Appnexus.BrandCategoryUniqueness = &includeBrandCategory
		reqExt.Appnexus.IncludeBrandCategory = &includeBrandCategory
	}
	reqExt.Appnexus.IsAMP = isAMP
	reqExt.Appnexus.HeaderBiddingSource = a.hbSource + isVIDEO

	return reqExt, nil
}

func validateAndBuildAppNexusExt(imp *openrtb2.Imp) (openrtb_ext.ExtImpAppnexus, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpAppnexus{}, err
	}

	var appnexusExt openrtb_ext.ExtImpAppnexus
	if err := json.Unmarshal(bidderExt.Bidder, &appnexusExt); err != nil {
		return openrtb_ext.ExtImpAppnexus{}, err
	}

	handleLegacyParams(&appnexusExt)

	if err := validateAppnexusExt(&appnexusExt); err != nil {
		return openrtb_ext.ExtImpAppnexus{}, err
	}

	return appnexusExt, nil
}

func handleLegacyParams(appnexusExt *openrtb_ext.ExtImpAppnexus) {
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

func splitRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, requestExtension bidReqExt, uri string) ([]*adapters.RequestData, []error) {
	var errs []error
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

func validateAppnexusExt(appnexusExt *openrtb_ext.ExtImpAppnexus) error {
	if appnexusExt.PlacementId == 0 && (appnexusExt.InvCode == "" || appnexusExt.Member == "") {
		return &errortypes.BadInput{
			Message: "No placement or member+invcode provided",
		}
	}
	return nil
}

func buildRequestImp(imp *openrtb2.Imp, appnexusExt *openrtb_ext.ExtImpAppnexus, displayManagerVer string) error {
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

		if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
			firstFormat := bannerCopy.Format[0]
			bannerCopy.W = &(firstFormat.W)
			bannerCopy.H = &(firstFormat.H)
		}
		imp.Banner = &bannerCopy
	}

	// Populate imp.displaymanagerver if the SDK failed to do it.
	if len(imp.DisplayManagerVer) == 0 && len(displayManagerVer) > 0 {
		imp.DisplayManagerVer = displayManagerVer
	}

	impExt := impExt{Appnexus: impExtAppnexus{
		PlacementID:       int(appnexusExt.PlacementId),
		TrafficSourceCode: appnexusExt.TrafficSourceCode,
		Keywords:          appnexusExt.Keywords.String(),
		UsePmtRule:        appnexusExt.UsePaymentRule,
		PrivateSizes:      appnexusExt.PrivateSizes,
		ExtInvCode:        appnexusExt.ExtInvCode,
		ExternalImpId:     appnexusExt.ExternalImpId,
	}}

	var err error
	imp.Ext, err = json.Marshal(&impExt)

	return err
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *bidExt) (openrtb_ext.BidType, error) {
	switch bid.Appnexus.BidType {
	case 0:
		return openrtb_ext.BidTypeBanner, nil
	case 1:
		return openrtb_ext.BidTypeVideo, nil
	case 3:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unrecognized bid_ad_type in response from appnexus: %d", bid.Appnexus.BidType)
	}
}

// getIabCategoryForBid maps an appnexus brand id to an IAB category.
func (a *adapter) findIabCategoryForBid(bid *bidExt) (string, bool) {
	brandIDString := strconv.Itoa(bid.Appnexus.BrandCategory)
	iabCategory, ok := iabCategoryMap[brandIDString]
	return iabCategory, ok
}

func appendMemberId(uri url.URL, memberId string) url.URL {
	q := uri.Query()
	q.Set("member_id", memberId)
	uri.RawQuery = q.Encode()
	return uri
}

func buildDisplayManageVer(req *openrtb2.BidRequest) string {
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

func (a *adapter) buildAdPodRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, requestExtension bidReqExt, uri string) ([]*adapters.RequestData, []error) {
	var errs []error
	podImps := groupByPods(imps)
	requests := make([]*adapters.RequestData, 0, len(podImps))
	for _, podImps := range podImps {
		requestExtension.Appnexus.AdPodId = fmt.Sprint(a.randomGenerator.GenerateInt63())

		reqs, errors := splitRequests(podImps, request, requestExtension, uri)
		requests = append(requests, reqs...)
		errs = append(errs, errors...)
	}

	return requests, errs
}
