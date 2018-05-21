package oath

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"

	"github.com/prebid/prebid-server/openrtb_ext"
	"strconv"
)

const uri = "http://east-bid.ybp.yahoo.com/bid/appnexuspbs"

type OathAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

func (a *OathAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	if len(request.Imp) == 0 {
		return nil, errs
	}

	var bannerImps []openrtb.Imp
	var videoImps []openrtb.Imp

	for _, imp := range request.Imp {

		if imp.Banner != nil {
			bannerImps = append(bannerImps, imp)
		} else if imp.Video != nil {
			videoImps = append(videoImps, imp)
		} else {
			err := &adapters.BadInputError{
				Message: fmt.Sprintf("Oath only supports banner and video imps. Ignoring imp id=%s", imp.ID),
			}
			errs = append(errs, err)
		}
	}

	var validImpExists bool
	validImpExists = false
	if len(bannerImps) <= 0 && len(videoImps) <= 0 {
		err := &adapters.BadInputError{
			Message: fmt.Sprintf("No valid impression in the bid request"),
		}
		errs = append(errs, err)
		return nil, errs
	} else {
		validImpExists = true
	}

	reqJSON, err := json.Marshal(request)
	if err != nil && !validImpExists {
		errs = append(errs, err)
		return nil, errs
	}
	errors := make([]error, 0, 1)

	var bidderExt adapters.ExtImpBidder
	err = json.Unmarshal(request.Imp[0].Ext, &bidderExt)

	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	var oathExt openrtb_ext.ExtImpOath
	err = json.Unmarshal(bidderExt.Bidder, &oathExt)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	pubName := oathExt.PublisherName
	thisURI := uri
	thisURI = thisURI + "?publisher=" + pubName
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

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func (a *OathAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&adapters.BadInputError{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner //default type
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}

//func NewOathAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *OathAdapter {
//	return NewOathBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
//}

func NewOathBidder(client *http.Client, endpoint string) *OathAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &OathAdapter{
		http: a,
		URI:  endpoint,
	}
}
