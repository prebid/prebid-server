package adbuttler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/koddi"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)


func (a *AdButtlerAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errors []error 
	hostName := koddi.GetHostName(internalRequest)
	if len(hostName) == 0 {
		hostName = koddi.COMMERCE_DEFAULT_HOSTNAME
	}
	iurl, _ := a.buildImpressionURL(hostName) 
	curl, _ := a.buildClickURL(hostName)
	purl, _ := a.buildConversionURL(hostName)
	requestCount := koddi.GetRequestSlotCount(internalRequest)
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt koddi.ExtImpCommerce
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
		responseF := koddi.GetDummyBids(iurl, curl, purl, "adbuttler", requestCount, impiD)
		return responseF,nil
	}
	
	err := fmt.Errorf("No Bids available for the given request from Koddi")
	errors = append(errors,err )
	return nil, errors
}
