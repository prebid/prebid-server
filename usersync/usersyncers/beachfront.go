package usersyncers

import (
	"fmt"
	"github.com/prebid/prebid-server/usersync"
)

func NewBeachfrontSyncer(usersyncURL string, pId string) usersync.Usersyncer {
	// redirect_uri := fmt.Sprintf("%s/setuid?bidder=beachfront&uid=$UID", external)
	url := fmt.Sprintf("%s%s", usersyncURL, pId)

	return &syncer{
		familyName:          "beachfront",
		syncEndpointBuilder: resolveMacros(url),
		syncType:            SyncTypeRedirect,
	}
}
