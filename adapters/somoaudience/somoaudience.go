package somoaudience

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
)

const hbconfig = "hb_pbs_1.0.0"

type SomoaudienceAdapter struct {
	endpoint string
}

type somoaudienceReqExt struct {
	BidderConfig string `json:"prebid"`
}

func (a *SomoaudienceAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var bannerImps []openrtb.Imp
	var videoImps []openrtb.Imp
	var nativeImps []openrtb.Imp

	for _, imp := range request.Imp {
		if imp.Banner != nil {
			bannerImps = append(bannerImps, imp)
		} else if imp.Video != nil {
			videoImps = append(videoImps, imp)
		} else if imp.Native != nil {
			nativeImps = append(nativeImps, imp)
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

	// Somoaudience only supports single imp video request
	for _, videoImp := range videoImps {
		reqCopy.Imp = []openrtb.Imp{videoImp}
		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}

	// Somoaudience only supports single imp video request
	for _, nativeImp := range nativeImps {
		reqCopy.Imp = []openrtb.Imp{nativeImp}
		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}
	return adapterRequests, errs

}

func (a *SomoaudienceAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var err error
	var validImps []openrtb.Imp
	reqExt := somoaudienceReqExt{BidderConfig: hbconfig}

	var placementHash string

	for _, imp := range request.Imp {
		placementHash, err = preprocess(&imp, &reqExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		imp.Ext = nil
		validImps = append(validImps, imp)
	}

	// If all the imps were malformed, don't bother making a server call with no impressions.
	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

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
	headers.Add("x-openrtb-version", "2.5")

	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint + fmt.Sprintf("?s=%s", placementHash),
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

func preprocess(imp *openrtb.Imp, reqExt *somoaudienceReqExt) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: "ignoring imp id=empty-extbid-test, extImpBidder is empty",
		}
	}

	var somoExt openrtb_ext.ExtImpSomoaudience
	if err := json.Unmarshal(bidderExt.Bidder, &somoExt); err != nil {
		return "", &errortypes.BadInput{
			Message: "ignoring imp id=empty-extbid-test, error while decoding impExt, err: " + err.Error(),
		}
	}

	imp.BidFloor = somoExt.BidFloor
	imp.Ext = nil

	return somoExt.PlacementHash, nil
}

func (a *SomoaudienceAdapter) MakeBids(bidReq *openrtb.BidRequest, unused *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, bidReq.Imp),
			})
		}
	}

	return bidResponse, nil
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			if imp.Banner != nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeBanner
			}
			return mediaType
		}
	}
	return mediaType
}

//Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func NewSomoaudienceBidder(endpoint string) *SomoaudienceAdapter {
	return &SomoaudienceAdapter{
		endpoint: endpoint,
	}
}
