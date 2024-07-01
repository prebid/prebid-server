package device_detection

import (
	"github.com/tidwall/gjson"
)

type AccountInfo struct {
	Id string
}

type AccountInfoExtractor struct{}

func NewAccountInfoExtractor() *AccountInfoExtractor {
	return &AccountInfoExtractor{}
}

// Extract extracts the account information from the payload
// The account information is extracted from the publisher id or site publisher id
func (x AccountInfoExtractor) Extract(payload []byte) *AccountInfo {
	if payload == nil {
		return nil
	}

	publisherResult := gjson.GetBytes(payload, "app.publisher.id")
	if !publisherResult.Exists() {
		publisherResult = gjson.GetBytes(payload, "site.publisher.id")
		if !publisherResult.Exists() {
			return nil
		}

		return &AccountInfo{
			Id: publisherResult.String(),
		}
	}

	return &AccountInfo{
		Id: publisherResult.String(),
	}
}
