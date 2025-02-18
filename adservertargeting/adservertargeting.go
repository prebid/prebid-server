package adservertargeting

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
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

var (
	allowedTypes = []jsonparser.ValueType{jsonparser.String, jsonparser.Number}
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

func Apply(
	reqWrapper *openrtb_ext.RequestWrapper,
	resolvedRequest json.RawMessage,
	response *openrtb2.BidResponse,
	queryParams url.Values,
	bidResponseExt *openrtb_ext.ExtBidResponse,
	truncateTargetAttribute *int) *openrtb2.BidResponse {

	adServerTargeting, warnings := collect(reqWrapper, resolvedRequest, queryParams)
	response, warnings = resolve(adServerTargeting, response, warnings, truncateTargetAttribute)

	if len(warnings) > 0 {
		bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral] = append(bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral], warnings...)
	}
	return response
}

// collect gathers targeting keys and values from request based on initial config
// and optimizes future key and value that should be collected from response
func collect(
	reqWrapper *openrtb_ext.RequestWrapper, resolvedRequest json.RawMessage,
	queryParams url.Values) (*adServerTargetingData, []openrtb_ext.ExtBidderMessage) {

	var warnings []openrtb_ext.ExtBidderMessage

	adServerTargeting, err := getAdServerTargeting(reqWrapper)
	if err != nil {
		warnings = append(warnings, createWarning("unable to extract adServerTargeting from request"))
		return nil, warnings
	}
	adServerTargeting, validationWarnings := validateAdServerTargeting(adServerTargeting)
	if len(validationWarnings) > 0 {
		warnings = append(warnings, validationWarnings...)
	}

	requestTargetingData := map[string]RequestTargetingData{}
	responseTargetingData := []ResponseTargetingData{}

	impsCache := requestCache{resolvedReq: resolvedRequest}

	for _, targetingObj := range adServerTargeting {
		source := strings.ToLower(targetingObj.Source)
		switch DataSource(source) {
		case SourceBidRequest:
			//causes PBS to treat 'value' as a path to pull from the request object
			value, err := getValueFromBidRequest(&impsCache, targetingObj.Value, queryParams)
			if err != nil {
				warnings = append(warnings, createWarning(err.Error()))
			} else {
				requestTargetingData[targetingObj.Key] = value
			}
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

func resolve(
	adServerTargetingData *adServerTargetingData,
	response *openrtb2.BidResponse,
	warnings []openrtb_ext.ExtBidderMessage,
	truncateTargetAttribute *int) (*openrtb2.BidResponse, []openrtb_ext.ExtBidderMessage) {

	bidCache := bidsCache{bids: make(map[string]map[string][]byte)}

	for _, seat := range response.SeatBid {
		bidderName := seat.Seat
		for i, bid := range seat.Bid {
			targetingData := make(map[string]string)
			processRequestTargetingData(adServerTargetingData, targetingData, bid.ImpID)
			respWarnings := processResponseTargetingData(adServerTargetingData, targetingData, bidderName, bid, bidCache, response, seat.Ext)
			if len(respWarnings) > 0 {
				warnings = append(warnings, respWarnings...)
			}
			seat.Bid[i].Ext = buildBidExt(targetingData, bid, warnings, truncateTargetAttribute)
		}
	}
	return response, warnings
}
