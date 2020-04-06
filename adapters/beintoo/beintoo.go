package beintoo

import (

	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/golang/glog"
	"strconv"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/errortypes"
	"net/http"
	"fmt"
	"github.com/prebid/prebid-server/adapters"
)

type BeintooAdapter struct {
	endpoint string
}



func (a *BeintooAdapter) MakeRequests(Request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {



	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No impressions in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}

	// As of now, Beintoo supports only banner and video impressions

	validImpExists := false
	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			bannerCopy := *request.Imp[i].Banner
			if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
				firstFormat := bannerCopy.Format[0]
				bannerCopy.W = &(firstFormat.W)
				bannerCopy.H = &(firstFormat.H)
			}
			request.Imp[i].Banner = &bannerCopy
			validImpExists = true
		} else if request.Imp[i].Video != nil {
			validImpExists = true
		} else {
			err := &errortypes.BadInput{
				Message: fmt.Sprintf("Beintoo only supports banner and video media types. Ignoring imp id=%s", request.Imp[i].ID),
			}
			glog.Warning("Beintoo SUPPORT VIOLATION: only banner and video media types supported")
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	if !validImpExists {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("No valid impression in the bid request"),
		}
		errs = append(errs, err)
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	errors := make([]error, 0, 1)

	var bidderExt adapters.ExtImpBidder
	err = json.Unmarshal(request.Imp[0].Ext, &bidderExt)

	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	var beintooExt openrtb_ext.ExtImpBeintoo
	err = json.Unmarshal(bidderExt.Bidder, &BeintooExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder.supplyPartnerId not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}

	if beintooExt.SupplyPartnerId == "" {
		err = &errortypes.BadInput{
			Message: "supplyPartnerId is empty",
		}
		errors = append(errors, err)
		return nil, errors
	}



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
 		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}


func (a *BeintooAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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
			Message: fmt.Sprintf("bad server response: %v. ", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	sb := bidResp.SeatBid[0]
	for i := 0; i < len(sb.Bid); i++ {
		bid := sb.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: getMediaType(bid.ImpID, internalRequest.Imp),
		})
	}
	return bidResponse, nil
}

// Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func getMediaType(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}


func NewBeintooBidder(endpoint string) *BeintooAdapter {
	return &BeintooAdapter{
		endpoint: endpoint,
	}
}
