package undertone

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
)

const adapterId = 4
const adapterVersion = "1.0.0"

type adapter struct {
	endpoint string
}

type BidRequestExt struct {
	Prebid *openrtb_ext.ExtRequestPrebid `json:"prebid"`
}

type bidderParams struct {
	Id      int    `json:"id"`
	Version string `json:"version"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	imps, publisherId, errs := getImpsAndPublisherId(request)
	if len(imps) == 0 {
		return nil, errs
	}

	reqCopy := *request
	reqCopy.Imp = imps

	populateSiteApp(&reqCopy, publisherId, request.Site, request.App)
	e := populateBidReqExt(&reqCopy)
	if e != nil {
		errs = append(errs, e)
	}

	requestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, append(errs, e)
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	impIdBidTypeMap := map[string]openrtb_ext.BidType{}
	for _, imp := range request.Imp {
		if imp.Banner != nil {
			impIdBidTypeMap[imp.ID] = openrtb_ext.BidTypeBanner
		} else if imp.Video != nil {
			impIdBidTypeMap[imp.ID] = openrtb_ext.BidTypeVideo
		}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, ok := impIdBidTypeMap[bid.ImpID]
			if !ok {
				continue
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
		return bidResponse, nil
	}
	return bidResponse, nil
}

func populateBidReqExt(bidRequest *openrtb2.BidRequest) error {
	undertoneBidderParams := &bidderParams{
		Id:      adapterId,
		Version: adapterVersion,
	}
	undertoneBidderParamsJSON, err := json.Marshal(undertoneBidderParams)
	if err == nil {
		extRequestPrebid := &openrtb_ext.ExtRequestPrebid{BidderParams: undertoneBidderParamsJSON}
		bidRequestExt := &BidRequestExt{Prebid: extRequestPrebid}
		bidRequestExtJSON, err2 := json.Marshal(bidRequestExt)
		if err2 == nil {
			bidRequest.Ext = bidRequestExtJSON
		}
	}
	return err
}

func populateSiteApp(bidRequest *openrtb2.BidRequest, publisherId int, site *openrtb2.Site, app *openrtb2.App) {
	pubId := strconv.Itoa(publisherId)
	if site != nil {
		siteCopy := *site
		var publisher openrtb2.Publisher
		if siteCopy.Publisher != nil {
			publisher = *siteCopy.Publisher
		}
		publisher.ID = pubId
		bidRequest.Site = &siteCopy
		bidRequest.Site.Publisher = &publisher
	} else if app != nil {
		appCopy := *app
		var publisher openrtb2.Publisher
		if appCopy.Publisher != nil {
			publisher = *appCopy.Publisher
		}
		publisher.ID = pubId
		bidRequest.App = &appCopy
		bidRequest.App.Publisher = &publisher
	}
}

func getImpsAndPublisherId(bidRequest *openrtb2.BidRequest) ([]openrtb2.Imp, int, []error) {
	var errs []error
	var publisherId int
	var validImps []openrtb2.Imp

	for _, imp := range bidRequest.Imp {
		var extImpBidder adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &extImpBidder); err != nil {
			errs = append(errs, getInvalidImpErr(imp.ID, err))
			continue
		}

		var extImpUndertone openrtb_ext.ExtImpUndertone
		if err := json.Unmarshal(extImpBidder.Bidder, &extImpUndertone); err != nil {
			errs = append(errs, getInvalidImpErr(imp.ID, err))
			continue
		}

		if publisherId == 0 {
			publisherId = extImpUndertone.PublisherID
		}

		imp.TagID = strconv.Itoa(extImpUndertone.PlacementID)
		imp.Ext = nil
		validImps = append(validImps, imp)
	}

	return validImps, publisherId, errs
}

func getInvalidImpErr(impId string, err error) *errortypes.BadInput {
	return &errortypes.BadInput{
		Message: "Invalid impid=" + impId + ": " + err.Error(),
	}
}

func (a *adapter) makeImps(bidRequest *openrtb2.BidRequest, ext *openrtb_ext.ExtImpUndertone) []openrtb2.Imp {
	var validImps []openrtb2.Imp

	for _, imp := range bidRequest.Imp {
		if imp.Banner != nil || imp.Video != nil {
			imp.TagID = strconv.Itoa(ext.PlacementID)
			imp.Ext = nil
			validImps = append(validImps, imp)
			break
		}
	}

	return validImps
}
