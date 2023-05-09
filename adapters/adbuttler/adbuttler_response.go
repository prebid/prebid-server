package adbuttler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)


type AdButlerBeacon struct {
	Type           string     `json:"type,omitempty"`
	TrackingUrl    string     `json:"url,omitempty"`
}

type AdButlerBid struct {
	CPCBid         float64                `json:"cpc_bid,omitempty"`
	CPCSpend       float64                `json:"cpc_spend,omitempty"`
    CampaignID     string                 `json:"campaign_id,omitempty"`
	ProductData    map[string]string      `json:"item,omitempty"`
	Beacons        []*AdButlerBeacon      `json:"beacons,omitempty"`
}


type AdButlerResponse struct { 
	Status      string                `json:"status,omitempty"`
	Code        int32                 `json:"code,omitempty"`
	Bids       []*AdButlerBid         `json:"items,omitempty"`
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
	

	hostName := GetHostName(internalRequest)
	if len(hostName) == 0 {
		hostName = COMMERCE_DEFAULT_HOSTNAME
	}
	iurl := hostName
	curl := hostName
	purl := hostName
	requestCount := GetRequestSlotCount(internalRequest)
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt ExtImpCommerce
	json.Unmarshal(internalRequest.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(internalRequest.Imp[0].Ext, &commerceExt)
	customConfig := commerceExt.Bidder.CustomConfig
	Nobid := false
	for _, eachCustomConfig := range customConfig {
		if eachCustomConfig.Key == "no_bid"{
			//fff
			val := eachCustomConfig.Value
			if val == "true" {
				Nobid = true
			}

		}
	}
	impiD := internalRequest.Imp[0].ID
	
	if !Nobid {
		responseF := GetDummyBids(iurl, curl, purl, "adbuttler", requestCount, impiD)
		return responseF,nil
	}
	
	err := fmt.Errorf("No Bids available for the given request from adbuttler")
	errors = append(errors,err )
	return nil, errors
}
