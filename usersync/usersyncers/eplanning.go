package usersyncers

import (
	"fmt"
	"net/url"
)

func NewEPlanningSyncer(usersyncURL string, externalURL string) *syncer {
	redirectUri := fmt.Sprintf("%s/setuid?bidder=eplanning&uid=$UID", externalURL)

	return &syncer{
		familyName:          "eplanning",
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirectUri))),
		syncType:            SyncTypeRedirect,
	}
}
