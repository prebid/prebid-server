package pubmatic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func getTargetingKeys(bidExt json.RawMessage, bidderName string) map[string]string {
	targets := map[string]string{}
	if bidExt != nil {
		bidExtMap := make(map[string]interface{})
		err := json.Unmarshal(bidExt, &bidExtMap)
		if err == nil && bidExtMap[buyId] != nil {
			targets[buyIdTargetingKey+bidderName] = string(bidExtMap[buyId].(string))
		}
	}
	return targets
}

func copySBExtToBidExt(sbExt json.RawMessage, bidExt json.RawMessage) json.RawMessage {
	if sbExt != nil {
		sbExtMap := getMapFromJSON(sbExt)
		bidExtMap := make(map[string]interface{})
		if bidExt != nil {
			bidExtMap = getMapFromJSON(bidExt)
		}
		if bidExtMap != nil && sbExtMap != nil {
			if sbExtMap[buyId] != nil && bidExtMap[buyId] == nil {
				bidExtMap[buyId] = sbExtMap[buyId]
			}
		}
		byteAra, _ := json.Marshal(bidExtMap)
		return json.RawMessage(byteAra)
	}
	return bidExt
}

func getMapFromJSON(ext json.RawMessage) map[string]interface{} {
	if ext != nil {
		extMap := make(map[string]interface{})
		err := json.Unmarshal(ext, &extMap)
		if err == nil {
			return extMap
		}
	}
	return nil
}

//populateFirstPartyDataImpAttributes will parse imp.ext.data and populate imp extMap
func populateFirstPartyDataImpAttributes(data json.RawMessage, extMap map[string]interface{}) {
	dataMap := getMapFromJSON(data)

	if dataMap == nil {
		return
	}

	populateAdUnitKey(dataMap, extMap)
	populateDctrKey(dataMap, extMap)
}

func populateAdUnitKey(dataMap, extMap map[string]interface{}) {
	if adserverObj := dataMap[AdServerKey]; adserverObj != nil {
		var adserverExt ExtAdServer
		bodyBytes, _ := json.Marshal(adserverObj)
		if err := json.Unmarshal(bodyBytes, &adserverExt); err == nil {

			//if aderver name is gam, then copy adslot to imp.ext.dfp_ad_unit_code
			if adserverExt.Name == AdServerGAM && adserverExt.AdSlot != "" {
				extMap[ImpExtAdUnitKey] = adserverExt.AdSlot
			}
		}
	}

	//imp.ext.dfp_ad_unit_code is not set, then check pbadslot in imp.ext.data
	if extMap[ImpExtAdUnitKey] == nil && dataMap[PBAdslotKey] != nil {
		extMap[ImpExtAdUnitKey] = dataMap[PBAdslotKey].(string)
	}
}

func populateDctrKey(dataMap, extMap map[string]interface{}) {
	//read key-val pairs from imp.ext.data and add it in dctr
	dctr := strings.Builder{}
	for key, val := range dataMap {

		//ignore 'pbaslot' and 'adserver' key as they are not targeting keys
		if key == PBAdslotKey || key == AdServerKey {
			continue
		}
		var valStr string
		switch typedValue := val.(type) {
		case string:
			valStr = getString(typedValue)

		case float64:
			//integer data
			if typedValue == float64(int(typedValue)) {
				valStr = strconv.Itoa(int(typedValue))
			} else {
				valStr = strconv.FormatFloat(typedValue, 'f', 2, 64)
			}

		case bool:
			valStr = strconv.FormatBool(typedValue)
		case []interface{}:
			if isStringArray(typedValue) {
				if valStrArr := getStringArray(typedValue); valStrArr != nil && len(valStrArr) > 0 {
					valStr = strings.Join(valStrArr[:], ",")
				}
			}
		}
		if valStr != "" {
			appendKeyValToDctr(&dctr, key, valStr)
		}
	}

	if dctrStr := dctr.String(); dctrStr != "" {
		//merge the dctr values if already present in extMap
		if extMap[dctrKeyName] != nil {
			extMap[dctrKeyName] = fmt.Sprintf("%s|%s", extMap[dctrKeyName], dctrStr)
		} else {
			extMap[dctrKeyName] = dctrStr
		}
	}
}

func appendKeyValToDctr(dctrStr *strings.Builder, key, val string) {
	if key == "" || val == "" {
		return
	}
	if dctrStr.String() != "" {
		dctrStr.WriteString("|")
	}
	dctrStr.WriteString(key)
	dctrStr.WriteString("=")
	dctrStr.WriteString(val)
}

func isStringArray(array []interface{}) bool {
	for _, val := range array {
		if _, ok := val.(string); !ok {
			return false
		}
	}
	return true
}

func getStringArray(val interface{}) []string {
	aInterface, ok := val.([]interface{})
	if !ok {
		return nil
	}
	aString := make([]string, len(aInterface))
	for i, v := range aInterface {
		if str, ok := v.(string); ok {
			aString[i] = str
		}
	}

	return aString
}

func getString(val interface{}) string {
	var result string
	if val != nil {
		result, ok := val.(string)
		if ok {
			return result
		}
	}
	return result
}
