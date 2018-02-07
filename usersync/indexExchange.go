package usersync

import (
	"github.com/prebid/prebid-server/pbs"
)

func NewIndexSyncer(userSyncURL string) Usersyncer {
	return &syncer{
		familyName: "adnxs",
		syncInfo: &pbs.UsersyncInfo{
			URL:         userSyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
