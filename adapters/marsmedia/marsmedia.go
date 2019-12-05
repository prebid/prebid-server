package marsmedia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type MarsmediaAdapter struct {
	URI string
}

func (a *MarsmediaAdapter) MakeRequests(requestIn *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	request := *requestIn

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(request.Imp[0].Ext, &bidderExt)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "ext.bidder not provided",
		}}
	}

	var marsmediaExt openrtb_ext.ExtImpMarsmedia
	err = json.Unmarshal(bidderExt.Bidder, &marsmediaExt)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "ext.bidder.zone not provided",
		}}
	}

	if marsmediaExt.ZoneID == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "zone is empty",
		}}
	}

	validImpExists := false
	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			bannerCopy := *openrtb.BidRequest.Imp[i].Banner
			if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
				firstFormat := bannerCopy.Format[0]
				bannerCopy.W = &(firstFormat.W)
				bannerCopy.H = &(firstFormat.H)
				request.Imp[i].Banner = &bannerCopy
				validImpExists = true
			} else {
				return nil, []error{&errortypes.BadInput{
					Message: "No valid banner foramt in the bid request",
				}}
			}
		} else if request.Imp[i].Video != nil {
			validImpExists = true
			request.Imp[i].Video = request.Imp[i].Video
		}
	}
	if !validImpExists {
		return nil, []error{&errortypes.BadInput{
			Message: "No valid impression in the bid request",
		}}
	}

	if *openrtb.BidRequest.Site != nil {
		siteCopy := *openrtb.BidRequest.Site
		siteCopy.Publisher.ID = marsmediaExt.ZoneID
		request.Site = &siteCopy
	} else {
		appCopy := *openrtb.BidRequest.App
		appCopy.Publisher.ID = marsmediaExt.ZoneID
		request.App = &appCopy
	}

	request.AT = 1 //Defaulting to first price auction for all prebid requests

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "Json not encoded",
		}}
	}

	thisURI := a.URI
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

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, []error{}
}

func (a *MarsmediaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. ", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	sb := bidResp.SeatBid[0]
	for i := 0; i < len(sb.Bid); i++ {
		bid := sb.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
		})
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

func NewMarsmediaBidder(endpoint string) *MarsmediaAdapter {
	return &MarsmediaAdapter{
		URI: endpoint,
	}
}
