package onemobile

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type OneMobileAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

func (a *OneMobileAdapter) Name() string {
	return "onemobile"
}

func (a *OneMobileAdapter) SkipNoCookies() bool {
	return false
}

func (a *OneMobileAdapter) MakeRequests(requestIn *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	request := *requestIn
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No impression in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}

	errors := make([]error, 0, 1)

	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(request.Imp[0].Ext, &bidderExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	var oneMobileExt openrtb_ext.ExtImpOneMobile
	err = json.Unmarshal(bidderExt.Bidder, &oneMobileExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		errors = append(errors, err)
		return nil, errors
	}

	if oneMobileExt.Dcn == "" {
		err = &errortypes.BadInput{
			Message: "Missing param dcn",
		}
		errors = append(errors, err)
		return nil, errors
	}

	if oneMobileExt.Pos == "" {
		err = &errortypes.BadInput{
			Message: "Missing param pos",
		}
		errors = append(errors, err)
		return nil, errors
	}

	requestImp := make([]openrtb.Imp, len(requestIn.Imp))
	copy(requestImp, requestIn.Imp)
	request.Imp = requestImp
	siteCopy := *request.Site
	request.Site = &siteCopy
	changeRequestForBidService(&request, &oneMobileExt)
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	thisURI := a.URI

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Set("User-Agent", request.Device.UA)
	headers.Add("x-openrtb-version", "2.5")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

func (a *OneMobileAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. ", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d. ", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			impID := lookupImpID(bid.ImpID, internalRequest.Imp)
			if impID == "" {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
			})
		}
	}

	return bidResponse, nil
}

func changeRequestForBidService(request *openrtb.BidRequest, extension *openrtb_ext.ExtImpOneMobile) {
	if request.Imp[0].TagID == "" {
		request.Imp[0].TagID = extension.Pos
	}
	if request.Site.ID == "" {
		request.Site.ID = extension.Dcn
	}
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
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

func lookupImpID(impId string, imps []openrtb.Imp) string {
	for _, imp := range imps {
		if imp.ID == impId {
			return imp.ID
		}
	}
	return ""
}

func NewOneMobileAdapter(config *adapters.HTTPAdapterConfig, uri string) *OneMobileAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &OneMobileAdapter{
		http: a,
		URI:  uri,
	}
}

func NewOneMobileBidder(client *http.Client, endpoint string) *OneMobileAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &OneMobileAdapter{
		http: a,
		URI:  endpoint,
	}
}
