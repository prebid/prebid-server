package usersyncers

import "github.com/prebid/prebid-server/usersync"

func NewIndexSyncer(userSyncURL string) *syncer {
	return &syncer{
		familyName: "indexExchange",
		syncInfo: &usersync.UsersyncInfo{
			URL:         userSyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
