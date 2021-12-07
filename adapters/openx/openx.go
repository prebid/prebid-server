package openx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const hbconfig = "hb_pbs_1.0.0"

type OpenxAdapter struct {
	endpoint string
}

type openxImpExt struct {
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}

type openxReqExt struct {
	DelDomain    string `json:"delDomain,omitempty"`
	Platform     string `json:"platform,omitempty"`
	BidderConfig string `json:"bc"`
}

func (a *OpenxAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var bannerImps []openrtb2.Imp
	var videoImps []openrtb2.Imp

	for _, imp := range request.Imp {
		// OpenX doesn't allow multi-type imp. Banner takes priority over video.
		if imp.Banner != nil {
			bannerImps = append(bannerImps, imp)
		} else if imp.Video != nil {
			videoImps = append(videoImps, imp)
		}
	}

	var adapterRequests []*adapters.RequestData
	// Make a copy as we don't want to change the original request
	reqCopy := *request

	reqCopy.Imp = bannerImps
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
	}, errs
}

// Mutate the imp to get it ready to send to openx.
func preprocess(imp *openrtb2.Imp, reqExt *openxReqExt) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var openxExt openrtb_ext.ExtImpOpenx
	if err := json.Unmarshal(bidderExt.Bidder, &openxExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	reqExt.DelDomain = openxExt.DelDomain
	reqExt.Platform = openxExt.Platform

	imp.TagID = openxExt.Unit
	if imp.BidFloor == 0 && openxExt.CustomFloor > 0 {
		imp.BidFloor = openxExt.CustomFloor
	}
	imp.Ext = nil

	if openxExt.CustomParams != nil {
		impExt := openxImpExt{
			CustomParams: openxExt.CustomParams,
		}
		var err error
		if imp.Ext, err = json.Marshal(impExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	if imp.Video != nil {
		videoCopy := *imp.Video
		if bidderExt.Prebid != nil && bidderExt.Prebid.IsRewardedInventory == 1 {
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
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	// overrride default currency
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for.
//
// OpenX doesn't support multi-type impressions.
// If both banner and video exist, take banner as we do not want in-banner video.
func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

// Builder builds a new instance of the Openx adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &OpenxAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
