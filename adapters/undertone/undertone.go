package undertone

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

const adapterId = 4
const adapterVersion = "1.0.0"

type adapter struct {
	endpoint string
}

type undertoneParams struct {
	Id      int    `json:"id"`
	Version string `json:"version"`
}

type impExt struct {
	Bidder *openrtb_ext.ExtImpUndertone `json:"bidder,omitempty"`
	Gpid   string                       `json:"gpid,omitempty"`
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
		return nil, append(errs, err)
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(reqCopy.Imp),
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
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
	}
	return bidResponse, nil
}

func populateBidReqExt(bidRequest *openrtb2.BidRequest) error {
	undertoneParams := &undertoneParams{
		Id:      adapterId,
		Version: adapterVersion,
	}

	undertoneParamsJSON, err := json.Marshal(undertoneParams)
	if err != nil {
		return err
	}

	bidRequest.Ext = undertoneParamsJSON
	return nil
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
		var ext impExt
		if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
			errs = append(errs, getInvalidImpErr(imp.ID, err))
			continue
		}

		if publisherId == 0 {
			publisherId = ext.Bidder.PublisherID
		}

		imp.TagID = strconv.Itoa(ext.Bidder.PlacementID)
		imp.Ext = nil

		if ext.Gpid != "" {
			ext.Bidder = nil
			impExtJson, err := json.Marshal(&ext)
			if err != nil {
				errs = append(errs, getInvalidImpErr(imp.ID, err))
				continue
			}
			imp.Ext = impExtJson
		}

		validImps = append(validImps, imp)
	}

	return validImps, publisherId, errs
}

func getInvalidImpErr(impId string, err error) *errortypes.BadInput {
	return &errortypes.BadInput{
		Message: "Invalid impid=" + impId + ": " + err.Error(),
	}
}
