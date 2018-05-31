package brightroll

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
)

type BrightrollAdapter struct {
	URI string
}

func (a *BrightrollAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {

	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &adapters.BadInputError{
			Message: "No impression in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}

	validImpExists := false

	for _, imp := range request.Imp {
		//Brightroll supports only banner and video impressions as of now
		if imp.Banner != nil {
			validImpExists = true
		} else if imp.Video != nil {
			validImpExists = true
		} else {
			err := &adapters.BadInputError{
				Message: fmt.Sprintf("Brightroll only supports banner and video imps. Ignoring imp id=%s", imp.ID),
			}
			errs = append(errs, err)
		}
	}

	if !validImpExists {
		err := &adapters.BadInputError{
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
		err = &adapters.BadInputError{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	var brightrollExt openrtb_ext.ExtImpBrightroll
	err = json.Unmarshal(bidderExt.Bidder, &brightrollExt)
	if err != nil {
		err = &adapters.BadInputError{
			Message: "ext.bidder.publisherName not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}

	if brightrollExt.PublisherName == "" {
		err = &adapters.BadInputError{
			Message: "publisherName is empty",
		}
		errors = append(errors, err)
		return nil, errors
	}
	thisURI := a.URI
	thisURI = thisURI + "?publisher=" + brightrollExt.PublisherName
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(request.Device.DNT)))
	}
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

func (a *BrightrollAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&adapters.BadInputError{
			Message: fmt.Sprintf("Unexpected status code: %d. ", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&adapters.BadServerResponseError{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&adapters.BadServerResponseError{
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

func NewBrightrollBidder(endpoint string) *BrightrollAdapter {
	return &BrightrollAdapter{
		URI: endpoint,
	}
}
