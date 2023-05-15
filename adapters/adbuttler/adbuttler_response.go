package adbuttler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdButlerBeacon struct {
	Type        string `json:"type,omitempty"`
	TrackingUrl string `json:"url,omitempty"`
}

type AdButlerBid struct {
	CPCBid      float64           `json:"cpc_bid,omitempty"`
	CPCSpend    float64           `json:"cpc_spend,omitempty"`
	CampaignID  string            `json:"campaign_id,omitempty"`
	ProductData map[string]string `json:"item,omitempty"`
	Beacons     []*AdButlerBeacon `json:"beacons,omitempty"`
}

type AdButlerResponse struct {
	Status string         `json:"status,omitempty"`
	Code   int32          `json:"code,omitempty"`
	Bids   []*AdButlerBid `json:"items,omitempty"`
}


func AddDefaultFields(bid *openrtb2.Bid){
	if bid != nil {
		bid.CrID = "DefaultCRID"
	}
}

func GetDefaultBidID(name string) string {
	prefix := "BidResponse_" + name+ "_"
	t := time.Now().UnixNano() / int64(time.Microsecond)
	return prefix + strconv.Itoa(int(t))
}

func (a *AdButtlerAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
    var errors []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from Adbutler.",
		}
		return nil, []error{err}
	}

	if response.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", response.StatusCode),
		}
		return nil, []error{err}
	}

	var adButlerResp AdButlerResponse
	if err := json.Unmarshal(response.Body, &adButlerResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	//Temporarily for Debugging
	u, _ := json.Marshal(adButlerResp)
	fmt.Println(string(u))

	impID := internalRequest.Imp[0].ID
	responseF := GetBidderResponse(&adButlerResp, impID)
	return responseF, errors

}

func GetBidderResponse(adButlerResp *AdButlerResponse, requestImpID string) (*adapters.BidderResponse){

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adButlerResp.Bids))

	for index, adButlerBid := range adButlerResp.Bids {
		bidID := GetDefaultBidID(SEAT_ADBUTLER) + "_" + strconv.Itoa(index)
		impID := requestImpID + "_" + strconv.Itoa(index)
		bidPrice := adButlerBid.CPCBid
		campaignID := adButlerBid.CampaignID
		productid := adButlerBid.ProductData[RESPONSE_PRODUCTID]
		clickPrice := adButlerBid.CPCSpend
		var impressionUrl string
		var clickUrl string
		for _, beacon := range adButlerBid.Beacons {
			switch beacon.Type {
			case BEACONTYPE_IMP:
				impressionUrl = beacon.TrackingUrl
			case BEACONTYPE_CLICK:
				clickUrl = beacon.TrackingUrl
			}
		}

		bidExt := &openrtb_ext.ExtBidCommerce{
			ProductId:  productid,
			ClickUrl:   clickUrl,
			ClickPrice: clickPrice,
		}

		bid := &openrtb2.Bid{
			ID:    bidID,
			ImpID: impID,
			Price: bidPrice,
			CID:   campaignID,
			IURL:  impressionUrl,
		}

		AddDefaultFields(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		typedbid := &adapters.TypedBid{
			Bid:  bid,
			Seat: openrtb_ext.BidderName(SEAT_ADBUTLER),
		}
		bidResponse.Bids = append(bidResponse.Bids, typedbid)
	} 
	return bidResponse
}
