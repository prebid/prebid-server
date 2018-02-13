package usersync

import (
	"github.com/prebid/prebid-server/pbs"
)

func NewIndexSyncer(userSyncURL string) Usersyncer {
	return &syncer{
		familyName: "indexExchange",
		syncInfo: &pbs.UsersyncInfo{
			URL:         userSyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
