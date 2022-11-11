package smaato

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/errortypes"
)

type nativeAd struct {
	Native json.RawMessage `json:"native"`
}

func extractAdmNative(adMarkup string) (string, error) {
	var nativeAd nativeAd
	if err := json.Unmarshal([]byte(adMarkup), &nativeAd); err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Invalid ad markup %s.", adMarkup),
		}
	}
	adm, err := json.Marshal(&nativeAd.Native)
	if err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Invalid ad markup %s.", adMarkup),
		}
	}
	return string(adm), nil
}
