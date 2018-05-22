package usersync

import (
	"fmt"
)

func NewBeachfrontSyncer(usersyncURL string, pId string) Usersyncer {
	// redirect_uri := fmt.Sprintf("%s/setuid?bidder=beachfront&uid=$UID", external)
	url := fmt.Sprintf("%s%s", usersyncURL, pId )

	return &syncer{
		familyName: "beachfront",
		syncInfo: &UsersyncInfo{
			URL:         url,
			Type:        "redirect",
			SupportCORS: true,
		},
	}
}
