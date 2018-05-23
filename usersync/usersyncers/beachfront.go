package usersyncers

import (
	"fmt"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/config"
)

func NewBeachfrontSyncer(usersyncURL string, platformId string, hostCookie config.Cookie) usersync.Usersyncer {
	url := fmt.Sprintf("%s%s&yourmom=", usersyncURL, platformId, hostCookie.Value)

	return &syncer{
		familyName:          "beachfront",
		syncEndpointBuilder: resolveMacros(url),
		syncType:            SyncTypeRedirect,
	}
}
