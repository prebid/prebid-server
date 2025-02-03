package devicedetection

import (
	"github.com/tidwall/gjson"
)

type accountInfo struct {
	Id string
}

type accountInfoExtractor struct{}

func newAccountInfoExtractor() accountInfoExtractor {
	return accountInfoExtractor{}
}

// extract extracts the account information from the payload
// The account information is extracted from the publisher id or site publisher id
func (x accountInfoExtractor) extract(payload []byte) *accountInfo {
	if payload == nil {
		return nil
	}

	publisherResult := gjson.GetBytes(payload, "app.publisher.id")
	if publisherResult.Exists() {
		return &accountInfo{
			Id: publisherResult.String(),
		}
	}
	publisherResult = gjson.GetBytes(payload, "site.publisher.id")
	if publisherResult.Exists() {
		return &accountInfo{
			Id: publisherResult.String(),
		}
	}
	return nil
}
