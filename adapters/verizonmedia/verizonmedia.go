package verizonmedia

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type VerizonMediaAdapter struct {
	URI string
}

func (a *VerizonMediaAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errors := make([]error, 0, 1)

	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No impression in the bid request",
		}
		errors = append(errors, err)
		return nil, errors
	}

	reqs := make([]*adapters.RequestData, 0, len(request.Imp))
	headers := http.Header{}

	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	if request.Device != nil && request.Device.UA != "" {
		headers.Set("User-Agent", request.Device.UA)
	}

	for idx, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		err := json.Unmarshal(imp.Ext, &bidderExt)
		if err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: ext.bidder not provided", idx),
			}
			errors = append(errors, err)
			continue
		}

		var verizonMediaExt openrtb_ext.ExtImpVerizonMedia
		err = json.Unmarshal(bidderExt.Bidder, &verizonMediaExt)
		if err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: %s", idx, err.Error()),
			}
			errors = append(errors, err)
			continue
		}

		if verizonMediaExt.Dcn == "" {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: missing param dcn", idx),
			}
			errors = append(errors, err)
			continue
		}

		if verizonMediaExt.Pos == "" {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: missing param pos", idx),
			}
			errors = append(errors, err)
			continue
		}

		// Split up multi-impression requests into multiple requests so that
		// each split request is only associated to a single impression
		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}

		if request.Site != nil {
			siteCopy := *request.Site
			reqCopy.Site = &siteCopy
		} else if request.App != nil {
			appCopy := *request.App
			reqCopy.App = &appCopy
		}

		if err := changeRequestForBidService(&reqCopy, &verizonMediaExt); err != nil {
			errors = append(errors, err)
			continue
		}

		reqJSON, err := json.Marshal(&reqCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.URI,
			Body:    reqJSON,
			Headers: headers,
		})
	}

	return reqs, errors
}

func (a *VerizonMediaAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d.", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			exists, mediaTypeId := getImpInfo(bid.ImpID, internalRequest.Imp)
			if !exists {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}}
			}

			if openrtb_ext.BidTypeBanner != mediaTypeId {
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

func getImpInfo(impId string, imps []openrtb2.Imp) (bool, openrtb_ext.BidType) {
	var mediaType openrtb_ext.BidType
	var exists bool
	for _, imp := range imps {
		if imp.ID == impId {
			exists = true
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			}
			break
		}
	}
	return exists, mediaType
}

func changeRequestForBidService(request *openrtb2.BidRequest, extension *openrtb_ext.ExtImpVerizonMedia) error {
	/* Always override the tag ID and (site ID or app ID) of the request */
	request.Imp[0].TagID = extension.Pos
	if request.Site != nil {
		request.Site.ID = extension.Dcn
	} else if request.App != nil {
		request.App.ID = extension.Dcn
	}

	if request.Imp[0].Banner == nil {
		return nil
	}

	banner := *request.Imp[0].Banner
	request.Imp[0].Banner = &banner

	if banner.W != nil && banner.H != nil {
		if *banner.W == 0 || *banner.H == 0 {
			return errors.New(fmt.Sprintf("Invalid sizes provided for Banner %dx%d", *banner.W, *banner.H))
		}
		return nil
	}

	if len(banner.Format) == 0 {
		return errors.New(fmt.Sprintf("No sizes provided for Banner %v", banner.Format))
	}

	banner.W = openrtb2.Int64Ptr(banner.Format[0].W)
	banner.H = openrtb2.Int64Ptr(banner.Format[0].H)

	return nil
}

// Builder builds a new instance of the VerizonMedia adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &VerizonMediaAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
