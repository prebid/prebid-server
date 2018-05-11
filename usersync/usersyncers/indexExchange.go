package usersyncers

import "github.com/prebid/prebid-server/usersync"

func NewIndexSyncer(userSyncURL string) *syncer {
	return &syncer{
		familyName:   "indexExchange",
		gdprVendorID: 10,
		syncInfo: &usersync.UsersyncInfo{
			URL:         userSyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
