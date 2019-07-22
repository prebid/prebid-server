package verizonmedia

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type VerizonMediaAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

func (a *VerizonMediaAdapter) Name() string {
	return "verizonmedia"
}

func (a *VerizonMediaAdapter) SkipNoCookies() bool {
	return false
}

func (a *VerizonMediaAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

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
	var verizonMediaExt openrtb_ext.ExtImpVerizonMedia
	err = json.Unmarshal(bidderExt.Bidder, &verizonMediaExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		errors = append(errors, err)
		return nil, errors
	}

	if verizonMediaExt.Dcn == "" {
		err = &errortypes.BadInput{
			Message: "Missing param dcn",
		}
		errors = append(errors, err)
		return nil, errors
	}

	if verizonMediaExt.Pos == "" {
		err = &errortypes.BadInput{
			Message: "Missing param pos",
		}
		errors = append(errors, err)
		return nil, errors
	}

	siteCopy := *request.Site
	request.Site = &siteCopy
	changeRequestForBidService(request, &verizonMediaExt)
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

func (a *VerizonMediaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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
			if !impIDExists(bid.ImpID, internalRequest.Imp) {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}}
			}

			if openrtb_ext.BidTypeBanner != getMediaTypeForImp(bid.ImpID, internalRequest.Imp) {
				//only banner is supported, anything else is ignored
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	var mediaType openrtb_ext.BidType
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			}
			return mediaType
		}
	}
	return mediaType
}

func changeRequestForBidService(request *openrtb.BidRequest, extension *openrtb_ext.ExtImpVerizonMedia) {
	if request.Imp[0].TagID == "" {
		request.Imp[0].TagID = extension.Pos
	}
	if request.Site.ID == "" {
		request.Site.ID = extension.Dcn
	}
}

func impIDExists(impId string, imps []openrtb.Imp) bool {
	for _, imp := range imps {
		if imp.ID == impId {
			return true
		}
	}
	return false
}

func NewVerizonMediaAdapter(config *adapters.HTTPAdapterConfig, uri string) *VerizonMediaAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &VerizonMediaAdapter{
		http: a,
		URI:  uri,
	}
}

func NewVerizonMediaBidder(client *http.Client, endpoint string) *VerizonMediaAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &VerizonMediaAdapter{
		http: a,
		URI:  endpoint,
	}
}
