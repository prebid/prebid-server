package openx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const hbconfig = "hb_pbs_1.0.0"

type OpenxAdapter struct {
	bidderName string
	endpoint   string
}

type openxImpExt map[string]json.RawMessage

type openxReqExt struct {
	DelDomain    string `json:"delDomain,omitempty"`
	Platform     string `json:"platform,omitempty"`
	BidderConfig string `json:"bc"`
}

type openxRespExt struct {
	FledgeAuctionConfigs map[string]json.RawMessage `json:"fledge_auction_configs,omitempty"`
}

func (a *OpenxAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var bannerAndNativeImps []openrtb2.Imp
	var videoImps []openrtb2.Imp

	for _, imp := range request.Imp {
		// OpenX doesn't allow multi-type imp. Banner takes priority over video and video takes priority over native
		// Openx also wants to send banner and native imps in one request
		if imp.Banner != nil {
			bannerAndNativeImps = append(bannerAndNativeImps, imp)
		} else if imp.Video != nil {
			videoImps = append(videoImps, imp)
		} else if imp.Native != nil {
			bannerAndNativeImps = append(bannerAndNativeImps, imp)
		}
	}

	var adapterRequests []*adapters.RequestData
	// Make a copy as we don't want to change the original request
	reqCopy := *request

	reqCopy.Imp = bannerAndNativeImps
	adapterReq, errors := a.makeRequest(&reqCopy)
	if adapterReq != nil {
		adapterRequests = append(adapterRequests, adapterReq)
	}
	errs = append(errs, errors...)

	// OpenX only supports single imp video request
	for _, videoImp := range videoImps {
		reqCopy.Imp = []openrtb2.Imp{videoImp}
		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}

	return adapterRequests, errs
}

func (a *OpenxAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb2.Imp
	reqExt := openxReqExt{BidderConfig: hbconfig}

	for _, imp := range request.Imp {
		if err := preprocess(&imp, &reqExt); err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
	}

	// If all the imps were malformed, don't bother making a server call with no impressions.
	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	var err error
	request.Ext, err = json.Marshal(reqExt)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

// Mutate the imp to get it ready to send to openx.
func preprocess(imp *openrtb2.Imp, reqExt *openxReqExt) error {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var openxExt openrtb_ext.ExtImpOpenx
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &openxExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	reqExt.DelDomain = openxExt.DelDomain
	reqExt.Platform = openxExt.Platform

	imp.TagID = openxExt.Unit.String()
	if imp.BidFloor == 0 {
		customFloor, err := openxExt.CustomFloor.Float64()
		if err == nil && customFloor > 0 {
			imp.BidFloor = customFloor
		}
	}

	// outgoing imp.ext should be same as incoming imp.ext minus prebid and bidder
	impExt := openxImpExt{}
	if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	delete(impExt, openrtb_ext.PrebidExtKey)
	delete(impExt, openrtb_ext.PrebidExtBidderKey)

	if openxExt.CustomParams != nil {
		var err error
		if impExt["customParams"], err = json.Marshal(openxExt.CustomParams); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	if len(impExt) > 0 {
		var err error
		if imp.Ext, err = json.Marshal(impExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	} else {
		imp.Ext = nil
	}

	if imp.Video != nil {
		videoCopy := *imp.Video
		if imp.Rwdd == 1 {
			videoCopy.Ext = json.RawMessage(`{"rewarded":1}`)
		} else {
			videoCopy.Ext = nil
		}
		imp.Video = &videoCopy
	}

	return nil
}

func (a *OpenxAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	// overrride default currency
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	if bidResp.Ext != nil {
		var bidRespExt openxRespExt
		if err := jsonutil.Unmarshal(bidResp.Ext, &bidRespExt); err == nil && bidRespExt.FledgeAuctionConfigs != nil {
			bidResponse.FledgeAuctionConfigs = make([]*openrtb_ext.FledgeAuctionConfig, 0, len(bidRespExt.FledgeAuctionConfigs))
			for impId, config := range bidRespExt.FledgeAuctionConfigs {
				fledgeAuctionConfig := &openrtb_ext.FledgeAuctionConfig{
					ImpId:  impId,
					Bidder: a.bidderName,
					Config: config,
				}
				bidResponse.FledgeAuctionConfigs = append(bidResponse.FledgeAuctionConfigs, fledgeAuctionConfig)
			}
		}
	}

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &sb.Bid[i],
				BidType:  getBidType(sb.Bid[i].MType, sb.Bid[i].ImpID, internalRequest.Imp),
				BidVideo: getBidVideo(&sb.Bid[i]),
			})
		}
	}
	return bidResponse, nil
}

func getBidVideo(bid *openrtb2.Bid) *openrtb_ext.ExtBidPrebidVideo {
	var primaryCategory string
	if len(bid.Cat) > 0 {
		primaryCategory = bid.Cat[0]
	}
	return &openrtb_ext.ExtBidPrebidVideo{
		PrimaryCategory: primaryCategory,
		Duration:        int(bid.Dur),
	}
}

func getBidType(mtype openrtb2.MarkupType, impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	switch mtype {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative
	default:
		return getMediaTypeForImp(impId, imps)
	}
}

// getMediaTypeForImp figures out which media type this bid is for.
//
// OpenX doesn't support multi-type impressions.
// If both banner and video exist, take banner as we do not want in-banner video.
// If both video and native exist and banner is nil, take video.
// If both banner and native exist, take banner.
// If all of the types (banner, video, native) exist, take banner.
func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Banner == nil && imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}

// Builder builds a new instance of the Openx adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &OpenxAdapter{
		endpoint:   config.Endpoint,
		bidderName: string(bidderName),
	}
	return bidder, nil
}
