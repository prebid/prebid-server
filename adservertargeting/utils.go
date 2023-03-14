package adservertargeting

import (
	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"strings"
)

func splitAndGet(path string, data []byte, delimiter string) (string, error) {
	keySplit := strings.Split(path, delimiter)
	value, _, _, err := jsonparser.Get(data, keySplit...)
	if err != nil {
		return "", errors.Errorf("value not found for path: %s", path)
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
