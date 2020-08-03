package clients

import (
	"net/http"
)

var defaultHttpInstance = http.DefaultClient

func GetDefaultHttpInstance() *http.Client {
	// TODO 2020-06-22 @see https://github.com/prebid/prebid-server/pull/1331#discussion_r436110097
	return defaultHttpInstance
}
