package adprime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// AdprimeAdapter struct
type AdprimeAdapter struct {
	URI string
}

// Builder builds a new instance of the Adprime adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &AdprimeAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests create bid request for adprime demand
func (a *AdprimeAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var err error

	var adapterRequests []*adapters.RequestData

	var bidderExt adapters.ExtImpBidder
	var adprimeExt openrtb_ext.ExtImpAdprime

	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb2.Imp{imp}

		err = json.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		err = json.Unmarshal(bidderExt.Bidder, &adprimeExt)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		// tagId
		tagID := adprimeExt.TagID
		reqCopy.Imp[0].TagID = tagID

		// placementId
		newExt, err := json.Marshal(
			map[string]interface{}{
				"bidder": map[string]interface{}{
					"TagID":       tagID,
					"placementId": tagID,
				},
			})
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		reqCopy.Imp[0].Ext = newExt

		// keywords
		if reqCopy.Site != nil && len(adprimeExt.Keywords) > 0 {
			siteCopy := *reqCopy.Site
			siteCopy.Keywords = strings.Join(adprimeExt.Keywords, ",")
			reqCopy.Site = &siteCopy
		}

		// audiences
		if reqCopy.Site != nil && len(adprimeExt.Audiences) > 0 {
			if reqCopy.User == nil {
				reqCopy.User = &openrtb2.User{}
			}
			userCopy := *reqCopy.User
			userCopy.CustomData = strings.Join(adprimeExt.Audiences, ",")
			reqCopy.User = &userCopy
		}

		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}
	return adapterRequests, errs
}

func (a *AdprimeAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {

	var errs []error

	reqJSON, err := json.Marshal(request)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// MakeBids makes the bids
func (a *AdprimeAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Page not found: %d. Run with request.debug = 1 for more info", response.StatusCode),
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", impID),
	}
}
