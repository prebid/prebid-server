package adservertargeting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

const MaxKeyLength = 20

func processRequestTargetingData(adServerTargetingData *adServerTargetingData, targetingData map[string]string, bidImpId string) {
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
	seatExt json.RawMessage) []openrtb_ext.ExtBidderMessage {

	var warnings []openrtb_ext.ExtBidderMessage

	for _, respTargetingData := range adServerTargetingData.ResponseTargetingData {
		key := resolveKey(respTargetingData, bidderName)
		path := respTargetingData.Path

		var value string
		var err error
		pathSplit := strings.Split(path, pathDelimiter)

		switch pathSplit[0] {

		case "seatbid":
			switch pathSplit[1] {
			case "bid":
				value, err = getValueFromSeatBidBid(path, bidsHolder, bidderName, bid)
			case "ext":
				value, err = getValueFromExt(path, "seatbid.ext.", seatExt)
			}
		case "ext":
			value, err = getValueFromExt(path, "ext.", response.Ext)
		default:
			value, err = getValueFromResp(path, response)
		}

		if err != nil {
			message := fmt.Sprintf("%s for bidder: %s, bid id: %s", err.Error(), bidderName, bid.ID)
			warnings = append(warnings, createWarning(message))
		} else {
			targetingData[key] = value
		}
	}
	return warnings
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
	bidExtTargeting, err := jsonutil.Marshal(bidExtTargetingData)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error())) //nolint: ineffassign,staticcheck
		return nil
	}

	newExt, err := jsonpatch.MergePatch(bid.Ext, bidExtTargeting)
	if err != nil {
		warnings = append(warnings, createWarning(err.Error())) //nolint: ineffassign,staticcheck
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
	if truncateTargetAttribute != nil && *truncateTargetAttribute > 0 {
		maxLength = *truncateTargetAttribute
	}

	targetingDataTruncated := make(map[string]string)
	for key, value := range targetingData {
		newKey := openrtb_ext.TargetingKey(key).TruncateKey("", maxLength)
		targetingDataTruncated[newKey] = value
	}
	return targetingDataTruncated
}

func getValueFromSeatBidBid(path string, bidsHolder bidsCache, bidderName string, bid openrtb2.Bid) (string, error) {
	bidBytes, err := bidsHolder.GetBid(bidderName, bid.ID, bid)
	if err != nil {
		return "", err
	}

	bidSplit := strings.Split(path, "seatbid.bid.")
	value, err := splitAndGet(bidSplit[1], bidBytes, pathDelimiter)
	if err != nil {
		return "", err

	}
	return value, nil
}

func getValueFromExt(path, separator string, respExt json.RawMessage) (string, error) {
	//path points to resp.ext, means path starts with ext
	extSplit := strings.Split(path, separator)
	value, err := splitAndGet(extSplit[1], respExt, pathDelimiter)
	if err != nil {
		return "", err
	}

	return value, nil
}

// getValueFromResp optimizes retrieval of paths for a bid response (not a seat or ext)
func getValueFromResp(path string, response *openrtb2.BidResponse) (string, error) {
	value, err := getRespData(response, path)
	if err != nil {
		return "", err
	}
	return value, nil
}

func getRespData(bidderResp *openrtb2.BidResponse, field string) (string, error) {
	// this code should be modified if there are changes in OpenRtb format
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
		return "", fmt.Errorf("key not found for field in bid response: %s", field)
	}

}
