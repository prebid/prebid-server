package adbuttler

import (
	"encoding/json"
	"fmt"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdButlerRequest struct { 
	Status      string                  `json:"search,omitempty"`
	SearchType        string                  `json:"search_type,omitempty"`
	Params            map[string][]string     `json:"params,omitempty"`
	Identifiers       []string                `json:"identifiers,omitempty"`
	Target            map[string]interface{}  `json:"_abdk_json,omitempty"`
	Limit             int                     `json:"limit,omitempty"`
	Source            int64                   `json:"source,omitempty"`
	UserID            string                  `json:"udb_uid,omitempty"`
	IP                string                  `json:"ip,omitempty"`
	UserAgent         string                  `json:"ua,omitempty"`
	Referrer          string                  `json:"referrer,omitempty"`
	FloorCPC          float64                 `json:"bid_floor_cpc,omitempty"`
}

func (a *AdButtlerAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errors []error 
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
		if *eachCustomConfig.Key == "no_bid"{
			//fff
			val := *eachCustomConfig.Value
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
