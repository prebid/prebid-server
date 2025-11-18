package msft

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	defaultHBSource      = 5
	defaultHBSourceVideo = 6
	maxImpsPerReq        = 10
)

var (
	errMalformedExtraInfo  = errors.New("malformed extra adapter info")
	errMalformedRequestExt = errors.New("malformed request ext.appnexus")
	errMemberIDMismatch    = errors.New("member id mismatch: all impressions must use the same member id")
)

type adapter struct {
	uri           url.URL
	hbSource      int
	hbSourceVideo int
}

// Builder builds a new instance of the Microsoft adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	uri, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}

	extraInfo, err := parseExtraInfo(config.ExtraAdapterInfo)
	if err != nil {
		return nil, err
	}

	bidder := &adapter{
		uri:           *uri,
		hbSource:      extraInfo.HBSource,
		hbSourceVideo: extraInfo.HBSourceVideo,
	}
	return bidder, nil
}

func parseExtraInfo(v string) (extraAdapterInfo, error) {
	if len(v) == 0 {
		return buildDefaultExtraInfo(), nil
	}

	var info extraAdapterInfo
	if err := jsonutil.Unmarshal([]byte(v), &info); err != nil {
		return info, errMalformedExtraInfo
	}

	if info.HBSource == 0 {
		info.HBSource = defaultHBSource
	}

	if info.HBSourceVideo == 0 {
		info.HBSourceVideo = defaultHBSourceVideo
	}

	return info, nil
}

func buildDefaultExtraInfo() extraAdapterInfo {
	return extraAdapterInfo{
		HBSource:      defaultHBSource,
		HBSourceVideo: defaultHBSourceVideo,
	}
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	displayManagerVerBuilder := initDisplayManageVerBuilder(request)

	var (
		uniqueMemberID int
		errs           []error
	)

	validImps := []openrtb2.Imp{}
	for i := 0; i < len(request.Imp); i++ {
		var impExt impExtIncoming
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("malformed impression ext for id '%s'", request.Imp[i].ID))
			continue
		}

		if err := modifyImp(&request.Imp[i], impExt, displayManagerVerBuilder); err != nil {
			errs = append(errs, fmt.Errorf("error building impression ext for id '%s'", request.Imp[i].ID))
			continue
		}

		// ensure all impressions with member ids use the same member id
		memberId := impExt.Bidder.Member
		if memberId != 0 {
			if uniqueMemberID == 0 {
				uniqueMemberID = memberId
			} else if uniqueMemberID != memberId {
				errs = append(errs, errMemberIDMismatch)
				return nil, errs
			}
		}

		validImps = append(validImps, request.Imp[i])
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	requestURI := a.uri
	if uniqueMemberID != 0 {
		requestURI = appendMemberId(requestURI, uniqueMemberID)
	}

	if err := a.modifyRequestExt(request, requestInfo); err != nil {
		return nil, append(errs, err)
	}

	requests, errors := buildRequests(request.Imp, request, requestURI.String())
	return requests, append(errs, errors...)
}

func initDisplayManageVerBuilder(req *openrtb2.BidRequest) func() string {
	if req.App == nil || len(req.App.Ext) == 0 {
		return func() string { return "" }
	}

	var (
		appExt json.RawMessage = req.App.Ext
		result string
		hasRun bool
	)
	return func() string {
		if !hasRun {
			source, err := jsonparser.GetString(appExt, openrtb_ext.PrebidExtKey, "source")
			if err != nil {
				hasRun = true
				return result
			}

			version, err := jsonparser.GetString(appExt, openrtb_ext.PrebidExtKey, "version")
			if err != nil {
				hasRun = true
				return result
			}

			result = fmt.Sprintf("%s-%s", source, version)
			hasRun = true
		}
		return result
	}
}

func modifyImp(imp *openrtb2.Imp, ext impExtIncoming, displayManagerVerBuilder func() string) error {
	if ext.Bidder.InvCode != "" {
		imp.TagID = ext.Bidder.InvCode
	}

	if imp.Banner != nil {
		bannerCopy := *imp.Banner

		if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
			firstFormat := bannerCopy.Format[0]
			bannerCopy.W = &(firstFormat.W)
			bannerCopy.H = &(firstFormat.H)
		}

		if bannerCopy.API == nil {
			bannerCopy.API = ext.Bidder.BannerFrameworks
		}

		imp.Banner = &bannerCopy
	}

	if len(imp.DisplayManagerVer) == 0 {
		imp.DisplayManagerVer = displayManagerVerBuilder()
	}

	impExt := impExtOutgoing{
		Appnexus: impExtOutgoingAppnexus{
			PlacementID:       ext.Bidder.PlacementId,
			AllowSmallerSizes: ext.Bidder.AllowSmallerSizes,
			UsePmtRule:        ext.Bidder.UsePaymentRule,
			Keywords:          ext.Bidder.Keywords,
			TrafficSourceCode: ext.Bidder.TrafficSourceCode,
			PubClick:          ext.Bidder.PubClick,
			ExtInvCode:        ext.Bidder.ExtInvCode,
			ExtImpID:          ext.Bidder.ExtImpId,
		},
		GPID: ext.GPID,
	}

	var err error
	imp.Ext, err = jsonutil.Marshal(impExt)

	return err
}

func appendMemberId(uri url.URL, memberId int) url.URL {
	q := uri.Query()
	q.Set("member_id", fmt.Sprint(memberId))
	uri.RawQuery = q.Encode()
	return uri
}

func (a *adapter) modifyRequestExt(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) error {
	reqExt, err := getRequestExt(request.Ext)
	if err != nil {
		return err
	}

	reqExtAppnexus, err := a.getAppnexusExt(reqExt, requestInfo.PbsEntryPoint)
	if err != nil {
		return err
	}

	appnexusExtJson, err := jsonutil.Marshal(reqExtAppnexus)
	if err != nil {
		return err
	}

	reqExt["appnexus"] = appnexusExtJson

	request.Ext, err = jsonutil.Marshal(reqExt)
	return err
}

func getRequestExt(ext json.RawMessage) (map[string]json.RawMessage, error) {
	extMap := make(map[string]json.RawMessage)

	if len(ext) > 0 {
		if err := jsonutil.Unmarshal(ext, &extMap); err != nil {
			return nil, err
		}
	}

	return extMap, nil
}

func (a *adapter) getAppnexusExt(extMap map[string]json.RawMessage, reqType metrics.RequestType) (requestExAppnexus, error) {
	var appnexusExt requestExAppnexus

	if appnexusExtJson, exists := extMap["appnexus"]; exists && len(appnexusExtJson) > 0 {
		if err := jsonutil.Unmarshal(appnexusExtJson, &appnexusExt); err != nil {
			return appnexusExt, errMalformedRequestExt
		}
	}

	if prebidJson, exists := extMap["prebid"]; exists {
		_, valueType, _, err := jsonparser.Get(prebidJson, "targeting", "includebrandcategory")
		if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return appnexusExt, err
		}

		if valueType == jsonparser.Object {
			appnexusExt.BrandCategoryUniqueness = ptrutil.ToPtr(true)
			appnexusExt.IncludeBrandCategory = ptrutil.ToPtr(true)
		}
	}

	if reqType == metrics.ReqTypeAMP {
		appnexusExt.IsAMP = 1
	}

	if reqType == metrics.ReqTypeVideo {
		appnexusExt.HeaderBiddingSource = a.hbSourceVideo
	} else {
		appnexusExt.HeaderBiddingSource = a.hbSource
	}

	return appnexusExt, nil
}

func buildRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, uri string) ([]*adapters.RequestData, []error) {
	var (
		requestsCount = (len(imps) + maxImpsPerReq - 1) / maxImpsPerReq
		requests      = make([]*adapters.RequestData, 0, requestsCount)
	)

	for i := 0; i < requestsCount; i++ {
		imps := imps[i*maxImpsPerReq : min((i+1)*maxImpsPerReq, len(imps))]

		request.Imp = imps
		requestJSON, err := jsonutil.Marshal(request)
		if err != nil {
			return nil, []error{err}
		}

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    requestJSON,
			Headers: buildHeaders(),
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	return requests, nil
}

func buildHeaders() http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")
	headers.Set("x-openrtb-version", "2.6")
	return headers
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var appnexusResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &appnexusResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, sb := range appnexusResponse.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]

			var bidExt bidExt
			if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, err)
				continue
			}

			bidType, err := getMediaTypeForBid(&bidExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			iabCategory, found := findIABCategoryForBid(&bidExt)
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

func getMediaTypeForBid(bid *bidExt) (openrtb_ext.BidType, error) {
	switch bid.Appnexus.BidType {
	case 0:
		return openrtb_ext.BidTypeBanner, nil
	case 1:
		return openrtb_ext.BidTypeVideo, nil
	case 3:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unrecognized bid_ad_type in response: %d", bid.Appnexus.BidType)
	}
}
