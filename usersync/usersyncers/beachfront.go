package usersyncers

import (
	"fmt"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewBeachfrontSyncer(usersyncURL string, platformId string) usersync.Usersyncer {
	url := fmt.Sprintf("%s%s", usersyncURL, platformId)

	return &syncer{
		familyName:          "beachfront",
		syncEndpointBuilder: resolveMacros(url),
		syncType:            SyncTypeRedirect,
	}
}
