package somoaudience

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const hbconfig = "hb_pbs_1.0.0"

type SomoaudienceAdapter struct {
	endpoint string
}

type somoaudienceReqExt struct {
	BidderConfig string `json:"prebid"`
}

func (a *SomoaudienceAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var bannerImps []openrtb2.Imp
	var videoImps []openrtb2.Imp
	var nativeImps []openrtb2.Imp

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
		reqCopy.Imp = []openrtb2.Imp{videoImp}
		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}

	// Somoaudience only supports single imp video request
	for _, nativeImp := range nativeImps {
		reqCopy.Imp = []openrtb2.Imp{nativeImp}
		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}
	return adapterRequests, errs

}

func (a *SomoaudienceAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var err error
	var validImps []openrtb2.Imp
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

func preprocess(imp *openrtb2.Imp, reqExt *somoaudienceReqExt) (string, error) {
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

func (a *SomoaudienceAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
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

// Builder builds a new instance of the Somoaudience adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &SomoaudienceAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
