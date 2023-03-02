package adservertargeting

import (
	"encoding/json"
	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"strings"
)

func getValueFromBidRequest(dataHolder *reqImpCache, path string, ampData map[string]string) (RequestTargetingData, error) {
	//use the path specified in 'value' to look for data in the ortb bidrequest.
	res := RequestTargetingData{}

	//check if key points to query param from ext.prebid.amp.data
	ampDataValue, err := getValueFromQueryParam(path, ampData)
	if ampDataValue != nil || err != nil {
		res.SingleVal = ampDataValue
		return res, err
	}

	// check if key points to imp data
	impData, err := getValueFromImp(path, dataHolder)
	if len(impData) > 0 || err != nil {
		res.ImpData = impData
		return res, err
	}

	// get data by key from request
	requestValue, err := getDataFromRequest(path, dataHolder)
	if requestValue != nil || err != nil {
		res.SingleVal = requestValue
		return res, err
	}

	return res, nil
}

func getValueFromQueryParam(path string, ampData map[string]string) (json.RawMessage, error) {
	ampDataSplit := strings.Split(path, "ext.prebid.amp.data.")
	if len(ampDataSplit) == 2 && ampDataSplit[0] == "" {
		val, exists := ampData[ampDataSplit[1]]
		if exists {
			return json.RawMessage(val), nil
		} else {
			return nil, errors.Errorf("value not found for path: %s", path)
		}
	}
	return nil, nil
}

func getValueFromImp(path string, dataHolder *reqImpCache) (map[string][]byte, error) {
	impSplit := strings.Split(path, "imp.")
	impsDatas := make(map[string][]byte, 0)
	if len(impSplit) == 2 && impSplit[0] == "" {

		//If imp is specified in the path, the assumption is that the specific imp[] desired corresponds
		//to the seatbid[].bid[] we're working on. i.e. imp[].id=seatbid[].bid[].impid

		// key points to data in imp
		keySplit := strings.Split(impSplit[1], pathDelimiter)
		impsData, err := dataHolder.GetImpsData()
		if err != nil {
			return nil, err
		}
		for _, impData := range impsData {
			id, _, _, err := jsonparser.Get(impData, "id")
			if err != nil {
				return nil, err
			}
			value, _, _, err := jsonparser.Get(impData, keySplit...)
			if err != nil && err != jsonparser.KeyPathNotFoundError {
				return nil, err
			} else if err != nil && err == jsonparser.KeyPathNotFoundError {
				return nil, errors.Errorf("value not found for path: %s", path)
			}
			impsDatas[string(id)] = value
		}
	}
	return impsDatas, nil
}

func getDataFromRequest(path string, dataHolder *reqImpCache) (json.RawMessage, error) {
	keySplit := strings.Split(path, pathDelimiter)
	reqJson := dataHolder.GetReqJson()
	value, _, _, err := jsonparser.Get(reqJson, keySplit...)
	if err != nil {
		return nil, errors.Errorf("value not found for path: %s", path)
	}
	return value, nil
}
