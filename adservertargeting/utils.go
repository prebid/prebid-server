package adservertargeting

import (
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func splitAndGet(path string, data []byte, delimiter string) (string, error) {
	keySplit := strings.Split(path, delimiter)

	value, err := typedLookup(data, path, keySplit...)
	if err != nil {
		return "", err
	}

	return string(value), nil
}

func createWarning(message string) openrtb_ext.ExtBidderMessage {
	return openrtb_ext.ExtBidderMessage{
		Code:    errortypes.AdServerTargetingWarningCode,
		Message: message,
	}
}

func verifyPrefixAndTrim(path, prefix string) (string, bool) {
	ampDataSplit := strings.Split(path, prefix)
	if len(ampDataSplit) == 2 && ampDataSplit[0] == "" {
		return ampDataSplit[1], true
	}
	return "", false
}

func typedLookup(data []byte, path string, keys ...string) ([]byte, error) {
	value, dataType, _, err := jsonparser.Get(data, keys...)
	if err != nil && err != jsonparser.KeyPathNotFoundError {
		return nil, err
	} else if err != nil && err == jsonparser.KeyPathNotFoundError {
		return nil, fmt.Errorf("value not found for path: %s", path)
	}
	if verifyType(dataType) {
		return value, nil
	}
	return nil, fmt.Errorf("incorrect value type for path: %s, value can only be string or number", path)
}

func verifyType(dataType jsonparser.ValueType) bool {
	typeAllowed := false
	for _, allowedType := range allowedTypes {
		if dataType == allowedType {
			return true
		}
	}
	return typeAllowed
}
