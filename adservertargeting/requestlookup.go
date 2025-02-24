package adservertargeting

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func getAdServerTargeting(reqWrapper *openrtb_ext.RequestWrapper) ([]openrtb_ext.AdServerTarget, error) {
	reqExt, err := reqWrapper.GetRequestExt()
	if err != nil {
		return nil, err
	}

	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid == nil {
		return nil, nil
	}

	return reqExtPrebid.AdServerTargeting, nil
}

func validateAdServerTargeting(adServerTargeting []openrtb_ext.AdServerTarget) ([]openrtb_ext.AdServerTarget, []openrtb_ext.ExtBidderMessage) {
	var validatedAdServerTargeting []openrtb_ext.AdServerTarget
	var warnings []openrtb_ext.ExtBidderMessage
	for i, targetingObj := range adServerTargeting {

		isDataCorrect := true

		if len(targetingObj.Key) == 0 {
			isDataCorrect = false
			warnings = append(warnings, createWarning(fmt.Sprintf("Key is empty for the ad server targeting object at index %d", i)))
		}

		if len(targetingObj.Value) == 0 {
			isDataCorrect = false
			warnings = append(warnings, createWarning(fmt.Sprintf("Value is empty for the ad server targeting object at index %d", i)))
		}

		targetingObjSource := DataSource(strings.ToLower(targetingObj.Source))
		if targetingObjSource != SourceStatic &&
			targetingObjSource != SourceBidRequest &&
			targetingObjSource != SourceBidResponse {
			isDataCorrect = false
			warnings = append(warnings, createWarning(fmt.Sprintf("Incorrect source for the ad server targeting object at index %d", i)))
		}

		if isDataCorrect {
			validatedAdServerTargeting = append(validatedAdServerTargeting, targetingObj)
		}

	}
	return validatedAdServerTargeting, warnings
}

func getValueFromBidRequest(dataHolder *requestCache, path string, queryParams url.Values) (RequestTargetingData, error) {
	//use the path specified in 'value' to look for data in the ortb bidrequest.
	res := RequestTargetingData{}

	//check if key points to query param from ext.prebid.amp.data
	ampDataValue, err := getValueFromQueryParam(path, queryParams)
	if ampDataValue != nil || err != nil {
		res.SingleVal = ampDataValue
		return res, err
	}

	// check if key points to imp data
	impData, err := getValueFromImp(path, dataHolder)
	if len(impData) > 0 || err != nil {
		res.TargetingValueByImpId = impData
		return res, err
	}

	// get data by key from request
	requestValue, err := getDataFromRequestJson(path, dataHolder)
	if requestValue != nil || err != nil {
		res.SingleVal = requestValue
		return res, err
	}

	return res, nil
}

func getValueFromQueryParam(path string, queryParams url.Values) (json.RawMessage, error) {
	ampDataSplit, hasPrefix := verifyPrefixAndTrim(path, "ext.prebid.amp.data.")
	if hasPrefix {
		val := queryParams.Get(ampDataSplit)
		if val != "" {
			return json.RawMessage(val), nil
		} else {
			return nil, fmt.Errorf("value not found for path: %s", path)
		}
	}
	return nil, nil
}

func getValueFromImp(path string, dataHolder *requestCache) (map[string][]byte, error) {
	impsDatas := make(map[string][]byte, 0)
	impSplit, hasPrefix := verifyPrefixAndTrim(path, "imp.")
	if hasPrefix {
		//If imp is specified in the path, the assumption is that the specific imp[] desired corresponds
		//to the seatbid[].bid[] we're working on. i.e. imp[].id=seatbid[].bid[].impid
		// key points to data in imp
		keySplit := strings.Split(impSplit, pathDelimiter)
		impsData, err := dataHolder.GetImpsData()
		if err != nil {
			return nil, err
		}
		for _, impData := range impsData {
			id, _, _, err := jsonparser.Get(impData, "id")
			if err != nil {
				return nil, err
			}
			value, err := typedLookup(impData, path, keySplit...)
			if err != nil {
				return nil, err
			}
			impsDatas[string(id)] = value
		}
	}
	return impsDatas, nil
}

func getDataFromRequestJson(path string, dataHolder *requestCache) (json.RawMessage, error) {
	keySplit := strings.Split(path, pathDelimiter)
	reqJson := dataHolder.GetReqJson()
	value, err := typedLookup(reqJson, path, keySplit...)

	if err != nil {
		return nil, err
	}
	return value, nil
}
