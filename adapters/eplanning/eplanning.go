package eplanning

import (
	"encoding/json"
	"net/http"

	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type EPlanningAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

type EPlanningRequest struct {
	id      string
	user    *openrtb.User
	device  *openrtb.Device
	adUnits []*EPlanningAdUnit
}

type EPlanningBid struct {
	Id     string  `json:"id"`
	Price  float64 `json:"price,omitempty"`
	Width  uint64  `json:"w,omitempty"`
	Height uint64  `json:"h,omitempty"`
	DealId string  `json:"dealid,omitempty"`
}

type EPlanningAdUnit struct {
	Id         string
	Bidfloor   float64
	Instl      int8
	SspSpaceId int `json:"ssp_espacio_id,omitempty"`
	Video      *openrtb.Video
	Banner     *openrtb.Banner
}

func (adapter *EPlanningAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	ePlanningRequest, errors := openRtbToEPlanningRequest(request)
	if len(ePlanningRequest.adUnits) == 0 {
		return nil, errors
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	requestData := adapters.RequestData{
		Method: "POST",
		Uri:    adapter.URI,
		Body:   reqJSON,
	}

	requests := []*adapters.RequestData{&requestData}

	return requests, errors
}

func openRtbToEPlanningRequest(request *openrtb.BidRequest) (*EPlanningRequest, []error) {

	adUnits := make([]*EPlanningAdUnit, 0, len(request.Imp))
	errors := make([]error, 0, len(request.Imp))
	for _, imp := range request.Imp {
		var params openrtb_ext.ExtImpEPlanning
		err := json.Unmarshal(imp.Ext, &params)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		ePlanningAdUnit := EPlanningAdUnit{
			Id:         imp.ID,
			Bidfloor:   imp.BidFloor,
			Instl:      imp.Instl,
			Video:      imp.Video,
			Banner:     imp.Banner,
			SspSpaceId: params.SspSpaceId,
		}
		adUnits = append(adUnits, &ePlanningAdUnit)
	}
	return &EPlanningRequest{
		adUnits: adUnits,
		user:    request.User,
		device:  request.Device,
	}, errors
}

func (adapter *EPlanningAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	ePlanningOutput, err := parseEPlanningBids(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	bids := toOpenRtbBids(ePlanningOutput, internalRequest)

	return bids, nil
}

func NewEPlanningBidder(client *http.Client, endpoint string) *EPlanningAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}

	return &EPlanningAdapter{
		http: adapter,
		URI:  endpoint,
	}
}

func parseEPlanningBids(response []byte) ([]*EPlanningBid, error) {
	var bids []*EPlanningBid
	if err := json.Unmarshal(response, &bids); err != nil {
		return nil, err
	}

	return bids, nil
}

func toOpenRtbBids(ePlanningBids []*EPlanningBid, r *openrtb.BidRequest) []*adapters.TypedBid {
	bids := make([]*adapters.TypedBid, 0, len(ePlanningBids))

	for i, bid := range ePlanningBids {
		if bid.Id != "" {
			openRtbBid := openrtb.Bid{
				ID:     bid.Id,
				ImpID:  r.Imp[i].ID,
				Price:  bid.Price,
				W:      bid.Width,
				H:      bid.Height,
				DealID: bid.DealId,
			}
			bids = append(bids, &adapters.TypedBid{Bid: &openRtbBid})
		}
	}
	return bids
}
