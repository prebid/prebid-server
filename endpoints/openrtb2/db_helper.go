package openrtb2

import (
	"encoding/json"
	"fmt"

	"github.com/buger/jsonparser"
)

func isDbFetched(requestJson []byte) bool {
	val, dataType, _, err := jsonparser.Get(requestJson, "ext", "db_fetched")
	if err != nil {
		return false
	}
	if dataType == jsonparser.Boolean && string(val) == "true" {
		return true
	}
	return false
}

func parseDbStoredMaps(requestJson []byte) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	var errs []error
	storedReqMap := make(map[string]json.RawMessage)
	storedImpMap := make(map[string]json.RawMessage)

	rawReqs, dt, _, err := jsonparser.Get(requestJson, "ext", "db_storedrequests")
	if err == nil && dt == jsonparser.Object {
		if e := json.Unmarshal(rawReqs, &storedReqMap); e != nil {
			errs = append(errs, fmt.Errorf("failed to unmarshal db_storedrequests: %v", e))
		}
	}

	rawImps, dt2, _, err2 := jsonparser.Get(requestJson, "ext", "db_storedimps")
	if err2 == nil && dt2 == jsonparser.Object {
		if e := json.Unmarshal(rawImps, &storedImpMap); e != nil {
			errs = append(errs, fmt.Errorf("failed to unmarshal db_storedimps: %v", e))
		}
	}

	return storedReqMap, storedImpMap, errs
}
