package marsmedia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type MarsmediaAdapter struct {
	URI string
}

func (a *MarsmediaAdapter) MakeRequests(requestIn *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	request := *requestIn

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "ext.bidder not provided",
		}}
	}

	var marsmediaExt openrtb_ext.ExtImpMarsmedia
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &marsmediaExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "ext.bidder.zoneId not provided",
		}}
	}

	if marsmediaExt.ZoneID == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "zoneId is empty",
		}}
	}

	validImpExists := false
	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			bannerCopy := *requestIn.Imp[i].Banner
			if len(bannerCopy.Format) > 0 {
				firstFormat := bannerCopy.Format[0]
				bannerCopy.W = &(firstFormat.W)
				bannerCopy.H = &(firstFormat.H)
				request.Imp[i].Banner = &bannerCopy
				validImpExists = true
			} else if bannerCopy.W != nil && bannerCopy.H != nil {
				validImpExists = true
			} else {
				return nil, []error{&errortypes.BadInput{
					Message: "No valid banner format in the bid request",
				}}
			}
		} else if request.Imp[i].Video != nil {
			validImpExists = true
		}
	}
	if !validImpExists {
		return nil, []error{&errortypes.BadInput{
			Message: "No valid impression in the bid request",
		}}
	}

	request.AT = 1 //Defaulting to first price auction for all prebid requests

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Json not encoded. err: %s", err),
		}}
	}

	uri := a.URI + "&zone=" + marsmediaExt.ZoneID.String()
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
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, []error{}
}

func (a *MarsmediaAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d. ", err),
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

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
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

// Builder builds a new instance of the Marsmedia adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &MarsmediaAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
