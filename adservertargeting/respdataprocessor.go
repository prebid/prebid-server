package adservertargeting

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	"strings"
)

const MaxKeyLength = 20

func processRequestTargetingData(adServerTargetingData *adServerTargetingData, targetingData map[string]string, bidImpId string) {
	if len(adServerTargetingData.RequestTargetingData) == 0 {
		return
	}
	for key, val := range adServerTargetingData.RequestTargetingData {
		if len(val.SingleVal) > 0 {
			targetingData[key] = string(val.SingleVal)
		} else if len(val.TargetingValueByImpId) > 0 {
			targetingData[key] = string(val.TargetingValueByImpId[bidImpId])
		}
	}
}

func processResponseTargetingData(
	adServerTargetingData *adServerTargetingData,
	targetingData map[string]string,
	bidderName string,
	bid openrtb2.Bid,
	bidsHolder bidsCache,
	response *openrtb2.BidResponse,
	seatExt json.RawMessage,
	warnings []openrtb_ext.ExtBidderMessage) {
	if len(adServerTargetingData.ResponseTargetingData) == 0 {
		return
	}
	for _, respTargetingData := range adServerTargetingData.ResponseTargetingData {
		key := resolveKey(respTargetingData, bidderName)
		path := respTargetingData.Path

		pathSplit := strings.Split(path, pathDelimiter)

		switch pathSplit[0] {

		case "seatbid":
			switch pathSplit[1] {
			case "bid":
				getValueFromSeatBidBid(key, path, targetingData, bidsHolder, bidderName, bid, warnings)
			case "ext":
				getValueFromSeatBidExt(key, path, targetingData, seatExt, warnings)
			}

		case "ext":
			getValueFromRespExt(key, path, targetingData, response.Ext, warnings)
		default:
			// path points to value in response, not seat or ext
			// bidder response has limited number of properties
			// instead of unmarshal the whole response with seats and bids
			// try to find data by field name
			getValueFromResp(key, path, targetingData, response, warnings)

		}
	}
}

func buildBidExt(targetingData map[string]string,
	bid openrtb2.Bid,
	warnings []openrtb_ext.ExtBidderMessage,
	truncateTargetAttribute *int) json.RawMessage {

	targetingDataTruncated := truncateTargetingKeys(targetingData, truncateTargetAttribute)

	bidExtTargetingData := openrtb_ext.ExtBid{
		Prebid: &openrtb_ext.ExtBidPrebid{
			Targeting: targetingDataTruncated,
		},
	}
	bidExtTargeting, err := json.Marshal(bidExtTargetingData)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))
		return nil
	}

	newExt, err := jsonpatch.MergePatch(bid.Ext, bidExtTargeting)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))
		return nil
	}
	return newExt
}

func resolveKey(respTargetingData ResponseTargetingData, bidderName string) string {
	key := respTargetingData.Key
	if respTargetingData.HasMacro {
		key = strings.Replace(respTargetingData.Key, bidderMacro, bidderName, -1)
	}
	return key
}

func truncateTargetingKeys(targetingData map[string]string, truncateTargetAttribute *int) map[string]string {
	maxLength := MaxKeyLength
	if truncateTargetAttribute != nil {
		maxLength = *truncateTargetAttribute
		if maxLength < 0 {
			maxLength = MaxKeyLength
		}
	}

	targetingDataTruncated := make(map[string]string)
	for key, value := range targetingData {
		newKey := openrtb_ext.TargetingKey(key).TruncateKey(maxLength)
		targetingDataTruncated[newKey] = value
	}
	return targetingDataTruncated
}

func getValueFromSeatBidBid(
	key, path string,
	targetingData map[string]string,
	bidsHolder bidsCache,
	bidderName string,
	bid openrtb2.Bid,
	warnings []openrtb_ext.ExtBidderMessage) {
	bidBytes, err := bidsHolder.GetBid(bidderName, bid.ID, bid)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))

	}
	bidSplit := strings.Split(path, "seatbid.bid.")
	value, err := splitAndGet(bidSplit[1], bidBytes, pathDelimiter)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))

	}
	targetingData[key] = value
}

func getValueFromSeatBidExt(
	key, path string,
	targetingData map[string]string,
	seatExt json.RawMessage,
	warnings []openrtb_ext.ExtBidderMessage) {
	seatBidSplit := strings.Split(path, "seatbid.ext.")
	value, err := splitAndGet(seatBidSplit[1], seatExt, pathDelimiter)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))
	}

	targetingData[key] = value
}

func getValueFromRespExt(
	key, path string,
	targetingData map[string]string,
	respExt json.RawMessage,
	warnings []openrtb_ext.ExtBidderMessage) {
	//path points to resp.ext, means path starts with ext
	extSplit := strings.Split(path, "ext.")
	value, err := splitAndGet(extSplit[1], respExt, pathDelimiter)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))
	}
	targetingData[key] = value
}

func getValueFromResp(
	key, path string,
	targetingData map[string]string,
	response *openrtb2.BidResponse,
	warnings []openrtb_ext.ExtBidderMessage) {

	value, err := getRespData(response, path)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error()))
	}
	targetingData[key] = value
}

func getRespData(bidderResp *openrtb2.BidResponse, field string) (string, error) {
	// this code should be modified if there are changes in Op[enRtb format
	// it's up to date with OpenRTB 2.5 and 2.6
	switch field {
	case "id":
		return bidderResp.ID, nil
	case "bidid":
		return bidderResp.BidID, nil
	case "cur":
		return bidderResp.Cur, nil
	case "customdata":
		return bidderResp.CustomData, nil
	case "nbr":
		return fmt.Sprint(bidderResp.NBR.Val()), nil

	default:
		return "", errors.Errorf("key not found for field in bid response: %s", field)
	}

}
