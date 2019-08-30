package gamma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type GammaAdapter struct {
	URI string
}

func checkParams(gammaExt openrtb_ext.ExtImpGamma) error {
	if gammaExt.PartnerID == "" {
		return &errortypes.BadInput{
			Message: "PartnerID is empty",
		}
	}
	if gammaExt.ZoneID == "" {
		return &errortypes.BadInput{
			Message: "ZoneID is empty",
		}
	}
	if gammaExt.WebID == "" {
		return &errortypes.BadInput{
			Message: "WebID is empty",
		}
	}
	return nil
}
func (a *GammaAdapter) makeRequest(request *openrtb.BidRequest, imp openrtb.Imp) (*adapters.RequestData, []error) {
	var errors []error

	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(imp.Ext, &bidderExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	var gammaExt openrtb_ext.ExtImpGamma
	err = json.Unmarshal(bidderExt.Bidder, &gammaExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder.publisher not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	err = checkParams(gammaExt)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	thisURI := a.URI
	thisURI = thisURI + "?id=" + gammaExt.PartnerID
	thisURI = thisURI + "&zid=" + gammaExt.ZoneID
	thisURI = thisURI + "&wid=" + gammaExt.WebID
	thisURI = thisURI + "&bidid=" + imp.ID
	thisURI = thisURI + "&hb=pbmobile"
	if request.Device != nil {
		if request.Device.IP != "" {
			thisURI = thisURI + "&device_ip=" + request.Device.IP
		}
		if request.Device.Model != "" {
			thisURI = thisURI + "&device_model=" + request.Device.Model
		}
		if request.Device.OS != "" {
			thisURI = thisURI + "&device_os=" + request.Device.OS
		}
		if request.Device.UA != "" {
			thisURI = thisURI + "&device_ua=" + url.QueryEscape(request.Device.UA)
		}
		if request.Device.IFA != "" {
			thisURI = thisURI + "&device_ifa=" + request.Device.IFA
		}
	}
	if request.App != nil {
		if request.App.ID != "" {
			thisURI = thisURI + "&app_id=" + request.App.ID
		}
		if request.App.Bundle != "" {
			thisURI = thisURI + "&app_bundle=" + request.App.Bundle
		}
		if request.App.Name != "" {
			thisURI = thisURI + "&app_name=" + request.App.Name
		}
	}
	headers := http.Header{}
	headers.Add("Accept", "*/*")
	headers.Add("x-openrtb-version", "2.5")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}
	headers.Add("Connection", "keep-alive")
	headers.Add("cache-control", "no-cache")
	headers.Add("Accept-Encoding", "gzip, deflate")

	return &adapters.RequestData{
		Method:  "GET",
		Uri:     thisURI,
		Headers: headers,
	}, errors
}
func (a *GammaAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No impressions in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}
	var invalidImpIndex = make([]int, 0, 0)

	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			bannerCopy := *request.Imp[i].Banner
			if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
				firstFormat := bannerCopy.Format[0]
				bannerCopy.W = &(firstFormat.W)
				bannerCopy.H = &(firstFormat.H)
			}
			request.Imp[i].Banner = &bannerCopy
		} else if request.Imp[i].Video == nil {
			err := &errortypes.BadInput{
				Message: fmt.Sprintf("Gamma only supports banner and video media types. Ignoring imp id=%s", request.Imp[i].ID),
			}
			errs = append(errs, err)
			invalidImpIndex = append(invalidImpIndex, i)
		}
	}

	var adapterRequests []*adapters.RequestData
	if len(invalidImpIndex) == 0 {
		for _, imp := range request.Imp {
			adapterReq, errors := a.makeRequest(request, imp)
			if adapterReq != nil {
				adapterRequests = append(adapterRequests, adapterReq)
			}
			errs = append(errs, errors...)
		}
	} else if len(request.Imp) == len(invalidImpIndex) {
		//only true if every Imp was not a Banner or a Video
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("No valid impression in the bid request"),
		}
		errs = append(errs, err)
		return nil, errs
	} else {
		var j int = 0
		for i := 0; i < len(request.Imp); i++ {
			if j < len(invalidImpIndex) && i == invalidImpIndex[j] {
				j++
			} else {
				adapterReq, errors := a.makeRequest(request, request.Imp[i])
				if adapterReq != nil {
					adapterRequests = append(adapterRequests, adapterReq)
				}
				errs = append(errs, errors...)
			}
		}
	}

	return adapterRequests, errs
}

func (a *GammaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadServerResponse{
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bidResp.ID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

//Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner //default type
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

func NewGammaBidder(endpoint string) *GammaAdapter {
	return &GammaAdapter{
		URI: endpoint,
	}
}
