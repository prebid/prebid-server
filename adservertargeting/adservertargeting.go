package adservertargeting

import (
	"encoding/json"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/url"
	"strings"
)

type DataSource string

const (
	SourceBidRequest  DataSource = "bidrequest"
	SourceStatic      DataSource = "static"
	SourceBidResponse DataSource = "bidresponse"
)

const (
	bidderMacro   = "{{BIDDER}}"
	pathDelimiter = "."
)

// RequestTargetingData struct to hold pre-processed ad server targeting keys and values
type RequestTargetingData struct {
	SingleVal             json.RawMessage
	TargetingValueByImpId map[string][]byte
}

type ResponseTargetingData struct {
	Key      string
	HasMacro bool
	Path     string
}

type adServerTargetingData struct {
	RequestTargetingData  map[string]RequestTargetingData
	ResponseTargetingData []ResponseTargetingData
}

func ProcessAdServerTargeting(
	reqWrapper *openrtb_ext.RequestWrapper,
	resolvedRequest json.RawMessage,
	response *openrtb2.BidResponse,
	queryParams url.Values,
	bidResponseExt *openrtb_ext.ExtBidResponse,
	truncateTargetAttribute *int) *openrtb2.BidResponse {

	adServerTargeting, warnings := CollectAdServerTargeting(reqWrapper, resolvedRequest, queryParams)
	response, warnings = ResolveAdServerTargeting(adServerTargeting, response, warnings, truncateTargetAttribute)

	if len(warnings) > 0 {
		bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral] = append(bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral], warnings...)
	}
	return response
}

func CollectAdServerTargeting(
	reqWrapper *openrtb_ext.RequestWrapper, resolvedRequest json.RawMessage,
	queryParams url.Values) (*adServerTargetingData, []openrtb_ext.ExtBidderMessage) {

	var warnings []openrtb_ext.ExtBidderMessage

	adServerTargeting, err := getAdServerTargeting(reqWrapper)
	if err != nil {
		warnings = append(warnings, createWarning("Unable to extract adServerTargetingFrom req"))
		return nil, warnings
	}
	if len(adServerTargeting) == 0 {
		return nil, warnings
	}

	requestTargetingData := map[string]RequestTargetingData{}
	responseTargetingData := []ResponseTargetingData{}

	dataHolder := reqImpCache{resolverReq: resolvedRequest}

	for _, targetingObj := range adServerTargeting {
		source := strings.ToLower(targetingObj.Source)
		switch DataSource(source) {
		case SourceBidRequest:
			//causes PBS to treat 'value' as a path to pull from the request object
			value, err := getValueFromBidRequest(&dataHolder, targetingObj.Value, queryParams)
			if err != nil {
				warnings = append(warnings, createWarning(err.Error()))
				continue
			}
			requestTargetingData[targetingObj.Key] = value
		case SourceStatic:
			// causes PBS to just use the 'value' provided
			staticValue := RequestTargetingData{SingleVal: json.RawMessage(targetingObj.Value)}
			requestTargetingData[targetingObj.Key] = staticValue
		case SourceBidResponse:
			//causes PBS to treat 'value' as a path to pull from the bidder's response object, specifically seatbid[j].bid[k]
			bidResponseTargeting := ResponseTargetingData{}
			bidResponseTargeting.Key = targetingObj.Key
			bidResponseTargeting.Path = targetingObj.Value
			bidResponseTargeting.HasMacro = strings.Contains(strings.ToUpper(targetingObj.Key), bidderMacro)
			responseTargetingData = append(responseTargetingData, bidResponseTargeting)
		}
	}

	adServerTargetingData := &adServerTargetingData{
		RequestTargetingData:  requestTargetingData,
		ResponseTargetingData: responseTargetingData,
	}

	return adServerTargetingData, warnings
}

func ResolveAdServerTargeting(
	adServerTargetingData *adServerTargetingData,
	response *openrtb2.BidResponse,
	warnings []openrtb_ext.ExtBidderMessage,
	truncateTargetAttribute *int) (*openrtb2.BidResponse, []openrtb_ext.ExtBidderMessage) {
	if adServerTargetingData == nil {
		return response, nil
	}

	// ad server targeting data will go to seatbid[].bid[].ext.prebid.targeting
	//TODO: truncate keys

	bidsHolder := bidsCache{bids: make(map[string]map[string][]byte)}

	for _, seat := range response.SeatBid {
		bidderName := seat.Seat
		for i, bid := range seat.Bid {
			targetingData := make(map[string]string, 0)
			processRequestTargetingData(adServerTargetingData, targetingData, bid.ImpID)
			processResponseTargetingData(adServerTargetingData, targetingData, bidderName, bid, bidsHolder, response, seat.Ext, warnings)
			seat.Bid[i].Ext = buildBidExt(targetingData, bid, warnings, truncateTargetAttribute)
		}
	}
	return response, warnings
}
